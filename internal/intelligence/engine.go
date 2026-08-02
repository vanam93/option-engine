package intelligence

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
)

// Engine consumes recommendation.state.updated events and publishes recommendation.intelligence.updated.
type Engine struct {
	cfg     Config
	bus     ports.EventBus
	clk     clock.Clock
	cache   *Cache
	builder *Builder
	health  healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a recommendation intelligence engine.
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
		cfg:     cfg,
		bus:     bus,
		clk:     clk,
		cache:   NewCache(),
		builder: NewBuilder(cfg),
	}, nil
}

// Start subscribes to recommendation.state.updated before the consumer goroutine starts.
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
		return evt.Type == events.RecommendationStateUpdated
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
	update, ok := parseStateUpdate(evt.Payload)
	if !ok {
		return
	}

	at := e.clk.Now().UTC()
	if !update.LatestTimelineEntry.Timestamp.IsZero() {
		at = update.LatestTimelineEntry.Timestamp.UTC()
	}

	var previousTimeline []TimelineEntry
	var previous *storedSnapshot
	if prev := e.cache.Peek(update.RecommendationID); prev != nil {
		previousTimeline = prev.timeline
		previous = prev
	}
	timeline := mergeTimeline(previousTimeline, update.LatestTimelineEntry)

	doc := e.builder.Build(update, timeline, previous, at)
	e.cache.Apply(doc, timeline)

	e.publish(doc, at)
	e.health.record(doc, doc.ReasonForUpgrade != "", doc.ReasonForDowngrade != "")
}

func parseStateUpdate(payload json.RawMessage) (StateUpdate, bool) {
	var raw struct {
		RecommendationID     string             `json:"recommendation_id"`
		Symbol               string             `json:"symbol"`
		Timeframe            string             `json:"timeframe"`
		Strategy             string             `json:"strategy"`
		Recommendation       string             `json:"recommendation"`
		CurrentStatus        string             `json:"current_status"`
		Confidence           float64            `json:"confidence"`
		LatestTimelineEntry  TimelineEntry      `json:"latest_timeline_entry"`
		Summary              string             `json:"summary"`
		Reasons              []string           `json:"reasons"`
		SupportingIndicators []string           `json:"supporting_indicators"`
		SupportingStrategies []string           `json:"supporting_strategies"`
		OptimizationSummary  string             `json:"optimization_summary"`
		WalkForwardSummary   string             `json:"walk_forward_summary"`
		MonteCarloSummary    string             `json:"monte_carlo_summary"`
		Components           map[string]float64 `json:"components"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return StateUpdate{}, false
	}
	if raw.RecommendationID == "" || raw.Symbol == "" || raw.Timeframe == "" {
		return StateUpdate{}, false
	}

	level := Level(strings.ToUpper(strings.TrimSpace(raw.Recommendation)))
	if level == "" {
		level = ""
	}

	return StateUpdate{
		RecommendationID:     raw.RecommendationID,
		Symbol:               raw.Symbol,
		Timeframe:            raw.Timeframe,
		Strategy:             strings.TrimSpace(raw.Strategy),
		Recommendation:       level,
		CurrentStatus:        Status(strings.ToUpper(strings.TrimSpace(raw.CurrentStatus))),
		Confidence:           raw.Confidence,
		LatestTimelineEntry:  raw.LatestTimelineEntry,
		Summary:              raw.Summary,
		Reasons:              raw.Reasons,
		SupportingIndicators: raw.SupportingIndicators,
		SupportingStrategies: raw.SupportingStrategies,
		OptimizationSummary:  raw.OptimizationSummary,
		WalkForwardSummary:   raw.WalkForwardSummary,
		MonteCarloSummary:    raw.MonteCarloSummary,
		Components:           raw.Components,
	}, true
}

func (e *Engine) publish(doc IntelligenceDocument, at time.Time) {
	out := RecommendationIntelligenceUpdated{
		RecommendationID: doc.RecommendationID,
		Symbol:           doc.Symbol,
		Timeframe:        doc.Timeframe,
		Strategy:         doc.Strategy,
		Document:         doc,
		GeneratedAt:      at,
	}
	evt, err := events.NewEventWithClock(e.clk, events.RecommendationIntelligenceUpdated, engineName, out)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

// Get returns the latest intelligence document for a recommendation ID.
func (e *Engine) Get(id string) (IntelligenceDocument, bool) {
	return e.cache.Get(id)
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

	documents, avgConfidence := e.cache.Stats()
	return e.health.report(e.cfg, connected, dropped, documents, avgConfidence)
}
