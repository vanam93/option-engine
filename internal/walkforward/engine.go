package walkforward

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/experiments"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// Engine orchestrates walk-forward analysis across rolling train/validation windows.
type Engine struct {
	cfg           Config
	bus           ports.EventBus
	clk           clock.Clock
	runner        experiments.BacktestRunner
	cache         *Cache
	walkForwardID string
	health        healthSnapshot

	scheduler *Scheduler
	workerWG  *sync.WaitGroup

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
}

type windowSession struct {
	mu            sync.Mutex
	walkForwardID string
	windowIndex   int
	phase         string
	training      *trainingCollector
	validation    *validationCollector
}

type trainingCollector struct {
	mu        sync.Mutex
	pending   map[string]struct{}
	runs      map[string]experiments.ExperimentRun
	collected []experiments.RunResult
	done      chan struct{}
	closed    bool
}

type validationCollector struct {
	mu     sync.Mutex
	runID  string
	result *optimization.OptimizationUpdated
	done   chan struct{}
	closed bool
}

// New creates a walk-forward analysis engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock, runner experiments.BacktestRunner) (*Engine, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	walkForwardID := GenerateWalkForwardID()
	return &Engine{
		cfg:           cfg,
		bus:           bus,
		clk:           clk,
		runner:        runner,
		cache:         NewCache(walkForwardID),
		walkForwardID: walkForwardID,
		scheduler:     NewScheduler(8),
	}, nil
}

// Start subscribes to optimization.updated and begins window processing.
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
		return evt.Type == events.OptimizationUpdated
	})
	e.started = true
	e.mu.Unlock()

	windows := GenerateWindows(
		e.walkForwardID,
		e.cfg.DataStart,
		e.cfg.DataEnd,
		e.cfg.TrainWindowDays,
		e.cfg.ValidationWindowDays,
		e.cfg.StepDays,
	)
	if len(windows) == 0 {
		return ErrInvalidRange
	}
	e.cache.RegisterWindows(windows)

	if e.runner != nil {
		e.workerWG = e.scheduler.Start(engineCtx, e.processWindow)
		go func() {
			e.scheduler.Enqueue(windows)
		}()
	}

	return nil
}

func (e *Engine) processWindow(ctx context.Context, window Window) error {
	session := &windowSession{
		walkForwardID: window.WalkForwardID,
		windowIndex:   window.Index,
	}

	e.cache.MarkTraining(window.Index)

	experimentID := experiments.GenerateExperimentID()
	trainingRuns := e.generateTrainingRuns(window, experimentID)
	if len(trainingRuns) == 0 {
		e.cache.MarkFailed(window.Index)
		return nil
	}

	session.phase = "training"
	session.training = newTrainingCollector(trainingRuns)

	for _, run := range trainingRuns {
		if err := e.runner.Execute(ctx, run); err != nil {
			e.cache.MarkFailed(window.Index)
			return err
		}
		if err := e.awaitOptimization(ctx, session, run.RunID); err != nil {
			e.cache.MarkFailed(window.Index)
			return err
		}
	}

	best, ok := SelectBest(session.training.snapshotResults())
	if !ok {
		e.cache.MarkFailed(window.Index)
		return nil
	}

	e.cache.MarkValidating(window.Index)
	session.phase = "validation"

	validationRun := e.generateValidationRun(window, experimentID, best)
	session.validation = newValidationCollector(validationRun.RunID)

	if err := e.runner.Execute(ctx, validationRun); err != nil {
		e.cache.MarkFailed(window.Index)
		return err
	}

	update, err := e.waitForValidation(ctx, session)
	if err != nil {
		e.cache.MarkFailed(window.Index)
		return err
	}

	at := update.Timestamp
	if at.IsZero() {
		at = e.clk.Now()
	}

	result := WindowResult{
		WalkForwardID:    window.WalkForwardID,
		WindowIndex:      window.Index,
		ExperimentID:     experimentID,
		RunID:            validationRun.RunID,
		TrainPeriod:      window.TrainPeriod(),
		ValidationPeriod: window.ValidationPeriod(),
		BestParameters:   stripMetadata(best.Parameters),
		TrainingScore:    best.OptimizationScore,
		ValidationScore:  update.Score,
		Metrics:          update.Metrics,
		CompletedAt:      at,
	}
	e.cache.StoreResult(result)
	e.publishCompleted(result)
	return nil
}

