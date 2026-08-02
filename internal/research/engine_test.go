package research_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/montecarlo"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

func TestEnginePublishesResearchUpdated(t *testing.T) {
	repo := testRepository(t)
	bus := eventbus.New()
	clk := clock.NewSystem()
	exportDir := t.TempDir()

	engine, err := research.New(research.Config{
		Enabled:          true,
		ExportDirectory:  exportDir,
		Formats:          []string{"json", "csv"},
		SubscriberBuffer: 16,
	}, bus, clk, repo)
	require.NoError(t, err)

	updated := make(chan research.ResearchUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ResearchUpdated
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var payload research.ResearchUpdated
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				continue
			}
			updated <- payload
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	experimentID := "exp-engine-" + time.Now().Format("150405.000")
	optPayload, err := json.Marshal(optimization.OptimizationUpdated{
		Strategy:   "trend_following",
		Symbol:     "NIFTY",
		Timeframe:  "5m",
		Parameters: `{"experiment_id":"` + experimentID + `"}`,
		Metrics: optimization.EvaluationMetrics{
			TotalTrades:  10,
			NetPnL:       200,
			WinRate:      0.6,
			Expectancy:   20,
			ProfitFactor: 1.5,
			MaxDrawdown:  0.1,
		},
		Score:     0.75,
		Timestamp: clk.Now(),
	})
	require.NoError(t, err)
	bus.Publish(events.Event{Type: events.OptimizationUpdated, Payload: optPayload, Timestamp: clk.Now()})

	mcPayload, err := json.Marshal(montecarlo.MonteCarloCompleted{
		SimulationID:  "mc-test-1",
		WalkForwardID: "wf-test-1",
		ExperimentID:  experimentID,
		Simulations:   100,
		ConfidenceInterval: montecarlo.ConfidenceInterval{
			Level: 0.95, Lower: 50, Upper: 300, Mean: 180, Median: 175,
		},
		ProbabilityOfProfit: 0.72,
		ProbabilityOfLoss:   0.28,
		RiskOfRuin:          0.05,
		DistributionSummary: montecarlo.DistributionSummary{MeanReturn: 180},
		Timestamp:           clk.Now(),
	})
	require.NoError(t, err)
	bus.Publish(events.Event{Type: events.MonteCarloCompleted, Payload: mcPayload, Timestamp: clk.Now()})

	select {
	case result := <-updated:
		require.NotEmpty(t, result.ResearchID)
		require.Equal(t, experimentID, result.ExperimentID)
		require.Equal(t, "trend_following", result.Strategy)
		require.NotEmpty(t, result.ReportLocation.JSONPath)
		require.NotEmpty(t, result.ReportLocation.CSVPath)
		require.FileExists(t, result.ReportLocation.JSONPath)
		require.FileExists(t, result.ReportLocation.CSVPath)
		require.Contains(t, filepath.Base(result.ReportLocation.JSONPath), experimentID)
		require.NoError(t, engine.Close())
	case <-time.After(5 * time.Second):
		t.Fatal("expected research.updated event")
	}
}
