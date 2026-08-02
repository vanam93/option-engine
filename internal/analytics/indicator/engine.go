package indicator

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine computes technical indicators from CandleClosed events.
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

// New creates an indicator engine subscribed to CandleClosed events only.
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
		cache: NewCache(cfg),
	}, nil
}

// Start subscribes to CandleClosed before the consumer goroutine starts.
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
		return evt.Type == events.CandleClosed
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
	var candle market.Candle
	if err := json.Unmarshal(evt.Payload, &candle); err != nil {
		return
	}
	e.health.recordProcessed(candle.CloseTime)

	values := e.cache.Update(candle)
	e.publish(values)
}

func (e *Engine) publish(values []domainindicator.IndicatorValue) {
	for _, value := range values {
		out, err := events.NewEventWithClock(e.clk, events.IndicatorUpdated, engineName, value)
		if err != nil {
			continue
		}
		e.bus.Publish(out)
	}
	e.health.recordPublished(len(values))
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

	active := 0
	stats := CacheStats{}
	if e.cache != nil {
		active = e.cache.ActiveSeries()
		stats = e.cache.Stats()
	}
	return e.health.report(e.cfg, connected, dropped, active, stats)
}
