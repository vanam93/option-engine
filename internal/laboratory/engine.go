package laboratory

import (
	"context"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// Engine orchestrates research studies through the historical backtest runner.
type Engine struct {
	cfg     Config
	bus     ports.EventBus
	clk     clock.Clock
	runner  *StudyRunner
	repo    *Repository
	catalog *Catalog
	health  healthSnapshot

	mu            sync.Mutex
	activeStudies int
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
	closed        bool
	wg            sync.WaitGroup
}

// New creates a strategy laboratory engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock, backtest BacktestRunner, sessions SessionSource) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if backtest == nil {
		return nil, ErrNilBacktestRunner
	}
	if sessions == nil {
		return nil, ErrNilSessionSource
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	return &Engine{
		cfg:     cfg,
		bus:     bus,
		clk:     clk,
		runner:  NewStudyRunner(backtest, sessions),
		repo:    NewRepository(),
		catalog: NewCatalog(),
	}, nil
}

// Repository returns the study repository.
func (e *Engine) Repository() *Repository {
	return e.repo
}

// Catalog returns the study catalog.
func (e *Engine) Catalog() *Catalog {
	return e.catalog
}

// Start enables the laboratory engine.
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
	e.mu.Unlock()
	return nil
}

// CreateStudy registers a new research study and returns its identifier.
func (e *Engine) CreateStudy(req StudyRequest) (string, error) {
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
	e.mu.Unlock()

	version := "v1"
	if e.cfg.AutoVersion {
		version = e.repo.NextVersion(req.Strategy, req.Parameters)
	}

	at := e.clk.Now().UTC()
	study := newStudy(req, version, at)
	e.repo.SaveStudy(study)
	e.catalog.Index(study)
	e.health.recordCreated()

	return study.StudyID, nil
}

// CreateVersion creates a new version of an existing study definition.
func (e *Engine) CreateVersion(studyID string) (string, error) {
	if !e.cfg.Enabled {
		return "", ErrEngineClosed
	}

	existing, ok := e.repo.GetStudy(studyID)
	if !ok {
		return "", ErrStudyNotFound
	}

	req := StudyRequest{
		Name:        existing.Name,
		Description: existing.Description,
		Strategy:    existing.Strategy,
		Parameters:  copyParameters(existing.Parameters),
		Symbols:     append([]string(nil), existing.Symbols...),
		Timeframes:  append([]market.Timeframe(nil), existing.Timeframes...),
		StartTime:   existing.StartTime,
		EndTime:     existing.EndTime,
	}

	return e.CreateStudy(req)
}

// ExecuteStudy runs a study through the historical backtest runner.
func (e *Engine) ExecuteStudy(ctx context.Context, studyID string) error {
	if !e.cfg.Enabled {
		return ErrEngineClosed
	}

	study, ok := e.repo.GetStudy(studyID)
	if !ok {
		return ErrStudyNotFound
	}
	if study.Status == StudyStatusRunning {
		return ErrStudyInProgress
	}

	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return ErrEngineClosed
	}
	if e.activeStudies >= e.cfg.ConcurrentStudies {
		e.mu.Unlock()
		return ErrConcurrentLimit
	}
	e.activeStudies++
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.activeStudies--
		e.mu.Unlock()
	}()

	prevStatus := study.Status
	study.Status = StudyStatusRunning
	e.repo.SaveStudy(study)
	e.catalog.UpdateStatus(study.StudyID, prevStatus, StudyStatusRunning)

	startedAt := e.clk.Now().UTC()
	e.publishStarted(study, startedAt)

	runCtx := ctx
	if runCtx == nil {
		runCtx = e.ctx
	}
	if runCtx == nil {
		runCtx = context.Background()
	}

	execStarted := time.Now()
	sessionID, session, err := e.runner.Execute(runCtx, study)
	execDuration := time.Since(execStarted)

	completedAt := e.clk.Now().UTC()
	study.BacktestSessionIDs = append(study.BacktestSessionIDs, sessionID)

	if err != nil {
		study.Status = StudyStatusFailed
		study.Error = err.Error()
		study.CompletedAt = &completedAt
		e.repo.SaveStudy(study)
		e.catalog.UpdateStatus(study.StudyID, StudyStatusRunning, StudyStatusFailed)
		e.health.recordFailed(execDuration)
		e.publishCompleted(study, completedAt, err.Error())
		return err
	}

	output := buildStudyOutput([]backtestrunner.Session{session})
	study.Output = &output
	study.Status = StudyStatusCompleted
	if session.Status == backtestrunner.SessionStatusFailed {
		study.Status = StudyStatusFailed
		if session.Error != "" {
			study.Error = session.Error
		}
	}
	study.CompletedAt = &completedAt
	e.repo.SaveStudy(study)
	e.catalog.UpdateStatus(study.StudyID, StudyStatusRunning, study.Status)
	e.health.recordCompleted(execDuration)
	e.publishCompleted(study, completedAt, study.Error)

	return nil
}

// GetStudy returns a study by ID.
func (e *Engine) GetStudy(id string) (Study, bool) {
	return e.repo.GetStudy(id)
}

// Close stops the laboratory and waits for active studies.
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
	active := e.activeStudies
	e.mu.Unlock()
	return e.health.report(e.cfg, connected, active, e.repo.Count(), e.repo.ComparisonCount())
}

func (e *Engine) publishStarted(study Study, at time.Time) {
	payload := StudyStarted{
		StudyID:         study.StudyID,
		Name:            study.Name,
		Strategy:        study.Strategy,
		Symbols:         append([]string(nil), study.Symbols...),
		ResearchVersion: study.ResearchVersion,
		StartedAt:       at,
	}
	evt, err := events.NewEventWithClock(e.clk, events.StudyStarted, engineName, payload)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

func (e *Engine) publishCompleted(study Study, at time.Time, errMsg string) {
	payload := StudyCompleted{
		StudyID:            study.StudyID,
		Status:             study.Status,
		ResearchVersion:    study.ResearchVersion,
		BacktestSessionIDs: append([]string(nil), study.BacktestSessionIDs...),
		CompletedAt:        at,
		Error:              errMsg,
	}
	evt, err := events.NewEventWithClock(e.clk, events.StudyCompleted, engineName, payload)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

func (e *Engine) publishCompared(comparison Comparison) {
	payload := StudyCompared{
		ComparisonID: comparison.ComparisonID,
		Criteria:     comparison.Criteria,
		StudyIDs:     append([]string(nil), comparison.StudyIDs...),
		ComparedAt:   comparison.CreatedAt,
	}
	evt, err := events.NewEventWithClock(e.clk, events.StudyCompared, engineName, payload)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}
