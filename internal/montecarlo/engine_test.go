package montecarlo_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/montecarlo"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/walkforward"
)

func testSeed() int64 {
	return 42
}

func sampleTrades() []float64 {
	return []float64{100, -50, 75, -25, 150, -30, 60}
}

func TestBootstrapResamplingSampleCount(t *testing.T) {
	seed := testSeed()
	cfg := montecarlo.Config{
		Enabled:         true,
		Simulations:     100,
		ConfidenceLevel: 0.95,
		RandomSeed:      &seed,
	}
	sim := montecarlo.NewSimulator(cfg)

	trades := sampleTrades()
	outcomes, err := sim.Run(trades)
	require.NoError(t, err)
	require.Len(t, outcomes, 100)
}

func TestConfidenceIntervalLowerMeanUpper(t *testing.T) {
	seed := testSeed()
	cfg := montecarlo.Config{
		Enabled:         true,
		Simulations:     500,
		ConfidenceLevel: 0.95,
		RandomSeed:      &seed,
	}
	sim := montecarlo.NewSimulator(cfg)

	trades := sampleTrades()
	outcomes, err := sim.Run(trades)
	require.NoError(t, err)

	result := sim.Summarize("mc-test", "wf-1", "exp-1", outcomes, montecarlo.StartingCapital(trades))
	ci := result.ConfidenceInterval

	require.Less(t, ci.Lower, ci.Mean)
	require.Less(t, ci.Mean, ci.Upper)
	require.InDelta(t, 0.95, ci.Level, 0.001)
}

func TestRiskOfRuinValidProbability(t *testing.T) {
	seed := testSeed()
	cfg := montecarlo.Config{
		Enabled:         true,
		Simulations:     200,
		ConfidenceLevel: 0.95,
		RandomSeed:      &seed,
	}
	sim := montecarlo.NewSimulator(cfg)

	trades := sampleTrades()
	outcomes, err := sim.Run(trades)
	require.NoError(t, err)

	result := sim.Summarize("mc-test", "wf-1", "exp-1", outcomes, montecarlo.StartingCapital(trades))
	require.GreaterOrEqual(t, result.RiskOfRuin, 0.0)
	require.LessOrEqual(t, result.RiskOfRuin, 1.0)
}

func TestEnginePublishesMonteCarloCompleted(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	seed := testSeed()

	engine, err := montecarlo.New(montecarlo.Config{
		Enabled:          true,
		Simulations:      50,
		ConfidenceLevel:  0.95,
		RandomSeed:       &seed,
		SubscriberBuffer: 16,
	}, bus, clk)
	require.NoError(t, err)

	completed := make(chan montecarlo.MonteCarloCompleted, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.MonteCarloCompleted
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var payload montecarlo.MonteCarloCompleted
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				continue
			}
			completed <- payload
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	payload, err := json.Marshal(walkforward.WalkForwardCompleted{
		WalkForwardID: "wf-test",
		ExperimentID:  "exp-test",
		RunID:         "run-test",
		PerformanceMetrics: optimization.EvaluationMetrics{
			TotalTrades:  10,
			NetPnL:       280,
			WinRate:      0.6,
			AverageTrade: 28,
			MaxDrawdown:  50,
		},
		Timestamp: clk.Now(),
	})
	require.NoError(t, err)

	bus.Publish(events.Event{
		Type:      events.WalkForwardCompleted,
		Source:    "walkforward_engine",
		Timestamp: clk.Now(),
		Payload:   payload,
	})

	select {
	case result := <-completed:
		require.NotEmpty(t, result.SimulationID)
		require.Equal(t, "wf-test", result.WalkForwardID)
		require.Equal(t, "exp-test", result.ExperimentID)
		require.Equal(t, 50, result.Simulations)
		require.GreaterOrEqual(t, result.ProbabilityOfProfit, 0.0)
		require.LessOrEqual(t, result.ProbabilityOfProfit, 1.0)
		require.NoError(t, engine.Close())
	case <-time.After(5 * time.Second):
		t.Fatal("expected montecarlo.completed event")
	}
}
