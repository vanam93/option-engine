package alerts

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

// Engine consumes recommendation.state.updated events and publishes alert.generated events.
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

// New creates an alert engine subscribed to recommendation.state.updated events only.
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
		cache: NewCache(cfg.Cooldown()),
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

	firstSeen := e.cache.MarkSeen(update.RecommendationID)
	candidates := evaluateAlerts(update, e.cfg, firstSeen)
	if len(candidates) == 0 {
		if !firstSeen {
			e.health.recordNoMeaningfulAlert()
		}
		return
	}

	for _, candidate := range candidates {
		if !e.cache.ShouldEmit(update, candidate.AlertType, candidate.Reason, at) {
			e.health.record(candidate.AlertType, false, false, true)
			continue
		}
		e.publish(update, candidate, at)
		e.health.record(candidate.AlertType, true, false, false)
	}
}

func parseStateUpdate(payload json.RawMessage) (StateUpdate, bool) {
	var raw struct {
		RecommendationID    string        `json:"recommendation_id"`
		Symbol              string        `json:"symbol"`
		Timeframe           string        `json:"timeframe"`
		Strategy            string        `json:"strategy"`
		CurrentStatus       string        `json:"current_status"`
		Confidence          float64       `json:"confidence"`
		LatestTimelineEntry TimelineEntry `json:"latest_timeline_entry"`
		Summary             string        `json:"summary"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return StateUpdate{}, false
	}
	if raw.RecommendationID == "" || raw.Symbol == "" || raw.Timeframe == "" {
		return StateUpdate{}, false
	}

	return StateUpdate{
		RecommendationID:    raw.RecommendationID,
		Symbol:              raw.Symbol,
		Timeframe:           raw.Timeframe,
		Strategy:            strings.TrimSpace(raw.Strategy),
		CurrentStatus:       Status(strings.ToUpper(strings.TrimSpace(raw.CurrentStatus))),
		Confidence:          raw.Confidence,
		LatestTimelineEntry: raw.LatestTimelineEntry,
		Summary:             raw.Summary,
	}, true
}

func (e *Engine) publish(update StateUpdate, candidate candidateAlert, at time.Time) {
	out := AlertGenerated{
		AlertID:          e.cache.NextAlertID(update.Symbol, at),
		RecommendationID: update.RecommendationID,
		Symbol:           update.Symbol,
		Timeframe:        update.Timeframe,
		AlertType:        candidate.AlertType,
		CurrentStatus:    update.CurrentStatus,
		Confidence:       update.Confidence,
		Message:          candidate.Message,
		Reason:           candidate.Reason,
		GeneratedAt:      at,
	}
	evt, err := events.NewEventWithClock(e.clk, events.AlertGenerated, engineName, out)
	if err != nil {
		return
	}
	e.cache.Record(out)
	e.bus.Publish(evt)
}

// List returns alerts matching optional filters.
func (e *Engine) List(symbol, strategy, timeframe, status string, confidenceMin float64) []AlertGenerated {
	_ = strategy
	return e.cache.List(symbol, strategy, timeframe, status, confidenceMin)
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
