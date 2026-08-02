package recommendationstate

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

// Engine consumes validated.recommendation events and publishes recommendation.state.updated events.
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

// New creates a recommendation state manager subscribed to validated.recommendation events only.
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
		cache: NewCache(cfg.MaxActive),
	}, nil
}

// Start subscribes to validated.recommendation before the consumer goroutine starts.
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
		return evt.Type == events.ValidatedRecommendation
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
	input, ok := parseValidated(evt.Payload)
	if !ok {
		return
	}

	at := e.clk.Now().UTC()
	if !input.ValidatedAt.IsZero() {
		at = input.ValidatedAt.UTC()
	}

	rec, latest, duplicateMerged, changed := e.cache.ApplyValidated(input, at)
	if !changed {
		return
	}

	e.publish(rec, latest)
	e.health.record(duplicateMerged)
}

func parseValidated(payload json.RawMessage) (InputValidated, bool) {
	var raw struct {
		Symbol           string    `json:"symbol"`
		Timeframe        string    `json:"timeframe"`
		Strategy         string    `json:"strategy"`
		Recommendation   string    `json:"recommendation"`
		Confidence       float64   `json:"confidence"`
		ValidationStatus string    `json:"validation_status"`
		RejectionReasons []string  `json:"rejection_reasons"`
		ValidatedAt      time.Time `json:"validated_at"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputValidated{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" {
		return InputValidated{}, false
	}

	strategy := strings.TrimSpace(raw.Strategy)
	if strategy == "" {
		strategy = "default"
	}

	return InputValidated{
		Symbol:           raw.Symbol,
		Timeframe:        raw.Timeframe,
		Strategy:         strategy,
		Recommendation:   recommendation.Level(strings.ToUpper(strings.TrimSpace(raw.Recommendation))),
		Confidence:       raw.Confidence,
		ValidationStatus: strings.ToUpper(strings.TrimSpace(raw.ValidationStatus)),
		RejectionReasons: raw.RejectionReasons,
		ValidatedAt:      raw.ValidatedAt,
	}, true
}

func (e *Engine) publish(rec Recommendation, latest TimelineEntry) {
	out := RecommendationStateUpdated{
		RecommendationID:    rec.RecommendationID,
		Symbol:              rec.Symbol,
		Timeframe:           rec.Timeframe,
		Strategy:            rec.Strategy,
		CurrentStatus:       rec.CurrentStatus,
		Confidence:          rec.Confidence,
		LatestTimelineEntry: latest,
		Summary:             buildSummary(rec, latest),
	}
	evt, err := events.NewEventWithClock(e.clk, events.RecommendationStateUpdated, engineName, out)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

// List returns recommendations matching optional filters.
func (e *Engine) List(symbol, strategy, timeframe, status string, confidenceMin float64) []Recommendation {
	return e.cache.List(symbol, strategy, timeframe, status, confidenceMin)
}

// Get returns a recommendation and timeline by ID.
func (e *Engine) Get(id string) (Recommendation, []TimelineEntry, bool) {
	return e.cache.GetByID(id)
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
