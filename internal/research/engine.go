package research

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
	"github.com/vanam-gangireddy/option-engine/internal/montecarlo"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/walkforward"
)

// Engine persists research artifacts and generates reports from PostgreSQL.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	repo      Repository
	reports   *ReportGenerator
	exporters []Exporter
	cache     *Cache
	health    healthSnapshot
	workerWG  sync.WaitGroup

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
}

// New creates a research repository and reporting engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock, repo Repository) (*Engine, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if repo == nil {
		return nil, ErrNilRepository
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	return &Engine{
		cfg:       cfg,
		bus:       bus,
		clk:       clk,
		repo:      repo,
		reports:   NewReportGenerator(repo),
		exporters: NewExporters(cfg.Formats),
		cache:     NewCache(),
	}, nil
}

// Start subscribes to research pipeline events and ensures database schema.
func (e *Engine) Start(ctx context.Context) error {
	if !e.cfg.Enabled {
		return nil
	}

	if err := e.repo.EnsureSchema(ctx); err != nil {
		return err
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
		switch evt.Type {
		case events.OptimizationUpdated, events.WalkForwardCompleted, events.MonteCarloCompleted:
			return true
		default:
			return false
		}
	})
	e.started = true
	e.mu.Unlock()

	e.workerWG.Add(1)
	go e.consume(engineCtx)

	return nil
}

func (e *Engine) consume(ctx context.Context) {
	defer e.workerWG.Done()
	for {
		select {
		case <-ctx.Done():
			e.drain()
			return
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handleEvent(evt)
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
			e.handleEvent(evt)
		default:
			return
		}
	}
}

func (e *Engine) handleEvent(evt events.Event) {
	switch evt.Type {
	case events.OptimizationUpdated:
		e.handleOptimization(evt)
	case events.WalkForwardCompleted:
		e.handleWalkForward(evt)
	case events.MonteCarloCompleted:
		e.handleMonteCarlo(evt)
	}
}

func (e *Engine) handleOptimization(evt events.Event) {
	update, ok := parseOptimizationUpdate(evt.Payload)
	if !ok {
		return
	}
	experimentID := experimentIDFromParameters(update.Parameters)
	if experimentID == "" {
		return
	}

	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	at := update.Timestamp
	if at.IsZero() {
		at = e.clk.Now()
	}

	params := parametersJSON(update.Parameters)
	exp := ResearchExperiment{
		ExperimentID: experimentID,
		Strategy:     update.Strategy,
		Symbol:       update.Symbol,
		Timeframe:    update.Timeframe,
		Parameters:   params,
		CreatedAt:    at,
	}
	if err := e.repo.UpsertExperiment(ctx, exp); err != nil {
		return
	}
	e.health.recordWrite()

	result := optimizationFromUpdate(experimentID, update, at)
	if err := e.repo.InsertOptimizationResult(ctx, result); err != nil {
		return
	}
	e.health.recordWrite()
}

func (e *Engine) handleWalkForward(evt events.Event) {
	completed, ok := parseWalkForwardCompleted(evt.Payload)
	if !ok || completed.ExperimentID == "" {
		return
	}

	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	at := completed.Timestamp
	if at.IsZero() {
		at = e.clk.Now()
	}

	params, _ := json.Marshal(completed.BestParameters)
	exp := ResearchExperiment{
		ExperimentID: completed.ExperimentID,
		Strategy:     stringFromParams(completed.BestParameters, "strategy"),
		Symbol:       stringFromParams(completed.BestParameters, "symbol"),
		Timeframe:    stringFromParams(completed.BestParameters, "timeframe"),
		Parameters:   params,
		CreatedAt:    at,
	}
	if exp.Strategy == "" {
		exp.Strategy = "unknown"
	}
	if exp.Symbol == "" {
		exp.Symbol = "unknown"
	}
	if exp.Timeframe == "" {
		exp.Timeframe = "unknown"
	}
	if err := e.repo.UpsertExperiment(ctx, exp); err != nil {
		return
	}
	e.health.recordWrite()

	input := walkForwardCompletedInput{
		WalkForwardID:      completed.WalkForwardID,
		ExperimentID:       completed.ExperimentID,
		RunID:              completed.RunID,
		BestParameters:     completed.BestParameters,
		TrainingScore:      completed.TrainingScore,
		ValidationScore:    completed.ValidationScore,
		PerformanceMetrics: completed.PerformanceMetrics,
		Timestamp:          at,
	}
	result := walkForwardFromEvent(input)
	if err := e.repo.InsertWalkForwardResult(ctx, result); err != nil {
		return
	}
	e.health.recordWrite()
}

