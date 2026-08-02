package recommendation

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

// Engine consumes opportunity.updated events and publishes recommendation.updated events.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	cache     *Cache
	builder   *Builder
	formatter *Formatter
	health    healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a recommendation engine subscribed to opportunity.updated events only.
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
	formatter := NewFormatter()
	return &Engine{
		cfg:       cfg,
		bus:       bus,
		clk:       clk,
		cache:     NewCache(),
		formatter: formatter,
		builder:   NewBuilder(cfg, formatter),
	}, nil
}

// Start subscribes to opportunity.updated before the consumer goroutine starts.
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
		return evt.Type == events.OpportunityUpdated
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
	input, ok := parseOpportunity(evt.Payload)
	if !ok {
		return
	}
	at := input.Timestamp
	if at.IsZero() {
		at = e.clk.Now().UTC()
	}

	rec := e.builder.Build(input, at)
	e.cache.Put(rec)
	e.publish(rec)
	e.health.record(rec)
}

func parseOpportunity(payload json.RawMessage) (InputOpportunity, bool) {
	var raw struct {
		Symbol         string             `json:"symbol"`
		Timeframe      string             `json:"timeframe"`
		Rank           int                `json:"rank"`
		Confidence     float64            `json:"confidence"`
		Classification string             `json:"classification"`
		Score          float64            `json:"score"`
		Components     map[string]float64 `json:"components"`
		Timestamp      time.Time          `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputOpportunity{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" {
		return InputOpportunity{}, false
	}
	return InputOpportunity{
		Symbol:         raw.Symbol,
		Timeframe:      raw.Timeframe,
		Rank:           raw.Rank,
		Confidence:     raw.Confidence,
		Classification: raw.Classification,
		Score:          raw.Score,
		Components:     raw.Components,
		Timestamp:      raw.Timestamp,
	}, true
}

func (e *Engine) publish(rec RecommendationUpdated) {
	out, err := events.NewEventWithClock(e.clk, events.RecommendationUpdated, engineName, rec)
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// Latest returns the most recent recommendation for a symbol and timeframe.
func (e *Engine) Latest(symbol, timeframe string) (RecommendationUpdated, bool) {
	return e.cache.Get(symbol, timeframe)
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
