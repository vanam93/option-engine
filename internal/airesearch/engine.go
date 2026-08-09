package airesearch

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// StudySource provides read access to completed research studies.
type StudySource interface {
	GetStudy(id string) (laboratory.Study, bool)
}

// Engine consumes study.completed events and produces AI research reports.
type Engine struct {
	cfg      Config
	bus      ports.EventBus
	clk      clock.Clock
	studies  StudySource
	builder  *ReportBuilder
	repo     *Repository
	health   healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates an AI research engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock, studies StudySource) (*Engine, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if studies == nil {
		return nil, ErrNilStudySource
	}
	if clk == nil {
		clk = clock.NewSystem()
	}

	analyzer, err := newAnalyzer(cfg.Analyzer)
	if err != nil {
		return nil, err
	}

	return &Engine{
		cfg:     cfg,
		bus:     bus,
		clk:     clk,
		studies: studies,
		builder: NewReportBuilder(cfg, analyzer),
		repo:    NewRepository(),
	}, nil
}

func newAnalyzer(name string) (ResearchAnalyzer, error) {
	switch name {
	case AnalyzerRuleBased:
		return NewRuleBasedAnalyzer(), nil
	default:
		return nil, ErrUnknownAnalyzer
	}
}

// Repository returns the report repository.
func (e *Engine) Repository() *Repository {
	return e.repo
}

// Start subscribes to study.completed before the consumer goroutine starts.
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
		return evt.Type == events.StudyCompleted
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
	if evt.Type != events.StudyCompleted {
		return
	}

	var completed laboratory.StudyCompleted
	if err := json.Unmarshal(evt.Payload, &completed); err != nil {
		return
	}

	study, ok := e.studies.GetStudy(completed.StudyID)
	if !ok {
		return
	}

	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	started := time.Now()
	report, err := e.builder.Build(e.ctx, study, e.clk.Now().UTC())
	if err != nil {
		return
	}
	e.health.recordGenerated(time.Since(started))

	e.repo.Save(report)
	e.health.recordCached()

	e.publishCompleted(report)
}

func (e *Engine) publishCompleted(report ResearchReport) {
	payload := StudyAICompleted{
		ReportID:         report.ReportID,
		StudyID:          report.StudyID,
		ResearchVersion:  report.ResearchVersion,
		Analyzer:         report.Analyzer,
		ExecutiveSummary: report.Sections.ExecutiveSummary,
		OverallVerdict:   report.Sections.OverallVerdict,
		CompletedAt:      report.GeneratedAt,
	}
	evt, err := events.NewEventWithClock(e.clk, events.StudyAICompleted, engineName, payload)
	if err != nil {
		e.health.recordPublishFailure()
		return
	}
	e.bus.Publish(evt)
}

// Close stops the engine and waits for the consumer goroutine.
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
	if sub != nil {
		sub.Close()
	}
	e.wg.Wait()
	return nil
}

// Health reports runtime status for observability probes.
func (e *Engine) Health() health.Report {
	e.mu.Lock()
	connected := e.started && !e.closed
	sub := e.subscription
	e.mu.Unlock()

	var dropped uint64
	if sub != nil {
		dropped = sub.Dropped()
	}

	return e.health.report(e.cfg, connected, e.repo.Count(), dropped)
}
