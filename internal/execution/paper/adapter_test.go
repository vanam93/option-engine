package paper_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/execution"
	"github.com/vanam-gangireddy/option-engine/internal/execution/paper"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func TestPaperAdapterSatisfiesExecutionAdapter(t *testing.T) {
	bus := eventbus.New()
	engine, err := paper.New(baseCfg(), bus, clock.NewSystem())
	require.NoError(t, err)

	var adapter execution.ExecutionAdapter = engine
	require.NotNil(t, adapter)
	require.Equal(t, execution.CapabilityModeSimulated, adapter.Capabilities().Mode)
	require.True(t, adapter.Capabilities().Simulated)
}

func TestExecuteThroughAdapterProducesIdenticalReport(t *testing.T) {
	bus := eventbus.New()
	engine, err := paper.New(baseCfg(), bus, clock.NewSystem())
	require.NoError(t, err)

	exec := paper.NewExecutor(baseCfg())
	cache := paper.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	intent := approvedIntent("LONG_ENTRY", 100, at)

	direct := exec.Execute(intent, cache)

	adapterReport, err := engine.Execute(context.Background(), intent)
	require.NoError(t, err)
	require.Equal(t, direct.Status, adapterReport.Status)
	require.Equal(t, direct.Action, adapterReport.Action)
	require.Equal(t, direct.Symbol, adapterReport.Symbol)
	require.Equal(t, direct.Quantity, adapterReport.Quantity)
	require.InDelta(t, direct.ExecutionPrice, adapterReport.ExecutionPrice, 0.0001)
	require.NotEmpty(t, adapterReport.OrderID)
}

func TestAdapterHealthWorks(t *testing.T) {
	bus := eventbus.New()
	engine, err := paper.New(baseCfg(), bus, clock.NewSystem())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	report := engine.Health()
	require.Equal(t, "paper_execution_engine", report.Component)
	require.Equal(t, health.StatusHealthy, report.Status)
	require.True(t, report.Connected)

	require.NoError(t, engine.Stop(ctx))

	afterStop := engine.Health()
	require.False(t, afterStop.Connected)
}
