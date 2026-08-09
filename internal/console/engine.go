package console

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/delivery"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine consumes recommendation.delivery.updated and renders terminal output.
type Engine struct {
	cfg      Config
	bus      ports.EventBus
	renderer *Renderer
	health   healthSnapshot

	mu           sync.Mutex
	documents    map[string]delivery.DeliveryDocument
	timestamps   map[string]time.Time
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a recommendation console engine writing to stdout.
func New(cfg Config, bus ports.EventBus) (*Engine, error) {
	return NewWithWriter(cfg, bus, os.Stdout, true)
}

// NewWithWriter creates a recommendation console engine with a custom output writer.
func NewWithWriter(cfg Config, bus ports.EventBus, out io.Writer, overwrite bool) (*Engine, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	return &Engine{
		cfg:        cfg,
		bus:        bus,
		renderer:   NewRenderer(out, overwrite),
		documents:  make(map[string]delivery.DeliveryDocument),
		timestamps: make(map[string]time.Time),
	}, nil
}

// Start subscribes to delivery updates before the consumer goroutine starts.
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
		return evt.Type == events.RecommendationDeliveryUpdated
	})
	e.started = true
	e.mu.Unlock()

	e.wg.Add(1)
	go e.run(engineCtx)
	return nil
}

func (e *Engine) run(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.cfg.RefreshInterval)
	defer ticker.Stop()

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
		case <-ticker.C:
			e.refreshAll()
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
	if evt.Type != events.RecommendationDeliveryUpdated {
		return
	}

	update, ok := parseDeliveryUpdate(evt.Payload)
	if !ok {
		return
	}
	at := update.GeneratedAt
	if at.IsZero() {
		at = evt.Timestamp.UTC()
	}
	if update.Document.UpdatedAt.After(at) {
		at = update.Document.UpdatedAt.UTC()
	}

	e.mu.Lock()
	_, exists := e.documents[update.RecommendationID]
	e.documents[update.RecommendationID] = update.Document
	e.timestamps[update.RecommendationID] = at
	e.mu.Unlock()

	e.renderOne(update.RecommendationID, exists)
}

func (e *Engine) refreshAll() {
	e.mu.Lock()
	ids := make([]string, 0, len(e.documents))
	for id := range e.documents {
		ids = append(ids, id)
	}
	e.mu.Unlock()

	for _, id := range ids {
		e.renderOne(id, true)
	}
}

func (e *Engine) renderOne(id string, isUpdate bool) {
	start := time.Now()

	e.mu.Lock()
	doc, ok := e.documents[id]
	at := e.timestamps[id]
	e.mu.Unlock()
	if !ok {
		return
	}

	err := e.renderer.Render(doc, at, isUpdate)
	e.health.recordRender(time.Since(start), isUpdate, err)
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
	tracked := len(e.documents)
	if e.subscription != nil {
		dropped = e.subscription.Dropped()
	}
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, tracked)
}

func parseDeliveryUpdate(payload json.RawMessage) (delivery.RecommendationDeliveryUpdated, bool) {
	var update delivery.RecommendationDeliveryUpdated
	if err := json.Unmarshal(payload, &update); err != nil {
		return delivery.RecommendationDeliveryUpdated{}, false
	}
	if update.RecommendationID == "" {
		return delivery.RecommendationDeliveryUpdated{}, false
	}
	return update, true
}
