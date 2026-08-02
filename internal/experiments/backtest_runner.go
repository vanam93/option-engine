package experiments

import (
	"context"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/backtest"
)

// BacktestEngineRunner executes runs through the existing backtest replay engine.
type BacktestEngineRunner struct {
	factory func(run ExperimentRun) (*backtest.Engine, error)
}

// NewBacktestEngineRunner creates a runner that invokes backtest.Engine per run.
func NewBacktestEngineRunner(factory func(run ExperimentRun) (*backtest.Engine, error)) *BacktestEngineRunner {
	return &BacktestEngineRunner{factory: factory}
}

// Execute connects the replay provider and waits for replay completion.
func (r *BacktestEngineRunner) Execute(ctx context.Context, run ExperimentRun) error {
	if r.factory == nil {
		return ErrNilRunner
	}
	engine, err := r.factory(run)
	if err != nil {
		return err
	}
	defer func() { _ = engine.Close() }()

	provider := engine.Provider()
	if err := provider.Connect(ctx); err != nil {
		return err
	}
	defer func() { _ = provider.Disconnect(ctx) }()

	if err := provider.Subscribe(ctx, []string{run.Symbol}); err != nil {
		return err
	}

	for {
		status := engine.Status()
		if status == backtest.ReplayStatusCompleted || status == backtest.ReplayStatusStopped {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// SharedEngineRunner reuses a single backtest engine instance serially.
type SharedEngineRunner struct {
	mu     sync.Mutex
	engine *backtest.Engine
}

// NewSharedEngineRunner creates a runner around one backtest engine.
func NewSharedEngineRunner(engine *backtest.Engine) *SharedEngineRunner {
	return &SharedEngineRunner{engine: engine}
}

// Execute runs replay on the shared engine with exclusive access.
func (r *SharedEngineRunner) Execute(ctx context.Context, run ExperimentRun) error {
	if r.engine == nil {
		return ErrNilRunner
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	provider := r.engine.Provider()
	if err := provider.Connect(ctx); err != nil {
		return err
	}
	defer func() { _ = provider.Disconnect(ctx) }()

	if err := provider.Subscribe(ctx, []string{run.Symbol}); err != nil {
		return err
	}

	for {
		status := r.engine.Status()
		if status == backtest.ReplayStatusCompleted || status == backtest.ReplayStatusStopped {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}
