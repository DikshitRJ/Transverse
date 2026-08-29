// Package database provides PostgreSQL connection pooling and lifecycle management using pgxpool.
package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"transverse/internal/config"
)

// NewPostgresPool initializes a PostgreSQL connection pool using pgxpool,
// configures min/max connection limits from the application configuration,
// verifies connectivity via a ping with timeout, and returns the active pool.
func NewPostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database initialization failed: config cannot be nil")
	}

	return NewPoolFromURL(ctx, cfg.DatabaseURL, cfg.DBPoolMinConns, cfg.DBPoolMaxConns)
}

// NewPoolFromURL creates and validates a pgxpool.Pool given a connection string and pool bounds.
func NewPoolFromURL(ctx context.Context, databaseURL string, minConns, maxConns int) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration from URL: %w", err)
	}

	if minConns > 0 {
		poolConfig.MinConns = int32(minConns)
	}
	if maxConns > 0 {
		poolConfig.MaxConns = int32(maxConns)
	}

	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 15 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx connection pool: %w", err)
	}

	// Verify connectivity with a 5-second deadline
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	log.Printf("[database] successfully connected to postgres pool (min_conns=%d, max_conns=%d)", minConns, maxConns)
	return pool, nil
}
