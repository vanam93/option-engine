package quality

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
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine consumes intelligence, candle, and state events and publishes recommendation.quality.updated.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	trackers  *TrackerRegistry
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

// New creates a recommendation quality engine.
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
		cfg:       cfg,
		bus:       bus,
		clk:       clk,
		trackers:  NewTrackerRegistry(),
		evaluator: NewEvaluator(cfg),
		cache:     NewCache(),
	}, nil
}

// Start subscribes to intelligence, candle, and state events before the consumer starts.
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
		return evt.Type == events.RecommendationIntelligenceUpdated ||
			evt.Type == events.CandleClosed ||
			evt.Type == events.RecommendationStateUpdated
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
	at := evt.Timestamp.UTC()
	switch evt.Type {
	case events.RecommendationIntelligenceUpdated:
		e.handleIntelligence(evt)
	case events.CandleClosed:
		e.handleCandle(evt)
	case events.RecommendationStateUpdated:
		e.handleState(evt)
	}
	e.checkTimeoutsAt(at)
}

func (e *Engine) handleIntelligence(evt events.Event) {
	input, ok := parseIntelligenceInput(evt.Payload)
	if !ok {
		return
	}
	if input.CurrentStatus == StatusClosed {
		e.finalizeByID(input.RecommendationID, false)
		return
	}

	at := evt.Timestamp.UTC()
	if !input.GeneratedAt.IsZero() {
		at = input.GeneratedAt.UTC()
	}

	tracker := e.trackers.Start(input, at)
	if tracker == nil {
		return
	}
	e.health.recordTracked()
	e.publishProgress(tracker.snapshot(), at)
}

func (e *Engine) handleState(evt events.Event) {
	input, ok := parseStateInput(evt.Payload)
	if !ok {
		return
	}

	at := evt.Timestamp.UTC()
	if !input.UpdatedAt.IsZero() {
		at = input.UpdatedAt.UTC()
	}

	tracker := e.trackers.ApplyState(input, at)
	if tracker == nil {
		return
	}

	if input.CurrentStatus == StatusClosed {
		e.finalizeTracker(tracker, at, false)
		return
	}
	e.publishProgress(tracker.snapshot(), at)
}

func (e *Engine) handleCandle(evt events.Event) {
	var candle market.Candle
	if err := json.Unmarshal(evt.Payload, &candle); err != nil {
		return
	}
	at := evt.Timestamp.UTC()
	if !candle.CloseTime.IsZero() {
		at = candle.CloseTime.UTC()
	}

	update := CandleUpdate{
		Symbol:    candle.Symbol,
		Timeframe: string(candle.Timeframe),
		Candle:    candle,
		At:        at,
	}
	ids := e.trackers.ApplyCandle(update)
	for _, id := range ids {
		tracker, ok := e.trackers.Get(id)
		if !ok || tracker == nil {
			continue
		}
		e.publishProgress(tracker.snapshot(), at)
	}
}

func (e *Engine) checkTimeoutsAt(now time.Time) {
	expired := e.trackers.CheckTimeouts(e.cfg.TrackingTimeout(), now)
	for _, tracker := range expired {
		e.finalizeTracker(tracker, now, true)
	}
}

func (e *Engine) finalizeByID(id string, expired bool) {
	tracker, ok := e.trackers.Remove(id)
	if !ok || tracker == nil {
		return
	}
	e.finalizeTracker(tracker, e.clk.Now().UTC(), expired)
}

func (e *Engine) finalizeTracker(tracker *activeTracker, at time.Time, expired bool) {
	snapshot := tracker.snapshot()
	report := e.evaluator.BuildReport(snapshot, at, !expired, expired)
	e.cache.Complete(report)
	e.health.recordCompleted()
	e.publish(report, at)
}

func (e *Engine) publishProgress(tracker activeTracker, at time.Time) {
	report := e.evaluator.BuildReport(tracker, at, false, false)
	e.cache.UpdateActive(report)
	e.publish(report, at)
}

func (e *Engine) publish(report QualityReport, at time.Time) {
	out := RecommendationQualityUpdated{
		RecommendationID: report.RecommendationID,
		Symbol:           report.Symbol,
		Timeframe:        report.Timeframe,
		Strategy:         report.Strategy,
		Report:           report,
		GeneratedAt:      at,
	}
	evt, err := events.NewEventWithClock(e.clk, events.RecommendationQualityUpdated, engineName, out)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

// GetLatest returns the latest quality report for a recommendation ID.
func (e *Engine) GetLatest(id string) (QualityReport, bool) {
	return e.cache.GetLatest(id)
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

	activeTrackers := e.trackers.ActiveCount()
	active, completed, stats := e.cache.Stats()
	_ = active
	return e.health.report(e.cfg, connected, dropped, activeTrackers, completed, stats)
}

func parseIntelligenceInput(payload json.RawMessage) (IntelligenceInput, bool) {
	var raw struct {
		RecommendationID string    `json:"recommendation_id"`
		Symbol           string    `json:"symbol"`
		Timeframe        string    `json:"timeframe"`
		Strategy         string    `json:"strategy"`
		GeneratedAt      time.Time `json:"generated_at"`
		Document         struct {
			RecommendationLevel Level   `json:"recommendation_level"`
			Confidence          float64 `json:"confidence"`
			CurrentStatus       Status  `json:"current_status"`
		} `json:"document"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return IntelligenceInput{}, false
	}
	if raw.RecommendationID == "" || raw.Symbol == "" {
		return IntelligenceInput{}, false
	}

	status := raw.Document.CurrentStatus
	if status == "" {
		status = StatusActive
	}

	return IntelligenceInput{
		RecommendationID:    raw.RecommendationID,
		Symbol:              raw.Symbol,
		Timeframe:           raw.Timeframe,
		Strategy:            strings.TrimSpace(raw.Strategy),
		RecommendationLevel: raw.Document.RecommendationLevel,
		Confidence:          raw.Document.Confidence,
		CurrentStatus:       status,
		GeneratedAt:         raw.GeneratedAt,
	}, true
}

func parseStateInput(payload json.RawMessage) (StateInput, bool) {
	var raw struct {
		RecommendationID    string  `json:"recommendation_id"`
		Symbol              string  `json:"symbol"`
		Timeframe           string  `json:"timeframe"`
		Strategy            string  `json:"strategy"`
		CurrentStatus       Status  `json:"current_status"`
		Confidence          float64 `json:"confidence"`
		LatestTimelineEntry struct {
			Timestamp time.Time `json:"timestamp"`
		} `json:"latest_timeline_entry"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return StateInput{}, false
	}
	if raw.RecommendationID == "" {
		return StateInput{}, false
	}

	updatedAt := raw.LatestTimelineEntry.Timestamp
	return StateInput{
		RecommendationID: raw.RecommendationID,
		Symbol:           raw.Symbol,
		Timeframe:        raw.Timeframe,
		Strategy:         strings.TrimSpace(raw.Strategy),
		CurrentStatus:    raw.CurrentStatus,
		Confidence:       raw.Confidence,
		UpdatedAt:        updatedAt,
	}, true
}
