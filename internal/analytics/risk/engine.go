package risk

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

// Engine consumes StrategyDecision events and publishes ApprovedTradeIntent events.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	evaluator *Evaluator
	cache     *Cache
	health    healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a risk engine subscribed to StrategyDecision events only.
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
	cache, err := NewCache(cfg.DayResetTimezone)
	if err != nil {
		return nil, err
	}
	return &Engine{
		cfg:       cfg,
		bus:       bus,
		clk:       clk,
		evaluator: NewEvaluator(cfg),
		cache:     cache,
	}, nil
}

// Start subscribes to StrategyDecision before the consumer goroutine starts.
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
		return evt.Type == events.StrategyDecision
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
	input, ok := parseInputDecision(evt.Payload)
	if !ok {
		return
	}
	intent := e.evaluator.Process(input, e.cache)
	e.publish(intent)
}

func parseInputDecision(payload json.RawMessage) (InputDecision, bool) {
	var raw struct {
		Symbol     string    `json:"symbol"`
		Timeframe  string    `json:"timeframe"`
		Decision   string    `json:"decision"`
		Strategy   string    `json:"strategy"`
		Confidence float64   `json:"confidence"`
		Timestamp  time.Time `json:"timestamp"`
		Reason     string    `json:"reason"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputDecision{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" || raw.Decision == "" {
		return InputDecision{}, false
	}
	return InputDecision{
		Symbol:     raw.Symbol,
		Timeframe:  raw.Timeframe,
		Decision:   raw.Decision,
		Strategy:   raw.Strategy,
		Confidence: raw.Confidence,
		Timestamp:  raw.Timestamp,
		Reason:     raw.Reason,
	}, true
}

func (e *Engine) publish(intent ApprovedTradeIntent) {
	out, err := events.NewEventWithClock(e.clk, events.ApprovedTradeIntent, engineName, intent)
	if err != nil {
		return
	}
	e.bus.Publish(out)
	e.health.record(intent)
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
	activePositions := 0
	e.mu.Lock()
	connected := e.started && !e.closed
	if e.subscription != nil {
		dropped = e.subscription.Dropped()
	}
	e.mu.Unlock()

	e.cache.mu.Lock()
	activePositions = e.cache.activePositions()
	e.cache.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, activePositions)
}
