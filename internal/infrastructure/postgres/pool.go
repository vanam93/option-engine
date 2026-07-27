package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/option-engine/option-engine/internal/infrastructure/config"
)

// Pool wraps a pgx connection pool with health checking.
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool creates and verifies a PostgreSQL connection pool.
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	))
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Pool{pool: pool}, nil
}

// Ping verifies database connectivity.
func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Check implements ports.HealthChecker.
func (p *Pool) Check(ctx context.Context) error {
	return p.Ping(ctx)
}

// Close shuts down the connection pool.
func (p *Pool) Close() {
	p.pool.Close()
}

// Underlying returns the raw pgx pool for repositories.
func (p *Pool) Underlying() *pgxpool.Pool {
	return p.pool
}
