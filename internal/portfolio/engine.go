package portfolio

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

// Engine consumes ExecutionReport events and publishes portfolio.updated events.
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

// New creates a portfolio engine subscribed to ExecutionReport events only.
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

// Start subscribes to ExecutionReport before the consumer goroutine starts.
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
		return evt.Type == events.ExecutionReport
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
	report, ok := parseInputReport(evt.Payload)
	if !ok {
		return
	}
	if report.Status != statusFilled {
		return
	}
	result, applied := e.cache.Apply(report)
	if !applied {
		return
	}
	e.publish(result.Updated)
	e.health.record(result.Updated.Timestamp)
}

func parseInputReport(payload json.RawMessage) (InputReport, bool) {
	var raw struct {
		OrderID        string    `json:"order_id"`
		Symbol         string    `json:"symbol"`
		Timeframe      string    `json:"timeframe"`
		Action         string    `json:"action"`
		Quantity       int       `json:"quantity"`
		ExecutionPrice float64   `json:"execution_price"`
		Status         string    `json:"status"`
		Strategy       string    `json:"strategy"`
		Timestamp      time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputReport{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" || raw.Action == "" {
		return InputReport{}, false
	}
	return InputReport{
		OrderID:        raw.OrderID,
		Symbol:         raw.Symbol,
		Timeframe:      raw.Timeframe,
		Action:         raw.Action,
		Quantity:       raw.Quantity,
		ExecutionPrice: raw.ExecutionPrice,
		Status:         raw.Status,
		Strategy:       raw.Strategy,
		Timestamp:      raw.Timestamp,
	}, true
}

func (e *Engine) publish(update PortfolioUpdated) {
	out, err := events.NewEventWithClock(e.clk, events.PortfolioUpdated, engineName, update)
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// State returns an immutable snapshot of the current portfolio.
func (e *Engine) State() PortfolioState {
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
