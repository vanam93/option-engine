package feedback

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

// Engine consumes recommendation.quality.updated and publishes recommendation.feedback.updated.
type Engine struct {
	cfg     Config
	bus     ports.EventBus
	clk     clock.Clock
	cache   *Cache
	learner *Learner
	health  healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a recommendation feedback engine.
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
	cache := NewCache(cfg)
	return &Engine{
		cfg:     cfg,
		bus:     bus,
		clk:     clk,
		cache:   cache,
		learner: NewLearner(cache.aggregator),
	}, nil
}

// Start subscribes to quality events before the consumer starts.
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
		return evt.Type == events.RecommendationQualityUpdated
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
	start := time.Now()
	input, ok := parseQualityInput(evt.Payload)
	if !ok {
		e.health.recordMalformed()
		return
	}
	if !IsLearnable(input) {
		return
	}

	at := evt.Timestamp.UTC()
	if !input.EvaluatedAt.IsZero() {
		at = input.EvaluatedAt.UTC()
	}
	input.EvaluatedAt = at

	snapshot, recorded := e.cache.Record(input, e.learner)
	if !recorded {
		return
	}

	e.health.recordProcessed(time.Since(start))
	e.publish(snapshot)
}

func (e *Engine) publish(snapshot FeedbackSnapshot) {
	out := RecommendationFeedbackUpdated{
		Overall:               snapshot.Overall,
		Strategies:            snapshot.Strategies,
		Symbols:               snapshot.Symbols,
		Timeframes:            snapshot.Timeframes,
		ConfidenceCalibration: snapshot.ConfidenceCalibration,
		Rolling:               snapshot.Rolling,
		Timestamp:             snapshot.Timestamp,
		Version:               snapshot.Version,
	}
	evt, err := events.NewEventWithClock(e.clk, events.RecommendationFeedbackUpdated, engineName, out)
	if err != nil {
		e.health.recordPublishFailure()
		return
	}
	e.bus.Publish(evt)
	e.health.recordFeedbackGenerated()
}

// GetSnapshot returns the latest feedback snapshot.
func (e *Engine) GetSnapshot() (FeedbackSnapshot, bool) {
	return e.cache.Snapshot()
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

	strategies, symbols, timeframes, recommendations := e.cache.Stats()
	return e.health.report(
		e.cfg,
		connected,
		dropped,
		strategies,
		symbols,
		timeframes,
		recommendations,
		e.cache.BucketCount(),
		e.cache.RollingWindowCount(),
		e.cache.Entries(),
	)
}

func parseQualityInput(payload json.RawMessage) (QualityInput, bool) {
	var raw struct {
		RecommendationID string `json:"recommendation_id"`
		Symbol           string `json:"symbol"`
		Timeframe        string `json:"timeframe"`
		Strategy         string `json:"strategy"`
		Scanner          string `json:"scanner"`
		GeneratedAt      time.Time `json:"generated_at"`
		Report           struct {
			RecommendationID          string  `json:"recommendation_id"`
			Symbol                    string  `json:"symbol"`
			Timeframe                 string  `json:"timeframe"`
			Strategy                  string  `json:"strategy"`
			RecommendationLevel       Level   `json:"recommendation_level"`
			Confidence                float64 `json:"confidence"`
			Outcome                   Outcome `json:"outcome"`
			QualityScore              float64 `json:"quality_score"`
			Completed                 bool    `json:"completed"`
			EvaluatedAt               time.Time `json:"evaluated_at"`
			OpportunityClassification string  `json:"opportunity_classification"`
			RiskApproval              string  `json:"risk_approval"`
			MarketRegime              string  `json:"market_regime"`
			RecommendationSource      string  `json:"recommendation_source"`
			QualityMetrics            struct {
				MFE             float64 `json:"mfe"`
				MAE             float64 `json:"mae"`
				MaximumDrawdown float64 `json:"maximum_drawdown"`
				ReturnPct       float64 `json:"return_pct"`
				HoldingDuration int64   `json:"holding_duration_ms"`
			} `json:"quality_metrics"`
		} `json:"report"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return QualityInput{}, false
	}

	recID := strings.TrimSpace(raw.RecommendationID)
	if recID == "" {
		recID = strings.TrimSpace(raw.Report.RecommendationID)
	}
	if recID == "" {
		return QualityInput{}, false
	}

	symbol := strings.TrimSpace(raw.Symbol)
	if symbol == "" {
		symbol = strings.TrimSpace(raw.Report.Symbol)
	}
	timeframe := strings.TrimSpace(raw.Timeframe)
	if timeframe == "" {
		timeframe = strings.TrimSpace(raw.Report.Timeframe)
	}
	strategy := strings.TrimSpace(raw.Strategy)
	if strategy == "" {
		strategy = strings.TrimSpace(raw.Report.Strategy)
	}

	evaluatedAt := raw.Report.EvaluatedAt
	if evaluatedAt.IsZero() {
		evaluatedAt = raw.GeneratedAt
	}

	return QualityInput{
		RecommendationID:          recID,
		Symbol:                      symbol,
		Timeframe:                   timeframe,
		Strategy:                    strategy,
		Scanner:                     strings.TrimSpace(raw.Scanner),
		RecommendationLevel:         raw.Report.RecommendationLevel,
		Confidence:                  raw.Report.Confidence,
		OpportunityClassification:   strings.TrimSpace(raw.Report.OpportunityClassification),
		RiskApproval:                strings.TrimSpace(raw.Report.RiskApproval),
		MarketRegime:                strings.TrimSpace(raw.Report.MarketRegime),
		RecommendationSource:        strings.TrimSpace(raw.Report.RecommendationSource),
		Outcome:                     raw.Report.Outcome,
		QualityScore:                raw.Report.QualityScore,
		ReturnPct:                   raw.Report.QualityMetrics.ReturnPct,
		MFE:                         raw.Report.QualityMetrics.MFE,
		MAE:                         raw.Report.QualityMetrics.MAE,
		MaxDrawdown:                 raw.Report.QualityMetrics.MaximumDrawdown,
		HoldingDurationMs:           raw.Report.QualityMetrics.HoldingDuration,
		Completed:                   raw.Report.Completed,
		EvaluatedAt:                 evaluatedAt,
	}, true
}
