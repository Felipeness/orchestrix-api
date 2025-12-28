package alertrule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/db"
	"github.com/orchestrix/orchestrix-api/pkg/temporal"
)

// Domain errors
var (
	ErrTenantMismatch = errors.New("workflow belongs to different tenant")
	ErrInvalidTemplate = errors.New("invalid template syntax")
)

// templateVarRegex matches ${var} syntax
var templateVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// MetricData represents a metric being evaluated
type MetricData struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]interface{} `json:"labels"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
}

// ThresholdCondition represents a metric threshold condition config
type ThresholdCondition struct {
	MetricName string  `json:"metric_name"`
	Operator   string  `json:"operator"` // gt, gte, lt, lte, eq, ne
	Threshold  float64 `json:"threshold"`
}

// AlertTemplateData holds data for alert template rendering (CUPID: Domain-based)
type AlertTemplateData struct {
	MetricName string                 `json:"metric_name"`
	Value      float64                `json:"value"`
	Threshold  float64                `json:"threshold"`
	Operator   string                 `json:"operator"`
	Labels     map[string]interface{} `json:"labels"`
	Source     string                 `json:"source"`
	Timestamp  string                 `json:"timestamp"`
	RuleName   string                 `json:"rule_name"`
	Severity   string                 `json:"severity"`
}

// ToMap converts AlertTemplateData to map for template rendering
func (d AlertTemplateData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"metric_name": d.MetricName,
		"value":       d.Value,
		"threshold":   d.Threshold,
		"operator":    d.Operator,
		"labels":      d.Labels,
		"source":      d.Source,
		"timestamp":   d.Timestamp,
		"rule_name":   d.RuleName,
		"severity":    d.Severity,
	}
}

// CompareFunc defines a comparison function type
type CompareFunc func(value, threshold float64) bool

// operators maps operator strings to comparison functions
var operators = map[string]CompareFunc{
	"gt":  func(v, t float64) bool { return v > t },
	"gte": func(v, t float64) bool { return v >= t },
	"lt":  func(v, t float64) bool { return v < t },
	"lte": func(v, t float64) bool { return v <= t },
	"eq":  func(v, t float64) bool { return v == t },
	"ne":  func(v, t float64) bool { return v != t },
}

// ValidOperators returns the list of valid operator names
func ValidOperators() []string {
	keys := make([]string, 0, len(operators))
	for k := range operators {
		keys = append(keys, k)
	}
	return keys
}

// Evaluator evaluates alert rules against metrics
type Evaluator struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewEvaluator creates a new alert rule evaluator
func NewEvaluator(pool *pgxpool.Pool) *Evaluator {
	return &Evaluator{
		queries: db.New(pool),
		pool:    pool,
	}
}

// EvaluateMetric checks a metric against all relevant alert rules
func (e *Evaluator) EvaluateMetric(ctx context.Context, tenantID uuid.UUID, metric MetricData) error {
	// Get all enabled rules for this metric type
	rules, err := e.queries.GetAlertRulesForMetric(ctx, db.GetAlertRulesForMetricParams{
		TenantID:        tenantID,
		ConditionConfig: []byte(metric.Name), // The SQL query extracts metric_name from JSONB
	})
	if err != nil {
		slog.Error("failed to get alert rules for metric", "error", err, "metric", metric.Name)
		return err
	}

	for _, rule := range rules {
		triggered, err := e.evaluateRule(ctx, rule, metric)
		if err != nil {
			slog.Error("failed to evaluate rule", "error", err, "rule_id", rule.ID)
			continue
		}

		if triggered {
			if err := e.triggerAlert(ctx, tenantID, rule, metric); err != nil {
				slog.Error("failed to trigger alert", "error", err, "rule_id", rule.ID)
			}
		}
	}

	return nil
}

// evaluateRule checks if a metric triggers a specific rule
func (e *Evaluator) evaluateRule(_ context.Context, rule db.AlertRule, metric MetricData) (bool, error) {
	var condition ThresholdCondition
	if err := json.Unmarshal(rule.ConditionConfig, &condition); err != nil {
		return false, err
	}

	// Early return: metric name must match
	if condition.MetricName != metric.Name {
		return false, nil
	}

	// Lookup operator function
	compareFn, ok := operators[condition.Operator]
	if !ok {
		slog.Warn("unknown operator", "operator", condition.Operator, "valid", ValidOperators())
		return false, nil
	}

	return compareFn(metric.Value, condition.Threshold), nil
}

// triggerAlert creates an alert and optionally triggers a workflow
func (e *Evaluator) triggerAlert(ctx context.Context, tenantID uuid.UUID, rule db.AlertRule, metric MetricData) error {
	// Parse condition for template data
	var condition ThresholdCondition
	_ = json.Unmarshal(rule.ConditionConfig, &condition)

	// Build typed template data (CUPID: Domain-based)
	tplData := AlertTemplateData{
		MetricName: metric.Name,
		Value:      metric.Value,
		Threshold:  condition.Threshold,
		Operator:   condition.Operator,
		Labels:     metric.Labels,
		Source:     metric.Source,
		Timestamp:  metric.Timestamp.Format(time.RFC3339),
		RuleName:   rule.Name,
		Severity:   rule.Severity,
	}
	templateData := tplData.ToMap()

	// Render alert title
	title, err := e.renderTemplate(rule.AlertTitleTemplate, templateData)
	if err != nil {
		title = rule.AlertTitleTemplate // Fallback to raw template
	}

	// Render alert message
	var message string
	if rule.AlertMessageTemplate != nil && *rule.AlertMessageTemplate != "" {
		message, _ = e.renderTemplate(*rule.AlertMessageTemplate, templateData)
	}

	// Create the alert
	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"metric":    metric,
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
		"condition": condition,
	})

	alert, err := e.queries.CreateAlert(ctx, db.CreateAlertParams{
		TenantID:          tenantID,
		Title:             title,
		Message:           stringPtr(message),
		Severity:          rule.Severity,
		Source:            stringPtr(metric.Source),
		Metadata:          metadataJSON,
		TriggeredByRuleID: pgtype.UUID{Bytes: rule.ID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create alert: %w", err)
	}

	slog.Info("alert created",
		"alert_id", alert.ID,
		"rule_id", rule.ID,
		"metric", metric.Name,
		"value", metric.Value,
	)

	// Update last triggered timestamp
	if err := e.queries.UpdateAlertRuleLastTriggered(ctx, rule.ID); err != nil {
		slog.Warn("failed to update last triggered", "error", err)
	}

	// Trigger workflow if configured
	if rule.TriggerWorkflowID.Valid {
		if err := e.triggerWorkflow(ctx, tenantID, rule, alert, metric, templateData); err != nil {
			slog.Error("failed to trigger workflow", "error", err, "workflow_id", rule.TriggerWorkflowID.Bytes)
		}
	}

	return nil
}

// triggerWorkflow starts a workflow execution for auto-remediation
func (e *Evaluator) triggerWorkflow(ctx context.Context, tenantID uuid.UUID, rule db.AlertRule, alert db.Alert, metric MetricData, templateData map[string]interface{}) error {
	workflowUUID := uuid.UUID(rule.TriggerWorkflowID.Bytes)

	// Get workflow details
	workflow, err := e.queries.GetWorkflow(ctx, workflowUUID)
	if err != nil {
		return fmt.Errorf("get workflow: %w", err)
	}

	// Verify tenant ownership (CUPID: Predictable - explicit error)
	if workflow.TenantID != tenantID {
		slog.Warn("workflow tenant mismatch", "workflow_id", workflowUUID, "expected", tenantID, "got", workflow.TenantID)
		return ErrTenantMismatch
	}

	// Build workflow input from template
	var workflowInput map[string]interface{}
	if len(rule.TriggerInputTemplate) > 0 {
		// Parse template
		var inputTemplate map[string]interface{}
		if err := json.Unmarshal(rule.TriggerInputTemplate, &inputTemplate); err == nil {
			workflowInput = e.renderInputTemplate(inputTemplate, templateData)
		}
	}

	if workflowInput == nil {
		workflowInput = map[string]interface{}{
			"alert_id":    alert.ID.String(),
			"rule_id":     rule.ID.String(),
			"metric_name": metric.Name,
			"value":       metric.Value,
			"labels":      metric.Labels,
			"source":      metric.Source,
		}
	}

	inputJSON, _ := json.Marshal(workflowInput)

	// Create execution record
	temporalWorkflowID := "alert-" + alert.ID.String()
	execution, err := e.queries.CreateExecution(ctx, db.CreateExecutionParams{
		TenantID:           tenantID,
		WorkflowID:         workflowUUID,
		TemporalWorkflowID: &temporalWorkflowID,
		Status:             "pending",
		Input:              inputJSON,
		TriggeredBy:        stringPtr("alert_rule:" + rule.ID.String()),
	})
	if err != nil {
		return err
	}

	// Update alert with workflow execution ID
	if err := e.queries.UpdateAlertTriggeredExecution(ctx, db.UpdateAlertTriggeredExecutionParams{
		ID:                           alert.ID,
		TriggeredWorkflowExecutionID: pgtype.UUID{Bytes: execution.ID, Valid: true},
	}); err != nil {
		slog.Warn("failed to update alert with execution id", "error", err)
	}

	// Check if workflow has a dynamic definition
	var definition map[string]interface{}
	if len(workflow.Definition) > 0 {
		json.Unmarshal(workflow.Definition, &definition)
	}

	var run temporal.WorkflowRun
	var startErr error
	if steps, ok := definition["steps"]; ok && steps != nil {
		// Dynamic workflow
		run, startErr = temporal.ExecuteWorkflow(
			ctx,
			temporalWorkflowID,
			"DynamicWorkflow",
			map[string]interface{}{
				"workflow_id":  workflowUUID.String(),
				"execution_id": execution.ID.String(),
				"definition":   workflow.Definition,
				"input":        workflowInput,
			},
		)
	} else {
		// Static workflow
		run, startErr = temporal.ExecuteWorkflow(
			ctx,
			temporalWorkflowID,
			"ProcessWorkflow",
			map[string]interface{}{
				"workflow_id":  workflowUUID.String(),
				"execution_id": execution.ID.String(),
				"input":        workflowInput,
			},
		)
	}

	if startErr != nil {
		e.queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
			ID:    execution.ID,
			Status: "failed",
			Error: stringPtr("failed to start workflow: " + startErr.Error()),
		})
		return startErr
	}

	// Update execution with run ID
	e.queries.UpdateExecutionRunID(ctx, db.UpdateExecutionRunIDParams{
		ID:            execution.ID,
		TemporalRunID: stringPtr(run.GetRunID()),
		Status:        "running",
		StartedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	slog.Info("workflow triggered for alert",
		"alert_id", alert.ID,
		"workflow_id", workflowUUID,
		"execution_id", execution.ID,
		"temporal_workflow_id", temporalWorkflowID,
	)

	return nil
}

// renderTemplate renders a Go template string with the given data
func (e *Evaluator) renderTemplate(tmplStr string, data map[string]interface{}) (string, error) {
	// Convert ${var} syntax to {{.var}} for Go templates
	tmplStr = convertTemplateDelimiters(tmplStr)

	tmpl, err := template.New("alert").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderInputTemplate recursively renders template values in a map
func (e *Evaluator) renderInputTemplate(input map[string]interface{}, data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range input {
		switch val := v.(type) {
		case string:
			rendered, err := e.renderTemplate(val, data)
			if err != nil {
				result[k] = val
			} else {
				result[k] = rendered
			}
		case map[string]interface{}:
			result[k] = e.renderInputTemplate(val, data)
		default:
			result[k] = v
		}
	}

	return result
}

// convertTemplateDelimiters converts ${var} to {{.var}} using regex (CUPID: Predictable)
func convertTemplateDelimiters(s string) string {
	return templateVarRegex.ReplaceAllString(s, "{{.$1}}")
}