func (e *Engine) handleMonteCarlo(evt events.Event) {
	completed, ok := parseMonteCarloCompleted(evt.Payload)
	if !ok || completed.ExperimentID == "" {
		return
	}

	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	at := completed.Timestamp
	if at.IsZero() {
		at = e.clk.Now()
	}

	if err := e.repo.EnsureExperiment(ctx, ResearchExperiment{
		ExperimentID: completed.ExperimentID,
		Strategy:     "unknown",
		Symbol:       "unknown",
		Timeframe:    "unknown",
		Parameters:   []byte("{}"),
		CreatedAt:    at,
	}); err != nil {
		return
	}
	e.health.recordWrite()

	mcInput := monteCarloCompletedInput{
		SimulationID:        completed.SimulationID,
		WalkForwardID:       completed.WalkForwardID,
		ExperimentID:        completed.ExperimentID,
		Simulations:         completed.Simulations,
		ConfidenceInterval:  completed.ConfidenceInterval,
		ProbabilityOfProfit: completed.ProbabilityOfProfit,
		ProbabilityOfLoss:   completed.ProbabilityOfLoss,
		RiskOfRuin:          completed.RiskOfRuin,
		DistributionSummary: completed.DistributionSummary,
		Timestamp:           at,
	}
	result := monteCarloFromEvent(mcInput)
	if err := e.repo.InsertMonteCarloResult(ctx, result); err != nil {
		return
	}
	e.health.recordWrite()

	e.generateAndPublish(ctx, completed.ExperimentID, at)
}

func (e *Engine) generateAndPublish(ctx context.Context, experimentID string, at time.Time) {
	if e.cache.IsActive(experimentID) {
		return
	}

	readStart := time.Now()
	report, version, err := e.reports.Generate(ctx, experimentID, at)
	e.health.recordRead(time.Since(readStart))
	if err != nil {
		return
	}

	e.cache.MarkActive(ActiveReport{
		ResearchID:   report.ResearchID,
		ExperimentID: experimentID,
		StartedAt:    at,
	})
	defer e.cache.ClearActive(experimentID)

	location := ReportLocation{}
	for _, exporter := range e.exporters {
		path, exportErr := exporter.Export(report, e.cfg.ExportDirectory)
		if exportErr != nil {
			e.health.recordExport(false)
			continue
		}
		e.health.recordExport(true)
		switch exporter.Format() {
		case "json":
			location.JSONPath = path
		case "csv":
			location.CSVPath = path
		}
	}

	dbReport := ResearchReport{
		ResearchID:   report.ResearchID,
		ExperimentID: experimentID,
		Version:      version,
		JSONPath:     location.JSONPath,
		CSVPath:      location.CSVPath,
		GeneratedAt:  at,
	}
	if err := e.repo.InsertResearchReport(ctx, dbReport); err != nil {
		return
	}
	e.health.recordWrite()
	e.publishUpdated(report, location, at)
}

func (e *Engine) publishUpdated(report UnifiedReport, location ReportLocation, at time.Time) {
	out, err := events.NewEventWithClock(e.clk, events.ResearchUpdated, engineName, ResearchUpdated{
		ResearchID:     report.ResearchID,
		ExperimentID:   report.ExperimentID,
		Strategy:       report.Strategy,
		Metrics:        report.Summary,
		ReportLocation: location,
		Timestamp:      at,
	})
	if err != nil {
		return
	}
	e.bus.Publish(out)
	e.health.recordReport()
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
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.workerWG.Wait()
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

	ctx := context.Background()
	entries := int64(0)
	if e.repo != nil {
		if count, err := e.repo.CountEntries(ctx); err == nil {
			entries = count
		}
	}
	report := e.health.report(e.cfg, connected, dropped, e.repo, e.cache)
	report.Details["repository_entries"] = i64String(entries)
	return report
}

func parseOptimizationUpdate(payload []byte) (optimization.OptimizationUpdated, bool) {
	var update optimization.OptimizationUpdated
	if err := json.Unmarshal(payload, &update); err != nil {
		return optimization.OptimizationUpdated{}, false
	}
	return update, true
}

func parseWalkForwardCompleted(payload []byte) (walkforward.WalkForwardCompleted, bool) {
	var completed walkforward.WalkForwardCompleted
	if err := json.Unmarshal(payload, &completed); err != nil {
		return walkforward.WalkForwardCompleted{}, false
	}
	return completed, true
}

func parseMonteCarloCompleted(payload []byte) (montecarlo.MonteCarloCompleted, bool) {
	var completed montecarlo.MonteCarloCompleted
	if err := json.Unmarshal(payload, &completed); err != nil {
		return montecarlo.MonteCarloCompleted{}, false
	}
	return completed, true
}

func experimentIDFromParameters(serialized string) string {
	if serialized == "" {
		return ""
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
		return ""
	}
	if v, ok := raw["experiment_id"].(string); ok {
		return v
	}
	return ""
}

func parametersJSON(serialized string) []byte {
	if serialized == "" {
		return []byte("{}")
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
		return []byte("{}")
	}
	return raw
}

func stringFromParams(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}
