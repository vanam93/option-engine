package montecarlo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/walkforward"
)

// Engine orchestrates Monte Carlo simulations on walk-forward validation results.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	simulator *Simulator
	cache     *Cache
	health    healthSnapshot
	workerWG  sync.WaitGroup

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
}

// New creates a Monte Carlo simulation engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock) (*Engine, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	return &Engine{
		cfg:       cfg,
		bus:       bus,
		clk:       clk,
		simulator: NewSimulator(cfg),
		cache:     NewCache(),
	}, nil
}

// Start subscribes to walkforward.completed and begins simulation processing.
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
		return evt.Type == events.WalkForwardCompleted
	})
	e.started = true
	e.mu.Unlock()

	e.workerWG.Add(1)
	go e.consume(engineCtx)

	return nil
}

func (e *Engine) consume(ctx context.Context) {
	defer e.workerWG.Done()
	for {
		select {
		case <-ctx.Done():
			e.drain()
			return
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handleWalkForwardCompleted(evt)
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
			e.handleWalkForwardCompleted(evt)
		default:
			return
		}
	}
}

func (e *Engine) handleWalkForwardCompleted(evt events.Event) {
	input, ok := parseWalkForwardCompleted(evt.Payload)
	if !ok {
		return
	}
	e.runSimulation(input)
}

func (e *Engine) runSimulation(input WalkForwardInput) {
	simulationID := GenerateSimulationID()
	e.cache.MarkActive(simulationID, input.WalkForwardID, input.ExperimentID)
	e.health.recordStarted()

	start := time.Now()
	trades := ExtractTradeReturns(input.Metrics)
	outcomes, err := e.simulator.Run(trades)
	if err != nil {
		e.cache.MarkFailed(simulationID)
		return
	}

	capital := StartingCapital(trades)
	result := e.simulator.Summarize(simulationID, input.WalkForwardID, input.ExperimentID, outcomes, capital)
	at := input.Timestamp
	if at.IsZero() {
		at = e.clk.Now()
	}
	result.CompletedAt = at

	e.cache.StoreResult(result)
	e.health.recordCompleted(time.Since(start))
	e.publishCompleted(result)
}

func (e *Engine) publishCompleted(result SimulationResult) {
	out, err := events.NewEventWithClock(e.clk, events.MonteCarloCompleted, engineName, MonteCarloCompleted{
		SimulationID:        result.SimulationID,
		WalkForwardID:       result.WalkForwardID,
		ExperimentID:        result.ExperimentID,
		Simulations:         result.Simulations,
		ConfidenceInterval:  result.ConfidenceInterval,
		ProbabilityOfProfit: result.ProbabilityOfProfit,
		ProbabilityOfLoss:   result.ProbabilityOfLoss,
		RiskOfRuin:          result.RiskOfRuin,
		DistributionSummary: result.DistributionSummary,
		Timestamp:           result.CompletedAt,
	})
	if err != nil {
		return
	}
	e.bus.Publish(out)
	e.health.recordReport()
}

// State returns an immutable snapshot of Monte Carlo state.
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
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.workerWG.Wait()
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
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, e.cache)
}

// GenerateSimulationID creates a new Monte Carlo batch identifier.
func GenerateSimulationID() string {
	return fmt.Sprintf("mc-%s", uuid.NewString())
}

func parseWalkForwardCompleted(payload json.RawMessage) (WalkForwardInput, bool) {
	var completed walkforward.WalkForwardCompleted
	if err := json.Unmarshal(payload, &completed); err != nil {
		return WalkForwardInput{}, false
	}
	return WalkForwardInput{
		WalkForwardID: completed.WalkForwardID,
		ExperimentID:  completed.ExperimentID,
		RunID:         completed.RunID,
		Metrics:       completed.PerformanceMetrics,
		Timestamp:     completed.Timestamp,
	}, true
}
