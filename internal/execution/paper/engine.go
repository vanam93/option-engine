package paper

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/execution"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

const intentApproved = "APPROVED"

// Engine consumes ApprovedTradeIntent events and publishes ExecutionReport events.
type Engine struct {
	cfg      Config
	bus      ports.EventBus
	clk      clock.Clock
	executor *Executor
	cache    *Cache
	health   healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a paper execution engine subscribed to ApprovedTradeIntent events only.
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
		cfg:      cfg,
		bus:      bus,
		clk:      clk,
		executor: NewExecutor(cfg),
		cache:    NewCache(),
	}, nil
}

// Start subscribes to ApprovedTradeIntent before the consumer goroutine starts.
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
		return evt.Type == events.ApprovedTradeIntent
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
	intent, ok := parseInputIntent(evt.Payload)
	if !ok {
		return
	}
	if intent.Status != intentApproved {
		return
	}
	report, err := e.Execute(e.ctx, intent)
	if err != nil {
		return
	}
	e.publish(report)
}

func parseInputIntent(payload json.RawMessage) (execution.ApprovedTradeIntent, bool) {
	var raw struct {
		ID             uuid.UUID `json:"id"`
		Symbol         string    `json:"symbol"`
		Timeframe      string    `json:"timeframe"`
		Status         string    `json:"status"`
		Action         string    `json:"action"`
		Quantity       int       `json:"quantity"`
		Strategy       string    `json:"strategy"`
		ReferencePrice float64   `json:"reference_price"`
		Timestamp      time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return execution.ApprovedTradeIntent{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" || raw.Action == "" {
		return execution.ApprovedTradeIntent{}, false
	}
	return execution.ApprovedTradeIntent{
		ID:             raw.ID,
		Symbol:         raw.Symbol,
		Timeframe:      raw.Timeframe,
		Status:         raw.Status,
		Action:         raw.Action,
		Quantity:       raw.Quantity,
		Strategy:       raw.Strategy,
		ReferencePrice: raw.ReferencePrice,
		Timestamp:      raw.Timestamp,
	}, true
}

func (e *Engine) publish(report ExecutionReport) {
	out, err := events.NewEventWithClock(e.clk, events.ExecutionReport, engineName, report)
	if err != nil {
		return
	}
	e.bus.Publish(out)
	e.health.record(report)
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

	return e.health.report(e.cfg, connected, dropped, e.cache.ActivePositions())
}
