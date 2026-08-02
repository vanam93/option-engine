package backtestrunner

import (
	"context"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	providerapi "github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// EngineFactory creates a backtest replay engine for a session request.
type EngineFactory func(req SessionRequest) (*backtest.Engine, error)

// ProviderBinder wires a replay provider into the runtime market pipeline.
type ProviderBinder interface {
	BindProvider(provider providerapi.Provider) error
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string) error
}

// ReplayRunner executes historical replay for a session.
type ReplayRunner interface {
	Execute(ctx context.Context, req SessionRequest) (time.Duration, error)
}

// BacktestReplayRunner orchestrates replay through the existing backtest engine.
type BacktestReplayRunner struct {
	factory      EngineFactory
	binder       ProviderBinder
	pollInterval time.Duration
	mu           sync.Mutex
}

// NewReplayRunner creates a replay runner that reuses the backtest replay infrastructure.
func NewReplayRunner(factory EngineFactory, binder ProviderBinder) *BacktestReplayRunner {
	return &BacktestReplayRunner{
		factory:      factory,
		binder:       binder,
		pollInterval: 25 * time.Millisecond,
	}
}

// Execute connects replay, waits for completion, and returns replay duration.
func (r *BacktestReplayRunner) Execute(ctx context.Context, req SessionRequest) (time.Duration, error) {
	if r == nil || r.factory == nil {
		return 0, ErrNilRunner
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	engine, err := r.factory(req)
	if err != nil {
		return 0, err
	}
	if engine == nil {
		return 0, ErrNilRunner
	}

	ownsEngine := true
	if r.binder != nil {
		ownsEngine = false
	}

	if ownsEngine {
		defer func() { _ = engine.Close() }()
	}

	provider := engine.Provider()
	if provider == nil {
		return 0, ErrNilRunner
	}

	if r.binder != nil {
		if err := r.binder.BindProvider(provider); err != nil {
			return 0, err
		}
		if err := r.binder.Connect(ctx); err != nil {
			return 0, err
		}
		defer func() { _ = r.binder.Disconnect(ctx) }()
		if err := r.binder.Subscribe(ctx, req.Symbols); err != nil {
			return 0, err
		}
	} else {
		if err := provider.Connect(ctx); err != nil {
			return 0, err
		}
		defer func() { _ = provider.Disconnect(ctx) }()
		if err := provider.Subscribe(ctx, req.Symbols); err != nil {
			return 0, err
		}
	}

	started := time.Now()
	for {
		status := engine.Status()
		if status == backtest.ReplayStatusCompleted || status == backtest.ReplayStatusStopped {
			return time.Since(started), nil
		}
		select {
		case <-ctx.Done():
			return time.Since(started), ctx.Err()
		case <-time.After(r.pollInterval):
		}
	}
}