func (e *Engine) awaitOptimization(ctx context.Context, session *windowSession, runID string) error {
	deadline := time.After(2 * time.Minute)
	for {
		if session.training != nil && session.training.hasResult(runID) {
			return nil
		}
		if session.validation != nil && session.validation.hasResult(runID) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-e.subscription.C:
			if !ok {
				return context.Canceled
			}
			e.dispatchOptimization(session, evt)
		case <-deadline:
			return context.DeadlineExceeded
		}
	}
}

func (e *Engine) waitForValidation(ctx context.Context, session *windowSession) (optimization.OptimizationUpdated, error) {
	if err := e.awaitOptimization(ctx, session, session.validation.runID); err != nil {
		return optimization.OptimizationUpdated{}, err
	}
	session.validation.mu.Lock()
	defer session.validation.mu.Unlock()
	if session.validation.result == nil {
		return optimization.OptimizationUpdated{}, context.DeadlineExceeded
	}
	return *session.validation.result, nil
}

func (e *Engine) dispatchOptimization(session *windowSession, evt events.Event) {
	update, ok := parseOptimizationUpdate(evt.Payload)
	if !ok {
		return
	}
	session.handle(update)
}

func (e *Engine) generateTrainingRuns(window Window, experimentID string) []experiments.ExperimentRun {
	runs := experiments.GenerateMatrix(e.cfg.Experiments, experimentID)
	out := make([]experiments.ExperimentRun, 0, len(runs))
	for _, run := range runs {
		enriched := enrichRun(window, run, "training")
		out = append(out, enriched)
	}
	return out
}

func (e *Engine) generateValidationRun(window Window, experimentID string, best experiments.RunResult) experiments.ExperimentRun {
	runID := experiments.GenerateRunID()
	params := stripMetadata(best.Parameters)
	params["run_id"] = runID
	params["experiment_id"] = experimentID

	run := experiments.ExperimentRun{
		ExperimentID: experimentID,
		RunID:        runID,
		BacktestID:   window.WalkForwardID,
		Strategy:     best.Strategy,
		Symbol:       stringParam(best.Parameters, "symbol", e.cfg.Experiments.Symbols[0]),
		Timeframe:    stringParam(best.Parameters, "timeframe", e.cfg.Experiments.Timeframes[0]),
		Parameters:   params,
		Status:       experiments.RunStatusCreated,
	}
	return enrichRun(window, run, "validation")
}

func enrichRun(window Window, run experiments.ExperimentRun, phase string) experiments.ExperimentRun {
	params := cloneParams(run.Parameters)
	params["walkforward_id"] = window.WalkForwardID
	params["window_index"] = window.Index
	params["phase"] = phase
	params["train_start"] = window.TrainStart.Format(time.RFC3339)
	params["train_end"] = window.TrainEnd.Format(time.RFC3339)
	params["validation_start"] = window.ValidationStart.Format(time.RFC3339)
	params["validation_end"] = window.ValidationEnd.Format(time.RFC3339)
	params["run_id"] = run.RunID
	params["experiment_id"] = run.ExperimentID
	run.Parameters = params
	return run
}

func newTrainingCollector(runs []experiments.ExperimentRun) *trainingCollector {
	pending := make(map[string]struct{}, len(runs))
	runMap := make(map[string]experiments.ExperimentRun, len(runs))
	for _, run := range runs {
		pending[run.RunID] = struct{}{}
		runMap[run.RunID] = run
	}
	return &trainingCollector{
		pending: pending,
		runs:    runMap,
		done:    make(chan struct{}),
	}
}

