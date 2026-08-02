package experiments_test

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
)

func sweepCfg() experiments.Config {
	return experiments.Config{
		Enabled:           true,
		ParallelWorkers:   2,
		MaxConcurrentRuns: 2,
		SubscriberBuffer:  16,
		Symbols:           []string{"NIFTY"},
		Timeframes:        []string{"5m"},
		Strategy:          "trend_following",
		ParameterRanges: experiments.ParameterRanges{
			EMAFast:   []int{5, 9},
			EMASlow:   []int{21},
			RSIPeriod: []int{14},
		},
	}
}

func TestParameterGenerationMatrix(t *testing.T) {
	cfg := sweepCfg()
	experimentID := experiments.GenerateExperimentID()
	runs := experiments.GenerateMatrix(cfg, experimentID)

	require.Equal(t, 2, len(runs))
	seen := make(map[string]struct{})
	for _, run := range runs {
		require.Equal(t, experimentID, run.ExperimentID)
		require.NotEmpty(t, run.RunID)
		require.Equal(t, "NIFTY", run.Symbol)
		require.Equal(t, "5m", run.Timeframe)
		_, dup := seen[run.RunID]
		require.False(t, dup, "duplicate run_id generated")
		seen[run.RunID] = struct{}{}
	}
}

type recordingRunner struct {
	mu     sync.Mutex
	calls  []string
	scores map[string]float64
	bus    *eventbus.Bus
	clk    clock.Clock
}

func (r *recordingRunner) Execute(ctx context.Context, run experiments.ExperimentRun) error {
	r.mu.Lock()
	r.calls = append(r.calls, run.RunID)
	r.mu.Unlock()

	score := 0.5
	if v, ok := r.scores[run.RunID]; ok {
		score = v
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
	r.bus.Publish(events.Event{
		Type:      events.OptimizationUpdated,
		Source:    "optimization_engine",
		Timestamp: r.clk.Now(),
		Payload:   payload,
	})
	return nil
}

func (r *recordingRunner) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func TestExperimentExecutionInvokesBacktest(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	cfg := sweepCfg()
	cfg.ParallelWorkers = 1
	cfg.MaxConcurrentRuns = 1

	runner := &recordingRunner{bus: bus, clk: clk, scores: map[string]float64{}}
	engine, err := experiments.New(cfg, bus, clk, runner)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	deadline := time.After(2 * time.Second)
	for runner.callCount() < 2 {
		select {
		case <-deadline:
			t.Fatalf("expected 2 backtest invocations, got %d", runner.callCount())
		case <-time.After(20 * time.Millisecond):
		}
	}

	require.NoError(t, engine.Close())
}

func TestRankingHighestScoreFirst(t *testing.T) {
	cache := experiments.NewCache(experiments.GenerateExperimentID())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	runs := []experiments.ExperimentRun{
		{ExperimentID: "exp-1", RunID: "run-low", Strategy: "alpha", Parameters: experiments.ParameterSet{"run_id": "run-low"}},
		{ExperimentID: "exp-1", RunID: "run-high", Strategy: "beta", Parameters: experiments.ParameterSet{"run_id": "run-high"}},
	}
	cache.RegisterRuns(runs)
	cache.MarkRunning("run-low")
	cache.MarkRunning("run-high")

	_, ok := cache.StoreResult("run-low", 0.35, optimization.EvaluationMetrics{TotalTrades: 5}, at)
	require.True(t, ok)
	high, ok := cache.StoreResult("run-high", 0.92, optimization.EvaluationMetrics{TotalTrades: 12}, at)
	require.True(t, ok)

	require.Equal(t, 1, high.Rank)
	require.Greater(t, high.OptimizationScore, 0.35)

	state := cache.Snapshot()
	require.Len(t, state.Rankings, 2)
	require.Equal(t, "run-high", state.Rankings[0].RunID)
	require.Equal(t, "run-low", state.Rankings[1].RunID)
	require.Equal(t, 1, state.Rankings[0].Rank)
	require.Equal(t, 2, state.Rankings[1].Rank)
}

func TestParallelSchedulingNoDuplicateExecution(t *testing.T) {
	scheduler := experiments.NewScheduler(2, 2)

	require.True(t, scheduler.TryStartForTest("run-1"))
	require.False(t, scheduler.TryStartForTest("run-1"))
	scheduler.FinishForTest("run-1")
	require.True(t, scheduler.TryStartForTest("run-1"))
}

func TestEnginePublishesExperimentCompleted(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	cfg := sweepCfg()
	cfg.ParameterRanges = experiments.ParameterRanges{EMAFast: []int{9}}
	cfg.ParallelWorkers = 1
	cfg.MaxConcurrentRuns = 1

	runner := &recordingRunner{
		bus:    bus,
		clk:    clk,
		scores: map[string]float64{},
	}
	engine, err := experiments.New(cfg, bus, clk, runner)
	require.NoError(t, err)

	completed := make(chan experiments.ExperimentCompleted, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ExperimentCompleted
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var payload experiments.ExperimentCompleted
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				continue
			}
			completed <- payload
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	select {
	case result := <-completed:
		require.NotEmpty(t, result.ExperimentID)
		require.NotEmpty(t, result.RunID)
		require.Greater(t, result.OptimizationScore, 0.0)
	case <-time.After(2 * time.Second):
		t.Fatal("expected experiment.completed event")
	}

	require.NoError(t, engine.Close())
}
