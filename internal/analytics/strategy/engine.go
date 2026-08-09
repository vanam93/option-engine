package strategy

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

// Engine consumes SignalGenerated events and publishes StrategyDecision events.
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

// New creates a strategy engine subscribed to SignalGenerated events only.
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
		evaluator: NewEvaluator(cfg),
		cache:     NewCache(),
	}, nil
}

// Start subscribes to SignalGenerated before the consumer goroutine starts.
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
		return evt.Type == events.SignalGenerated
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
	sig, ok := parseInputSignal(evt.Payload)
	if !ok {
		return
	}
	decisions := e.evaluator.Process(sig, e.cache)
	e.publish(decisions)
}

func parseInputSignal(payload json.RawMessage) (InputSignal, bool) {
	var raw struct {
		Symbol     string    `json:"symbol"`
		Timeframe  string    `json:"timeframe"`
		Signal     string    `json:"signal"`
		Strategy   string    `json:"strategy"`
		Confidence float64   `json:"confidence"`
		Timestamp  time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputSignal{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" || raw.Strategy == "" {
		return InputSignal{}, false
	}
	return InputSignal{
		Symbol:     raw.Symbol,
		Timeframe:  raw.Timeframe,
		Signal:     raw.Signal,
		Strategy:   raw.Strategy,
		Confidence: raw.Confidence,
		Timestamp:  raw.Timestamp,
	}, true
}

func (e *Engine) publish(decisions []StrategyDecision) {
	for _, d := range decisions {
		out, err := events.NewEventWithClock(e.clk, events.StrategyDecision, engineName, d)
		if err != nil {
			continue
		}
		e.bus.Publish(out)
		e.health.record(d)
	}
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

	return e.health.report(e.cfg, connected, dropped)
}
