package walkforward_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/experiments"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/walkforward"
)

func testDataRange() (time.Time, time.Time) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	return start, end
}

func TestWindowGenerationRollingWindows(t *testing.T) {
	start, end := testDataRange()
	windows := walkforward.GenerateWindows("wf-1", start, end, 30, 10, 10)

	require.NotEmpty(t, windows)
	require.Equal(t, 0, windows[0].Index)
	require.Equal(t, start, windows[0].TrainStart)
	require.Equal(t, start.AddDate(0, 0, 30), windows[0].TrainEnd)
	require.Equal(t, windows[0].TrainEnd, windows[0].ValidationStart)
	require.Equal(t, windows[0].ValidationStart.AddDate(0, 0, 10), windows[0].ValidationEnd)

	if len(windows) > 1 {
		require.Equal(t, start.AddDate(0, 0, 10), windows[1].TrainStart)
	}
}

func TestSelectBestParameterHighestScore(t *testing.T) {
	results := []experiments.RunResult{
		{RunID: "low", OptimizationScore: 0.35},
		{RunID: "high", OptimizationScore: 0.92},
		{RunID: "mid", OptimizationScore: 0.55},
	}
	best, ok := walkforward.SelectBest(results)
	require.True(t, ok)
	require.Equal(t, "high", best.RunID)
	require.Equal(t, 0.92, best.OptimizationScore)
}

func TestValidationAggregationSummaryMetrics(t *testing.T) {
	completed := []walkforward.WindowResult{
		{TrainingScore: 0.80, ValidationScore: 0.60, BestParameters: experiments.ParameterSet{"ema_fast": 9}},
		{TrainingScore: 0.70, ValidationScore: 0.50, BestParameters: experiments.ParameterSet{"ema_fast": 12}},
	}
	agg := walkforward.AggregateValidation(completed)

	require.Equal(t, 2, agg.WindowCount)
	require.InDelta(t, 0.55, agg.MeanValidationScore, 0.001)
	require.InDelta(t, 0.75, agg.MeanTrainingScore, 0.001)
	require.InDelta(t, 0.20, agg.TrainingValidationGap, 0.001)
	require.Greater(t, agg.StabilityScore, 0.0)
}

type wfRunner struct {
	mu     sync.Mutex
	bus    *eventbus.Bus
	clk    clock.Clock
	scores map[string]float64
	phase  map[string]string
	calls  int
}

func (r *wfRunner) Execute(ctx context.Context, run experiments.ExperimentRun) error {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()

	phase, _ := run.Parameters["phase"].(string)
	score := 0.5
	if v, ok := r.scores[run.RunID]; ok {
		score = v
	}
	if phase == "validation" {
		score = 0.65
	}

	params := experiments.SerializeParameters(run.Parameters)
	payload, err := json.Marshal(optimization.OptimizationUpdated{
		Strategy:   run.Strategy,
		Symbol:     run.Symbol,
		Timeframe:  run.Timeframe,
		Parameters: params,
		Score:      score,
		Rank:       1,
		Metrics: optimization.EvaluationMetrics{
			TotalTrades: 10,
			NetPnL:      100,
			WinRate:     0.6,
		},
		Timestamp: r.clk.Now(),
	})
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.phase[run.RunID] = phase
	r.mu.Unlock()

	r.bus.Publish(events.Event{
		Type:      events.OptimizationUpdated,
		Source:    "optimization_engine",
		Timestamp: r.clk.Now(),
		Payload:   payload,
	})
	return nil
}

func walkForwardCfg() walkforward.Config {
	start, end := testDataRange()
	return walkforward.Config{
		Enabled:              true,
		TrainWindowDays:      30,
		ValidationWindowDays: 10,
		StepDays:             30,
		DataStart:            start,
		DataEnd:              end,
		SubscriberBuffer:     16,
		ParallelWorkers:      1,
		MaxConcurrentRuns:    1,
		Experiments: experiments.Config{
			Enabled:           true,
			ParallelWorkers:   1,
			MaxConcurrentRuns: 1,
			Symbols:           []string{"NIFTY"},
			Timeframes:        []string{"5m"},
			Strategy:          "trend_following",
			ParameterRanges: experiments.ParameterRanges{
				EMAFast: []int{5, 9},
				EMASlow: []int{21},
			},
		},
	}
}

func TestEnginePublishesWalkForwardCompleted(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	cfg := walkForwardCfg()

	runner := &wfRunner{
		bus:    bus,
		clk:    clk,
		scores: map[string]float64{},
		phase:  map[string]string{},
	}

	engine, err := walkforward.New(cfg, bus, clk, runner)
	require.NoError(t, err)

	completed := make(chan walkforward.WalkForwardCompleted, 4)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.WalkForwardCompleted
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var payload walkforward.WalkForwardCompleted
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				continue
			}
			completed <- payload
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	deadline := time.After(10 * time.Second)
	for {
		select {
		case result := <-completed:
			require.NotEmpty(t, result.WalkForwardID)
			require.NotEmpty(t, result.RunID)
			require.NotEmpty(t, result.ExperimentID)
			require.Greater(t, result.TrainingScore, 0.0)
			require.Greater(t, result.ValidationScore, 0.0)
			require.NoError(t, engine.Close())
			return
		case <-deadline:
			state := engine.State()
			statuses := make([]string, 0, len(state.Windows))
			for _, w := range state.Windows {
				statuses = append(statuses, string(w.Status))
			}
			t.Fatalf("expected walkforward.completed event; windows=%d completed=%d statuses=%v runner_calls=%d",
				len(state.Windows), len(state.Completed), statuses, runner.calls)
		case <-time.After(50 * time.Millisecond):
		}
	}
}
