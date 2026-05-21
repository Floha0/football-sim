// Package postgres provides PostgreSQL database connectivity and migration support.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Floha0/football-sim/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect establishes a connection pool to PostgreSQL and verifies it
// with a Ping. If the database isn't reachable, the returned error
// surfaces immediately rather than at first query time.
func Connect(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	// Pool tuning
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connectivity
	// Use a bounded context so we don't hang forever at startup.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close() // <-- critical: don't leak the pool on ping failure
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}

// DSN returns the URL-form connection string used by golang-migrate.
// pgxpool accepts both URL and key=value forms; migrate requires URL form.
func DSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
}
