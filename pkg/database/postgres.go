package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Pool settings
	poolConfig.MaxConns = cfg.MaxConns
	if poolConfig.MaxConns == 0 {
		poolConfig.MaxConns = 10
	}
	poolConfig.MinConns = cfg.MinConns
	if poolConfig.MinConns == 0 {
		poolConfig.MinConns = 2
	}
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	slog.Info("database connected",
		"host", cfg.Host,
		"database", cfg.Database,
		"max_conns", poolConfig.MaxConns,
	)

	return pool, nil
}

// SetTenantContext sets the current tenant for RLS
func SetTenantContext(ctx context.Context, pool *pgxpool.Pool, tenantID string) error {
	_, err := pool.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, false)", tenantID)
	return err
}
