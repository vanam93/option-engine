package research_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/config"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

func testRepository(t *testing.T) *research.PostgresRepository {
	t.Helper()

	cfg := config.PostgresConfig{
		Host:     envOr("OE_POSTGRES_HOST", "localhost"),
		Port:     envIntOr("OE_POSTGRES_PORT", 5432),
		User:     envOr("OE_POSTGRES_USER", "option_engine"),
		Password: envOr("OE_POSTGRES_PASSWORD", "option_engine"),
		Database: envOr("OE_POSTGRES_DATABASE", "option_engine"),
		SSLMode:  envOr("OE_POSTGRES_SSL_MODE", "disable"),
		MaxConns: 4,
		MinConns: 1,
	}
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Skipf("postgres config: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("postgres ping failed: %v", err)
	}

	repo := research.NewPostgresRepository(pool)
	require.NoError(t, repo.EnsureSchema(context.Background()))
	t.Cleanup(pool.Close)
	return repo
}

func TestRepositoryInsertAndRead(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	experimentID := "exp-repo-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	exp := research.ResearchExperiment{
		ExperimentID: experimentID,
		Strategy:     "trend_following",
		Symbol:       "NIFTY",
		Timeframe:    "5m",
		Parameters:   []byte(`{"ema_fast":9}`),
		CreatedAt:    time.Now().UTC(),
	}
	require.NoError(t, repo.UpsertExperiment(ctx, exp))

	result := research.OptimizationResult{
		ExperimentID: experimentID,
		Score:        0.82,
		WinRate:      0.6,
		Expectancy:   12.5,
		ProfitFactor: 1.8,
		Drawdown:     0.12,
		Metrics:      []byte(`{"total_trades":10}`),
		CreatedAt:    time.Now().UTC(),
	}
	require.NoError(t, repo.InsertOptimizationResult(ctx, result))

	stored, err := repo.GetExperiment(ctx, experimentID)
	require.NoError(t, err)
	require.Equal(t, experimentID, stored.ExperimentID)
	require.Equal(t, "trend_following", stored.Strategy)
	require.Equal(t, "NIFTY", stored.Symbol)
	require.Equal(t, "5m", stored.Timeframe)

	bundle, err := repo.GetResearchBundle(ctx, experimentID)
	require.NoError(t, err)
	require.Len(t, bundle.Optimization, 1)
	require.InDelta(t, 0.82, bundle.Optimization[0].Score, 0.001)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}
