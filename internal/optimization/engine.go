package optimization

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

// Engine consumes performance.updated events and publishes optimization.updated events.
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

// New creates an optimization engine subscribed to performance.updated events only.
func New(cfg Config, bus ports.EventBus, clk clock.Clock) (*Engine, error) {
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
	return &Engine{
		cfg:   cfg,
		bus:   bus,
		clk:   clk,
		cache: NewCache(),
	}, nil
}

// Start subscribes to performance.updated before the consumer goroutine starts.
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
		return evt.Type == events.PerformanceUpdated
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
	result := e.cache.Apply(update, e.cfg.Scoring)
	e.publish(result.Record)
	e.health.record(result.Record.UpdatedAt, len(result.Ranking) > 0)
}

func parseInputUpdate(payload json.RawMessage) (InputUpdate, bool) {
	var raw struct {
		Strategy      string    `json:"strategy"`
		Symbol        string    `json:"symbol"`
		Timeframe     string    `json:"timeframe"`
		Parameters    string    `json:"parameters"`
		TotalTrades   int       `json:"total_trades"`
		WinRate       float64   `json:"win_rate"`
		RealizedPnL   float64   `json:"realized_pnl"`
		UnrealizedPnL float64   `json:"unrealized_pnl"`
		Drawdown      float64   `json:"drawdown"`
		ProfitFactor  float64   `json:"profit_factor"`
		MaxDrawdown   float64   `json:"max_drawdown"`
		Timestamp     time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputUpdate{}, false
	}

	strategy := raw.Strategy
	if strategy == "" {
		strategy = "default"
	}
	symbol := raw.Symbol
	if symbol == "" {
		symbol = "portfolio"
	}
	timeframe := raw.Timeframe
	if timeframe == "" {
		timeframe = "1m"
	}

	return InputUpdate{
		Strategy:      strategy,
		Symbol:        symbol,
		Timeframe:     timeframe,
		Parameters:    raw.Parameters,
		TotalTrades:   raw.TotalTrades,
		WinRate:       raw.WinRate,
		RealizedPnL:   raw.RealizedPnL,
		UnrealizedPnL: raw.UnrealizedPnL,
		Drawdown:      raw.Drawdown,
		ProfitFactor:  raw.ProfitFactor,
		MaxDrawdown:   raw.MaxDrawdown,
		Timestamp:     raw.Timestamp,
	}, true
}

func (e *Engine) publish(record EvaluationRecord) {
	out, err := events.NewEventWithClock(e.clk, events.OptimizationUpdated, engineName, OptimizationUpdated{
		Strategy:   record.Key.Strategy,
		Symbol:     record.Key.Symbol,
		Timeframe:  record.Key.Timeframe,
		Parameters: record.Key.Parameters,
		Metrics:    record.Metrics,
		Score:      record.Score,
		Rank:       record.Rank,
		Timestamp:  record.UpdatedAt,
	})
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// State returns an immutable snapshot of current optimization state.
func (e *Engine) State() StateSnapshot {
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
