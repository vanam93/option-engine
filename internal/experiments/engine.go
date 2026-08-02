package experiments

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// Engine orchestrates parameter sweep experiments across backtest runs.
type Engine struct {
	cfg          Config
	bus          ports.EventBus
	clk          clock.Clock
	runner       BacktestRunner
	cache        *Cache
	experimentID string
	health       healthSnapshot

	scheduler *Scheduler
	workerWG  *sync.WaitGroup

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates an experiment engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock, runner BacktestRunner) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	experimentID := GenerateExperimentID()
	return &Engine{
		cfg:          cfg,
		bus:          bus,
		clk:          clk,
		runner:       runner,
		cache:        NewCache(experimentID),
		experimentID: experimentID,
		scheduler:    NewScheduler(cfg.ParallelWorkers, cfg.MaxConcurrentRuns),
	}, nil
}

// Start subscribes to optimization.updated and begins experiment execution.
func (e *Engine) Start(ctx context.Context) error {
	if !e.cfg.Enabled {
		return nil
	}

	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	engineCtx, cancel := context.WithCancel(ctx)
	e.ctx = engineCtx
	e.cancel = cancel
	e.subscription = e.bus.Subscribe(e.cfg.SubscriberBuffer, func(evt events.Event) bool {
		return evt.Type == events.OptimizationUpdated
	})
	e.started = true
	e.mu.Unlock()

	e.wg.Add(1)
	go e.consume(engineCtx)

	if e.runner != nil {
		runs := GenerateMatrix(e.cfg, e.experimentID)
		e.cache.RegisterRuns(runs)
		e.health.recordCreated(len(runs))

		e.workerWG = e.scheduler.Start(engineCtx, e.runner, e.onRunStart, e.onRunComplete)
		go func() {
			e.scheduler.Enqueue(runs)
		}()
	}

	return nil
}

func (e *Engine) consume(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			e.drain()
			return
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handleOptimization(evt)
		}
	}
}

func (e *Engine) drain() {
	for {
		select {
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handleOptimization(evt)
		default:
			return
		}
	}
}

func (e *Engine) handleOptimization(evt events.Event) {
	update, ok := parseOptimizationUpdate(evt.Payload)
	if !ok {
		return
	}
	runID := RunIDFromParameters(update.Parameters)
	if runID == "" {
		return
	}
	if !e.cache.IsPending(runID) {
		return
	}

	at := update.Timestamp
	if at.IsZero() {
		at = e.clk.Now()
	}
	result, ok := e.cache.StoreResult(runID, update.Score, update.Metrics, at)
	if !ok {
		return
	}
	e.publishCompleted(result)
}

func parseOptimizationUpdate(payload json.RawMessage) (optimization.OptimizationUpdated, bool) {
	var update optimization.OptimizationUpdated
	if err := json.Unmarshal(payload, &update); err != nil {
		return optimization.OptimizationUpdated{}, false
	}
	return update, true
}

func (e *Engine) onRunStart(run ExperimentRun) {
	e.cache.MarkRunning(run.RunID)
	e.health.recordStarted()
}

func (e *Engine) onRunComplete(run ExperimentRun, err error) {
	if err != nil {
		e.cache.MarkFailed(run.RunID)
		e.health.recordFailed()
		return
	}
	e.health.recordCompleted()
}

func (e *Engine) publishCompleted(result RunResult) {
	out, err := events.NewEventWithClock(e.clk, events.ExperimentCompleted, engineName, ExperimentCompleted{
		ExperimentID:      result.ExperimentID,
		RunID:             result.RunID,
		Strategy:          result.Strategy,
		Parameters:        result.Parameters,
		OptimizationScore: result.OptimizationScore,
		Rank:              result.Rank,
		Metrics:           result.Metrics,
		Timestamp:         result.CompletedAt,
	})
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// State returns an immutable snapshot of experiment state.
func (e *Engine) State() StateSnapshot {
	return e.cache.Snapshot()
}

// Close stops the engine and releases resources.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	cancel := e.cancel
	sub := e.subscription
	workerWG := e.workerWG
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if workerWG != nil {
		workerWG.Wait()
	}
	e.wg.Wait()
	if sub != nil {
		sub.Close()
	}
	return nil
}

// Health reports runtime status for observability probes.
func (e *Engine) Health() health.Report {
	dropped := uint64(0)
	e.mu.Lock()
	connected := e.started && !e.closed
	if e.subscription != nil {
		dropped = e.subscription.Dropped()
	}
	scheduler := e.scheduler
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, e.cache, scheduler)
}