func (c *trainingCollector) accept(update optimization.OptimizationUpdated) bool {
	runID := experiments.RunIDFromParameters(update.Parameters)
	if runID == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	if _, ok := c.pending[runID]; !ok {
		return false
	}
	delete(c.pending, runID)

	run := c.runs[runID]
	at := update.Timestamp
	if at.IsZero() {
		at = time.Now().UTC()
	}
	c.collected = append(c.collected, experiments.RunResult{
		RunID:             runID,
		ExperimentID:      run.ExperimentID,
		Strategy:          update.Strategy,
		Parameters:        cloneParams(run.Parameters),
		OptimizationScore: update.Score,
		Metrics:           update.Metrics,
		CompletedAt:       at,
	})

	if len(c.pending) == 0 {
		c.closed = true
		close(c.done)
	}
	return true
}

func (c *trainingCollector) hasResult(runID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, result := range c.collected {
		if result.RunID == runID {
			return true
		}
	}
	return false
}

func (c *validationCollector) hasResult(runID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.result != nil && experiments.RunIDFromParameters(c.result.Parameters) == runID
}

func (c *trainingCollector) snapshotResults() []experiments.RunResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]experiments.RunResult(nil), c.collected...)
}

func newValidationCollector(runID string) *validationCollector {
	return &validationCollector{
		runID: runID,
		done:  make(chan struct{}),
	}
}

func (c *validationCollector) accept(update optimization.OptimizationUpdated) bool {
	runID := experiments.RunIDFromParameters(update.Parameters)
	if runID == "" || runID != c.runID {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	copyUpdate := update
	c.result = &copyUpdate
	c.closed = true
	close(c.done)
	return true
}

func (s *windowSession) handle(update optimization.OptimizationUpdated) {
	phase := metadataString(update.Parameters, "phase")
	windowIndex := metadataInt(update.Parameters, "window_index")
	if windowIndex >= 0 && windowIndex != s.windowIndex {
		return
	}
	if phase == "" {
		return
	}

	switch phase {
	case "training":
		if s.training != nil {
			s.training.accept(update)
		}
	case "validation":
		if s.validation != nil {
			s.validation.accept(update)
		}
	}
}

func (e *Engine) publishCompleted(result WindowResult) {
	out, err := events.NewEventWithClock(e.clk, events.WalkForwardCompleted, engineName, WalkForwardCompleted{
		WalkForwardID:      result.WalkForwardID,
		ExperimentID:       result.ExperimentID,
		RunID:              result.RunID,
		TrainPeriod:        result.TrainPeriod,
		ValidationPeriod:   result.ValidationPeriod,
		BestParameters:     result.BestParameters,
		TrainingScore:      result.TrainingScore,
		ValidationScore:    result.ValidationScore,
		PerformanceMetrics: result.Metrics,
		Timestamp:          result.CompletedAt,
	})
	if err != nil {
		return
	}
	e.bus.Publish(out)
	e.health.recordReport()
}

// State returns an immutable snapshot of walk-forward state.
func (e *Engine) State() StateSnapshot {
	return e.cache.Snapshot()
}

// Close stops the engine and releases resources.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	cancel := e.cancel
	sub := e.subscription
	workerWG := e.workerWG
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if workerWG != nil {
		workerWG.Wait()
	}
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
	scheduler := e.scheduler
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, e.cache, scheduler)
}

func parseOptimizationUpdate(payload json.RawMessage) (optimization.OptimizationUpdated, bool) {
	var update optimization.OptimizationUpdated
	if err := json.Unmarshal(payload, &update); err != nil {
		return optimization.OptimizationUpdated{}, false
	}
	return update, true
}

func metadataString(serialized, key string) string {
	if serialized == "" {
		return ""
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
		return ""
	}
	if v, ok := raw[key].(string); ok {
		return v
	}
	return ""
}

func metadataInt(serialized, key string) int {
	if serialized == "" {
		return -1
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
		return -1
	}
	switch v := raw[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return -1
	}
}

func stringParam(params experiments.ParameterSet, key, fallback string) string {
	if v, ok := params[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func cloneParams(src experiments.ParameterSet) experiments.ParameterSet {
	if len(src) == 0 {
		return experiments.ParameterSet{}
	}
	out := make(experiments.ParameterSet, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
