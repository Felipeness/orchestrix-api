package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"

	// Driving adapters (HTTP)
	httpAdapter "github.com/orchestrix/orchestrix-api/internal/adapter/driving/http"

	// Driven adapters (Infrastructure)
	"github.com/orchestrix/orchestrix-api/internal/adapter/driven/postgres"
	temporalAdapter "github.com/orchestrix/orchestrix-api/internal/adapter/driven/temporal"

	// Core services
	"github.com/orchestrix/orchestrix-api/internal/core/service"

	// Auth (middleware)
	"github.com/orchestrix/orchestrix-api/internal/auth"

	// Legacy handlers (to be migrated)
	"github.com/orchestrix/orchestrix-api/internal/alertrule"
	"github.com/orchestrix/orchestrix-api/internal/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/orchestrix?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// Temporal client
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	temporalClient, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		slog.Error("failed to connect to temporal", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()
	slog.Info("temporal connected", "host", temporalHost)

	// ============================================================================
	// DEPENDENCY INJECTION - Hexagonal Architecture
	// ============================================================================

	// Driven Adapters (Secondary/Infrastructure)
	tenantContextSetter := postgres.NewTenantContextSetter(pool)
	workflowRepo := postgres.NewWorkflowRepository(pool)
	executionRepo := postgres.NewExecutionRepository(pool)
	alertRepo := postgres.NewAlertRepository(pool)
	auditRepo := postgres.NewAuditRepository(pool)
	workflowExecutor := temporalAdapter.NewWorkflowExecutor(temporalClient)

	// Core Services (Application Layer)
	auditService := service.NewAuditService(auditRepo, tenantContextSetter)
	alertService := service.NewAlertService(alertRepo, auditService, tenantContextSetter)
	executionService := service.NewExecutionService(executionRepo, workflowExecutor, tenantContextSetter)
	workflowService := service.NewWorkflowService(
		workflowRepo,
		executionRepo,
		workflowExecutor,
		auditService,
		tenantContextSetter,
	)

	// Driving Adapters (Primary/HTTP)
	workflowHandler := httpAdapter.NewWorkflowHandler(workflowService)
	executionHandler := httpAdapter.NewExecutionHandler(executionService)
	alertHandler := httpAdapter.NewAlertHandler(alertService)
	auditHandler := httpAdapter.NewAuditHandler(auditService)

	// Legacy handlers (not yet migrated to hexagonal)
	alertRuleHandler := alertrule.NewHandler(pool)
	metricsHandler := metrics.NewHandler(pool)

	// ============================================================================
	// MIDDLEWARE
	// ============================================================================

	authMiddleware := auth.NewMiddleware(auth.Config{
		KeycloakURL: getEnv("KEYCLOAK_URL", "http://localhost:8180"),
		Realm:       getEnv("KEYCLOAK_REALM", "orchestrix"),
		ClientID:    getEnv("KEYCLOAK_CLIENT_ID", "orchestrix-api"),
		SkipPaths:   []string{"/health"},
	})

	tenantMiddleware := auth.NewTenantMiddleware(pool)

	// ============================================================================
	// ROUTER
	// ============================================================================

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health endpoints (no auth)
	r.Get("/health", healthHandler)
	r.Get("/health/live", livenessHandler)
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status": "not ready", "error": "database unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ready"}`))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public info
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message": "Orchestrix API v1", "status": "ok"}`))
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Handler)
			r.Use(tenantMiddleware.Handler)

			// Workflow routes (hexagonal)
			r.Mount("/workflows", workflowHandler.Routes())

			// Execution routes (hexagonal)
			r.Mount("/executions", executionHandler.Routes())

			// Alert routes (hexagonal)
			r.Mount("/alerts", alertHandler.Routes())

			// Audit log routes (hexagonal)
			r.Mount("/audit-logs", auditHandler.Routes())

			// Alert rule routes (legacy - to be migrated)
			r.Mount("/alert-rules", alertRuleHandler.Routes())

			// Metrics routes (legacy - to be migrated)
			r.Mount("/metrics", metricsHandler.Routes())
		})
	})

	// ============================================================================
	// SERVER
	// ============================================================================

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy"}`))
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "alive"}`))
}
