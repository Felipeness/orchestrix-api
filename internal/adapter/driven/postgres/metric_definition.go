package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// definitionDBFields holds converted fields for DB operations
type definitionDBFields struct {
	alertThreshold []byte
	aggregation    *string
	retentionDays  *int32
}

// convertDefinitionToDBFields converts domain fields to DB-compatible types
func convertDefinitionToDBFields(def *domain.MetricDefinition) (definitionDBFields, error) {
	var fields definitionDBFields

	if def.AlertThreshold != nil {
		var err error
		fields.alertThreshold, err = json.Marshal(def.AlertThreshold)
		if err != nil {
			return fields, err
		}
	}

	if def.Aggregation != "" {
		s := string(def.Aggregation)
		fields.aggregation = &s
	}

	if def.RetentionDays > 0 {
		rd := int32(def.RetentionDays)
		fields.retentionDays = &rd
	}

	return fields, nil
}

// MetricDefinitionRepository implements port.MetricDefinitionRepository
type MetricDefinitionRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewMetricDefinitionRepository creates a new metric definition repository
func NewMetricDefinitionRepository(pool *pgxpool.Pool) *MetricDefinitionRepository {
	return &MetricDefinitionRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// FindByName finds a metric definition by name
func (r *MetricDefinitionRepository) FindByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.MetricDefinition, error) {
	row, err := r.queries.GetMetricDefinition(ctx, db.GetMetricDefinitionParams{
		TenantID: tenantID,
		Name:     name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMetricDefinitionNotFound
		}
		return nil, err
	}
	return r.toDomain(row), nil
}

// FindByTenant finds metric definitions by tenant with pagination
func (r *MetricDefinitionRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.MetricDefinition, error) {
	rows, err := r.queries.ListMetricDefinitions(ctx, db.ListMetricDefinitionsParams{
		TenantID: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	definitions := make([]*domain.MetricDefinition, len(rows))
	for i, row := range rows {
		definitions[i] = r.toDomain(row)
	}
	return definitions, nil
}

// CountByTenant counts metric definitions for a tenant
func (r *MetricDefinitionRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.queries.CountMetricDefinitions(ctx, tenantID)
}

// Save saves a new metric definition
func (r *MetricDefinitionRepository) Save(ctx context.Context, def *domain.MetricDefinition) error {
	fields, err := convertDefinitionToDBFields(def)
	if err != nil {
		return err
	}

	_, err = r.queries.CreateMetricDefinition(ctx, db.CreateMetricDefinitionParams{
		TenantID:       def.TenantID,
		Name:           def.Name,
		DisplayName:    def.DisplayName,
		Description:    def.Description,
		Unit:           def.Unit,
		Type:           string(def.Type),
		Aggregation:    fields.aggregation,
		AlertThreshold: fields.alertThreshold,
		RetentionDays:  fields.retentionDays,
	})
	return err
}

// Update updates a metric definition
func (r *MetricDefinitionRepository) Update(ctx context.Context, def *domain.MetricDefinition) error {
	fields, err := convertDefinitionToDBFields(def)
	if err != nil {
		return err
	}

	_, err = r.queries.UpdateMetricDefinition(ctx, db.UpdateMetricDefinitionParams{
		TenantID:       def.TenantID,
		Name:           def.Name,
		DisplayName:    def.DisplayName,
		Description:    def.Description,
		Unit:           def.Unit,
		Type:           string(def.Type),
		Aggregation:    fields.aggregation,
		AlertThreshold: fields.alertThreshold,
		RetentionDays:  fields.retentionDays,
	})
	return err
}

// Delete deletes a metric definition
func (r *MetricDefinitionRepository) Delete(ctx context.Context, tenantID uuid.UUID, name string) error {
	return r.queries.DeleteMetricDefinition(ctx, db.DeleteMetricDefinitionParams{
		TenantID: tenantID,
		Name:     name,
	})
}

// toDomain converts a db.MetricDefinition to domain.MetricDefinition
func (r *MetricDefinitionRepository) toDomain(row db.MetricDefinition) *domain.MetricDefinition {
	def := &domain.MetricDefinition{
		ID:          row.ID,
		TenantID:    row.TenantID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Description: row.Description,
		Unit:        row.Unit,
		Type:        domain.MetricType(row.Type),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}

	if row.Aggregation != nil {
		def.Aggregation = domain.AggregationType(*row.Aggregation)
	}

	if row.RetentionDays != nil {
		def.RetentionDays = int(*row.RetentionDays)
	}

	if len(row.AlertThreshold) > 0 {
		var threshold domain.AlertThreshold
		if err := json.Unmarshal(row.AlertThreshold, &threshold); err != nil {
			slog.Warn("failed to unmarshal alert threshold", "definition_id", row.ID, "error", err)
		} else {
			def.AlertThreshold = &threshold
		}
	}

	return def
}
