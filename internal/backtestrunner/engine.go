package backtestrunner

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/delivery"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/feedback"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// DeliverySource exposes read-only delivery documents.
type DeliverySource interface {
	ListRecommendations(filter delivery.Filter) []delivery.DeliveryDocument
}

// Engine orchestrates historical research sessions over the existing replay pipeline.
type Engine struct {
	cfg      Config
	bus      ports.EventBus
	clk      clock.Clock
	runner   ReplayRunner
	delivery DeliverySource
	repo     *Repository
	health   healthSnapshot

	mu             sync.Mutex
	activeSessions int
	defaultRequest SessionRequest
	ctx            context.Context
	cancel         context.CancelFunc
	started        bool
	closed         bool
	wg             sync.WaitGroup
}

// New creates a historical backtest runner engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock, runner ReplayRunner, delivery DeliverySource) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if runner == nil {
		return nil, ErrNilRunner
	}
	if delivery == nil {
		return nil, ErrNilDeliverySource
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	return &Engine{
		cfg:      cfg,
		bus:      bus,
		clk:      clk,
		runner:   runner,
		delivery: delivery,
		repo:     NewRepository(),
	}, nil
}

// SetDefaultRequest configures the request used when auto_start is enabled.
func (e *Engine) SetDefaultRequest(req SessionRequest) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultRequest = req.withDefaults()
}

// Repository returns the session repository.
func (e *Engine) Repository() *Repository {
	return e.repo
}

// Start enables the runner and optionally auto-starts a configured session.
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
	e.started = true
	autoStart := e.cfg.AutoStart
	defaultReq := e.defaultRequest
	e.mu.Unlock()

	if autoStart {
		if err := defaultReq.Validate(); err == nil {
			e.wg.Add(1)
			go func() {
				defer e.wg.Done()
				_, _ = e.StartSession(engineCtx, defaultReq)
			}()
		}
	}
	return nil
}

// StartSession begins a historical research session and returns its identifier.
func (e *Engine) StartSession(ctx context.Context, req SessionRequest) (string, error) {
	if !e.cfg.Enabled {
		return "", ErrEngineClosed
	}
	req = req.withDefaults()
	if err := req.Validate(); err != nil {
		return "", err
	}

	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return "", ErrEngineClosed
	}
	if e.activeSessions >= e.cfg.ConcurrentSessions {
		e.mu.Unlock()
		return "", ErrConcurrentLimit
	}
	e.activeSessions++
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.activeSessions--
		e.mu.Unlock()
	}()

	at := e.clk.Now().UTC()
	session := newSession(req, at)
	e.health.recordStarted()
	e.publishStarted(session, at)

	runCtx := ctx
	if runCtx == nil {
		runCtx = e.ctx
	}
	if runCtx == nil {
		runCtx = context.Background()
	}

	collector := newSessionCollector(e.bus, e.cfg.SubscriberBuffer)
	collector.Start(runCtx)
	defer collector.Close()

	replayStarted := time.Now()
	replayDuration, err := e.runner.Execute(runCtx, req)
	if err != nil {
		return e.failSession(session, replayDuration, collector.Snapshot(), err, replayStarted)
	}

	docs := e.collectDocuments(req, collector.Snapshot())
	summary := BuildSummary(session.BacktestID, docs, collector.Snapshot(), e.clk.Now().UTC())
	completedAt := e.clk.Now().UTC()

	session.Status = SessionStatusCompleted
	session.ReplayDuration = replayDuration
	session.Summary = &summary
	session.CompletedAt = &completedAt

	e.repo.Save(session)
	e.health.recordCompleted(replayDuration, len(docs))
	e.publishCompleted(session, summary, completedAt, "")

	return session.BacktestID, nil
}

func (e *Engine) failSession(session Session, replayDuration time.Duration, snap CollectorSnapshot, runErr error, started time.Time) (string, error) {
	completedAt := e.clk.Now().UTC()
	session.Status = SessionStatusFailed
	session.ReplayDuration = replayDuration
	session.Error = runErr.Error()
	session.CompletedAt = &completedAt

	docs := e.collectDocuments(session.Request, snap)
	summary := BuildSummary(session.BacktestID, docs, snap, completedAt)
	session.Summary = &summary

	e.repo.Save(session)
	e.health.recordFailed()
	e.publishCompleted(session, summary, completedAt, runErr.Error())
	return session.BacktestID, runErr
}

func (e *Engine) collectDocuments(req SessionRequest, snap CollectorSnapshot) []delivery.DeliveryDocument {
	seen := make(map[string]delivery.DeliveryDocument, len(snap.Documents))
	for id, doc := range snap.Documents {
		seen[id] = doc
	}

	filter := delivery.Filter{
		CreatedAfter: req.StartTime,
		UpdatedAfter: req.StartTime,
	}
	for _, doc := range e.delivery.ListRecommendations(filter) {
		if !containsSymbol(req.Symbols, doc.Symbol) {
			continue
		}
		if !doc.UpdatedAt.IsZero() && doc.UpdatedAt.After(req.EndTime) {
			continue
		}
		seen[doc.RecommendationID] = doc
	}

	out := make([]delivery.DeliveryDocument, 0, len(seen))
	for _, doc := range seen {
		out = append(out, doc)
	}
	return out
}

func containsSymbol(symbols []string, symbol string) bool {
	for _, item := range symbols {
		if item == symbol {
			return true
		}
	}
	return false
}

