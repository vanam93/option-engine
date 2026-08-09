package performance

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine consumes portfolio.updated events and publishes performance.updated events.
type Engine struct {
	cfg    Config
	bus    ports.EventBus
	clk    clock.Clock
	cache  *Cache
	health healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a performance analytics engine subscribed to portfolio.updated events only.
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
		cfg:   cfg,
		bus:   bus,
		clk:   clk,
		cache: NewCache(),
	}, nil
}

// Start subscribes to portfolio.updated before the consumer goroutine starts.
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
		return evt.Type == events.PortfolioUpdated
	})
	e.started = true
	e.mu.Unlock()

	e.wg.Add(1)
	go e.run(engineCtx)
	return nil
}

func (e *Engine) run(ctx context.Context) {
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
			e.handle(evt)
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
			e.handle(evt)
		default:
			return
		}
	}
}

func (e *Engine) handle(evt events.Event) {
	update, ok := parseInputUpdate(evt.Payload)
	if !ok {
		return
	}
	result := e.cache.Apply(update)
	e.publish(result.Updated)
	e.health.record(result.Updated.Timestamp)
}

func parseInputUpdate(payload json.RawMessage) (InputUpdate, bool) {
	var raw struct {
		Symbol        string    `json:"symbol"`
		Position      *struct{} `json:"position"`
		RealizedPnL   float64   `json:"realized_pnl"`
		UnrealizedPnL float64   `json:"unrealized_pnl"`
		Timestamp     time.Time `json:"timestamp"`
		Strategy      string    `json:"strategy"`
		Timeframe     string    `json:"timeframe"`
		ParameterSet  string    `json:"parameter_set"`
		Parameters    string    `json:"parameters"`
		BacktestID    string    `json:"backtest_id"`
		ExperimentID  string    `json:"experiment_id"`
		RunID         string    `json:"run_id"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputUpdate{}, false
	}
	if raw.Symbol == "" {
		return InputUpdate{}, false
	}
	return InputUpdate{
		Symbol:        raw.Symbol,
		PositionOpen:  raw.Position != nil,
		RealizedPnL:   raw.RealizedPnL,
		UnrealizedPnL: raw.UnrealizedPnL,
		Timestamp:     raw.Timestamp,
		Context: ExperimentContext{
			Strategy:     raw.Strategy,
			Symbol:       raw.Symbol,
			Timeframe:    raw.Timeframe,
			ParameterSet: raw.ParameterSet,
			Parameters:   firstNonEmpty(raw.Parameters, raw.ParameterSet),
			BacktestID:   raw.BacktestID,
			ExperimentID: raw.ExperimentID,
			RunID:        raw.RunID,
		},
	}, true
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (e *Engine) publish(update PerformanceUpdated) {
	out, err := events.NewEventWithClock(e.clk, events.PerformanceUpdated, engineName, update)
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// State returns an immutable snapshot of current performance metrics.
func (e *Engine) State() PerformanceSnapshot {
	return e.cache.Snapshot()
}

// Close stops the engine and releases its subscription.
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
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, e.cache)
}