func (e *Engine) publishStarted(session Session, at time.Time) {
	payload := SessionStarted{
		BacktestID: session.BacktestID,
		StartTime:  session.StartTime,
		EndTime:    session.EndTime,
		Symbols:    session.Request.Symbols,
		Expiries:   session.Request.Expiries,
		Mode:       session.Request.Mode,
		StartedAt:  at,
	}
	evt, err := events.NewEventWithClock(e.clk, events.BacktestSessionStarted, engineName, payload)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

func (e *Engine) publishCompleted(session Session, summary SessionSummary, at time.Time, errMsg string) {
	payload := SessionCompleted{
		BacktestID:     session.BacktestID,
		Status:         session.Status,
		ReplayDuration: session.ReplayDuration,
		Summary:        summary,
		CompletedAt:    at,
		Error:          errMsg,
	}
	evt, err := events.NewEventWithClock(e.clk, events.BacktestSessionCompleted, engineName, payload)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

// Close stops the runner and waits for active sessions.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	cancel := e.cancel
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()
	return nil
}

// Health reports runtime status for observability probes.
func (e *Engine) Health() health.Report {
	e.mu.Lock()
	connected := e.started && !e.closed
	active := e.activeSessions
	e.mu.Unlock()
	return e.health.report(e.cfg, connected, active)
}

type sessionCollector struct {
	bus          ports.EventBus
	buffer       int
	subscription *eventbus.Subscription

	mu               sync.Mutex
	documents        map[string]delivery.DeliveryDocument
	researchReports  int
	alertsGenerated  int
	optimizationRuns int
	walkForwardRuns  int
	monteCarloRuns   int
	feedback         *feedback.RecommendationFeedbackUpdated
}

func newSessionCollector(bus ports.EventBus, buffer int) *sessionCollector {
	if buffer <= 0 {
		buffer = 512
	}
	return &sessionCollector{
		bus:       bus,
		buffer:    buffer,
		documents: make(map[string]delivery.DeliveryDocument),
	}
}

func (c *sessionCollector) Start(ctx context.Context) {
	if c.bus == nil {
		return
	}
	c.subscription = c.bus.Subscribe(c.buffer, func(evt events.Event) bool {
		switch evt.Type {
		case events.RecommendationDeliveryUpdated,
			events.ResearchUpdated,
			events.RecommendationQualityUpdated,
			events.RecommendationFeedbackUpdated,
			events.AlertGenerated,
			events.OptimizationUpdated,
			events.WalkForwardCompleted,
			events.MonteCarloCompleted:
			return true
		default:
			return false
		}
	})
	go c.consume(ctx)
}

func (c *sessionCollector) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.drain()
			return
		case evt, ok := <-c.subscription.C:
			if !ok {
				return
			}
			c.handle(evt)
		}
	}
}

func (c *sessionCollector) drain() {
	for {
		select {
		case evt, ok := <-c.subscription.C:
			if !ok {
				return
			}
			c.handle(evt)
		default:
			return
		}
	}
}

func (c *sessionCollector) handle(evt events.Event) {
	switch evt.Type {
	case events.RecommendationDeliveryUpdated:
		var update delivery.RecommendationDeliveryUpdated
		if err := json.Unmarshal(evt.Payload, &update); err != nil || update.RecommendationID == "" {
			return
		}
		c.mu.Lock()
		c.documents[update.RecommendationID] = update.Document
		c.mu.Unlock()
	case events.ResearchUpdated:
		c.mu.Lock()
		c.researchReports++
		c.mu.Unlock()
	case events.RecommendationQualityUpdated:
		// Quality metrics are merged into delivery documents; no separate counter required.
	case events.RecommendationFeedbackUpdated:
		var update feedback.RecommendationFeedbackUpdated
		if err := json.Unmarshal(evt.Payload, &update); err != nil {
			return
		}
		c.mu.Lock()
		c.feedback = &update
		c.mu.Unlock()
	case events.AlertGenerated:
		c.mu.Lock()
		c.alertsGenerated++
		c.mu.Unlock()
	case events.OptimizationUpdated:
		c.mu.Lock()
		c.optimizationRuns++
		c.mu.Unlock()
	case events.WalkForwardCompleted:
		c.mu.Lock()
		c.walkForwardRuns++
		c.mu.Unlock()
	case events.MonteCarloCompleted:
		c.mu.Lock()
		c.monteCarloRuns++
		c.mu.Unlock()
	}
}

func (c *sessionCollector) Snapshot() CollectorSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	docs := make(map[string]delivery.DeliveryDocument, len(c.documents))
	for id, doc := range c.documents {
		docs[id] = doc
	}
	var feedbackCopy *feedback.RecommendationFeedbackUpdated
	if c.feedback != nil {
		copyValue := *c.feedback
		feedbackCopy = &copyValue
	}
	return CollectorSnapshot{
		Documents:        docs,
		ResearchReports:  c.researchReports,
		AlertsGenerated:  c.alertsGenerated,
		OptimizationRuns: c.optimizationRuns,
		WalkForwardRuns:  c.walkForwardRuns,
		MonteCarloRuns:   c.monteCarloRuns,
		Feedback:         feedbackCopy,
	}
}

func (c *sessionCollector) Close() {
	if c.subscription != nil {
		c.subscription.Close()
	}
}
