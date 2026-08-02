package delivery

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

// Engine consumes upstream recommendation events and publishes recommendation.delivery.updated.
type Engine struct {
	cfg        Config
	bus        ports.EventBus
	clk        clock.Clock
	builder    *Builder
	cache      *Cache
	repository *Repository
	health     healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a recommendation delivery engine.
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
	builder := NewBuilder()
	cache := NewCache(builder)
	return &Engine{
		cfg:        cfg,
		bus:        bus,
		clk:        clk,
		builder:    builder,
		cache:      cache,
		repository: NewRepository(cache),
	}, nil
}

// Repository returns the read-only delivery repository.
func (e *Engine) Repository() *Repository {
	return e.repository
}

// Start subscribes to upstream recommendation events before the consumer starts.
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
		return evt.Type == events.RecommendationStateUpdated ||
			evt.Type == events.RecommendationIntelligenceUpdated ||
			evt.Type == events.RecommendationQualityUpdated ||
			evt.Type == events.RecommendationFeedbackUpdated ||
			evt.Type == events.AlertGenerated
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
	at := evt.Timestamp.UTC()

	switch evt.Type {
	case events.RecommendationStateUpdated:
		input, ok := parseStateInput(evt.Payload)
		if !ok {
			return
		}
		if !input.LatestTimelineEntry.Timestamp.IsZero() {
			at = input.LatestTimelineEntry.Timestamp.UTC()
		}
		doc := e.cache.ApplyState(input, at)
		e.publish(doc, at)
	case events.RecommendationIntelligenceUpdated:
		input, ok := parseIntelligenceInput(evt.Payload)
		if !ok {
			return
		}
		if !input.GeneratedAt.IsZero() {
			at = input.GeneratedAt.UTC()
		}
		doc := e.cache.ApplyIntelligence(input)
		e.publish(doc, at)
	case events.RecommendationQualityUpdated:
		input, ok := parseQualityInput(evt.Payload)
		if !ok {
			return
		}
		if !input.GeneratedAt.IsZero() {
			at = input.GeneratedAt.UTC()
		}
		doc := e.cache.ApplyQuality(input)
		e.publish(doc, at)
	case events.RecommendationFeedbackUpdated:
		input, ok := parseFeedbackInput(evt.Payload)
		if !ok {
			return
		}
		if !input.Timestamp.IsZero() {
			at = input.Timestamp.UTC()
		}
		docs := e.cache.ApplyFeedback(input)
		for _, doc := range docs {
			e.publish(doc, at)
		}
	case events.AlertGenerated:
		input, ok := parseAlertInput(evt.Payload)
		if !ok {
			return
		}
		if !input.GeneratedAt.IsZero() {
			at = input.GeneratedAt.UTC()
		}
		doc, _ := e.cache.ApplyAlert(input)
		e.publish(doc, at)
	default:
		return
	}

	e.health.recordEvent(time.Since(start), true)
}

func (e *Engine) publish(doc DeliveryDocument, at time.Time) {
	out := RecommendationDeliveryUpdated{
		RecommendationID: doc.RecommendationID,
		Symbol:           doc.Symbol,
		Timeframe:        doc.Timeframe,
		Strategy:         doc.Strategy,
		Document:         doc,
		GeneratedAt:      at,
	}
	evt, err := events.NewEventWithClock(e.clk, events.RecommendationDeliveryUpdated, engineName, out)
	if err != nil {
		return
	}
	e.bus.Publish(evt)
}

// GetRecommendation returns a delivery document by recommendation ID.
func (e *Engine) GetRecommendation(id string) (DeliveryDocument, bool) {
	return e.repository.GetRecommendation(id)
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

	documents, active, closed, timelineEntries := e.cache.Stats()
	hits, misses := e.repository.Stats()
	return e.health.report(e.cfg, connected, dropped, documents, active, closed, timelineEntries, hits, misses)
}

func parseStateInput(payload json.RawMessage) (StateInput, bool) {
	var raw struct {
		RecommendationID    string             `json:"recommendation_id"`
		Symbol              string             `json:"symbol"`
		Timeframe           string             `json:"timeframe"`
		Strategy            string             `json:"strategy"`
		Recommendation      string             `json:"recommendation"`
		CurrentStatus       string             `json:"current_status"`
		Confidence          float64            `json:"confidence"`
		ValidationStatus    string             `json:"validation_status"`
		RejectionReasons    []string           `json:"rejection_reasons"`
		ScannerMatches      []string           `json:"scanner_matches"`
		OpportunityRank     int                `json:"opportunity_rank"`
		Components          map[string]float64 `json:"components"`
		LatestTimelineEntry struct {
			Timestamp     time.Time `json:"timestamp"`
			Event         string    `json:"event"`
			Reason        string    `json:"reason"`
			PreviousValue string    `json:"previous_value"`
			NewValue      string    `json:"new_value"`
		} `json:"latest_timeline_entry"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return StateInput{}, false
	}
	if raw.RecommendationID == "" || raw.Symbol == "" {
		return StateInput{}, false
	}

	return StateInput{
		RecommendationID: raw.RecommendationID,
		Symbol:           raw.Symbol,
		Timeframe:        raw.Timeframe,
		Strategy:         strings.TrimSpace(raw.Strategy),
		Recommendation:   Level(strings.ToUpper(strings.TrimSpace(raw.Recommendation))),
		CurrentStatus:    Status(strings.ToUpper(strings.TrimSpace(raw.CurrentStatus))),
		Confidence:       raw.Confidence,
		ValidationStatus: raw.ValidationStatus,
		RejectionReasons: raw.RejectionReasons,
		ScannerMatches:   raw.ScannerMatches,
		OpportunityRank:  raw.OpportunityRank,
		Components:       raw.Components,
		LatestTimelineEntry: stateTimelineEntry{
			Timestamp:     raw.LatestTimelineEntry.Timestamp,
			Event:         raw.LatestTimelineEntry.Event,
			Reason:        raw.LatestTimelineEntry.Reason,
			PreviousValue: raw.LatestTimelineEntry.PreviousValue,
			NewValue:      raw.LatestTimelineEntry.NewValue,
		},
	}, true
}

func parseIntelligenceInput(payload json.RawMessage) (IntelligenceInput, bool) {
	var raw struct {
		RecommendationID string    `json:"recommendation_id"`
		Symbol           string    `json:"symbol"`
		Timeframe        string    `json:"timeframe"`
		Strategy         string    `json:"strategy"`
		GeneratedAt      time.Time `json:"generated_at"`
		Document         struct {
			RecommendationLevel        string           `json:"recommendation_level"`
			Confidence                 float64          `json:"confidence"`
			CurrentStatus              string           `json:"current_status"`
			CurrentRecommendationState string           `json:"current_recommendation_state"`
			ResearchSummary            string           `json:"research_summary"`
			DecisionSummary            string           `json:"decision_summary"`
			Explanation                string           `json:"explanation"`
			ResearchEvidence           ResearchEvidence `json:"research_evidence"`
			ConfidenceBreakdown        struct {
				Signal       *float64 `json:"signal_contribution"`
				Strategy     *float64 `json:"strategy_contribution"`
				Performance  *float64 `json:"performance_contribution"`
				Optimization *float64 `json:"optimization_contribution"`
				WalkForward  *float64 `json:"walk_forward_contribution"`
				MonteCarlo   *float64 `json:"monte_carlo_contribution"`
			} `json:"confidence_breakdown"`
		} `json:"document"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return IntelligenceInput{}, false
	}
	if raw.RecommendationID == "" {
		return IntelligenceInput{}, false
	}

	components := make(map[string]float64)
	if raw.Document.ConfidenceBreakdown.Optimization != nil {
		components["optimization"] = *raw.Document.ConfidenceBreakdown.Optimization
	}
	if raw.Document.ConfidenceBreakdown.WalkForward != nil {
		components["walkforward"] = *raw.Document.ConfidenceBreakdown.WalkForward
	}
	if raw.Document.ConfidenceBreakdown.MonteCarlo != nil {
		components["montecarlo"] = *raw.Document.ConfidenceBreakdown.MonteCarlo
	}

	return IntelligenceInput{
		RecommendationID: raw.RecommendationID,
		Symbol:           raw.Symbol,
		Timeframe:        raw.Timeframe,
		Strategy:         strings.TrimSpace(raw.Strategy),
		GeneratedAt:      raw.GeneratedAt,
		Document: intelligenceDocumentInput{
			RecommendationLevel:        Level(strings.ToUpper(strings.TrimSpace(raw.Document.RecommendationLevel))),
			Confidence:                 raw.Document.Confidence,
			CurrentStatus:              Status(strings.ToUpper(strings.TrimSpace(raw.Document.CurrentStatus))),
			CurrentRecommendationState: raw.Document.CurrentRecommendationState,
			ResearchSummary:            raw.Document.ResearchSummary,
			DecisionSummary:            raw.Document.DecisionSummary,
			Explanation:                raw.Document.Explanation,
			ResearchEvidence:           raw.Document.ResearchEvidence,
			ConfidenceBreakdown:        components,
		},
	}, true
}

func parseQualityInput(payload json.RawMessage) (QualityInput, bool) {
	var raw struct {
		RecommendationID string    `json:"recommendation_id"`
		Symbol           string    `json:"symbol"`
		Timeframe        string    `json:"timeframe"`
		Strategy         string    `json:"strategy"`
		GeneratedAt      time.Time `json:"generated_at"`
		Report           struct {
			RecommendationID    string    `json:"recommendation_id"`
			RecommendationLevel string    `json:"recommendation_level"`
			Confidence          float64   `json:"confidence"`
			CurrentStatus       string    `json:"current_status"`
			Outcome             string    `json:"outcome"`
			Classification      string    `json:"classification"`
			QualityScore        float64   `json:"quality_score"`
			Completed           bool      `json:"completed"`
			EvaluatedAt         time.Time `json:"evaluated_at"`
			PriceStatistics     struct {
				EntryPrice       float64 `json:"entry_price"`
				LatestPrice      float64 `json:"latest_price"`
				HighestPrice     float64 `json:"highest_price"`
				LowestPrice      float64 `json:"lowest_price"`
				AbsoluteReturn   float64 `json:"absolute_return"`
				PercentageReturn float64 `json:"percentage_return"`
				HoldingDuration  int64   `json:"holding_duration"`
			} `json:"price_statistics"`
			QualityMetrics struct {
				MFE             float64 `json:"mfe"`
				MAE             float64 `json:"mae"`
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

	holding := time.Duration(raw.Report.PriceStatistics.HoldingDuration)
	if holding == 0 && raw.Report.QualityMetrics.HoldingDuration > 0 {
		holding = time.Duration(raw.Report.QualityMetrics.HoldingDuration) * time.Millisecond
	}
	returnPct := raw.Report.PriceStatistics.PercentageReturn
	if returnPct == 0 {
		returnPct = raw.Report.QualityMetrics.ReturnPct
	}

	return QualityInput{
		RecommendationID: recID,
		Symbol:           raw.Symbol,
		Timeframe:        raw.Timeframe,
		Strategy:         strings.TrimSpace(raw.Strategy),
		GeneratedAt:      raw.GeneratedAt,
		Report: qualityReportInput{
			RecommendationLevel: Level(strings.ToUpper(strings.TrimSpace(raw.Report.RecommendationLevel))),
			Confidence:          raw.Report.Confidence,
			CurrentStatus:       Status(strings.ToUpper(strings.TrimSpace(raw.Report.CurrentStatus))),
			Outcome:             raw.Report.Outcome,
			Classification:      raw.Report.Classification,
			QualityScore:        raw.Report.QualityScore,
			Completed:           raw.Report.Completed,
			EvaluatedAt:         raw.Report.EvaluatedAt,
			EntryPrice:          raw.Report.PriceStatistics.EntryPrice,
			LatestPrice:         raw.Report.PriceStatistics.LatestPrice,
			HighestPrice:        raw.Report.PriceStatistics.HighestPrice,
			LowestPrice:         raw.Report.PriceStatistics.LowestPrice,
			PercentageReturn:    returnPct,
			AbsoluteReturn:      raw.Report.PriceStatistics.AbsoluteReturn,
			HoldingDuration:     holding,
			MFE:                 raw.Report.QualityMetrics.MFE,
			MAE:                 raw.Report.QualityMetrics.MAE,
		},
	}, true
}

func parseFeedbackInput(payload json.RawMessage) (FeedbackInput, bool) {
	var raw struct {
		Overall struct {
			SuccessRate        float64 `json:"success_rate"`
			AverageReturn      float64 `json:"average_return"`
			AverageQuality     float64 `json:"average_quality"`
			ConfidenceAccuracy float64 `json:"confidence_accuracy"`
		} `json:"overall"`
		Strategies            []strategyStatsInput    `json:"strategies"`
		Symbols               []symbolStatsInput      `json:"symbols"`
		Timeframes            []timeframeStatsInput   `json:"timeframes"`
		ConfidenceCalibration []confidenceBucketInput `json:"confidence_calibration"`
		Timestamp             time.Time               `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return FeedbackInput{}, false
	}
	return FeedbackInput{
		Overall: overallStatsInput{
			SuccessRate:        raw.Overall.SuccessRate,
			AverageReturn:      raw.Overall.AverageReturn,
			AverageQuality:     raw.Overall.AverageQuality,
			ConfidenceAccuracy: raw.Overall.ConfidenceAccuracy,
		},
		Strategies:            raw.Strategies,
		Symbols:               raw.Symbols,
		Timeframes:            raw.Timeframes,
		ConfidenceCalibration: raw.ConfidenceCalibration,
		Timestamp:             raw.Timestamp,
	}, true
}

func parseAlertInput(payload json.RawMessage) (AlertInput, bool) {
	var raw struct {
		AlertID          string    `json:"alert_id"`
		RecommendationID string    `json:"recommendation_id"`
		Symbol           string    `json:"symbol"`
		Timeframe        string    `json:"timeframe"`
		AlertType        string    `json:"alert_type"`
		CurrentStatus    string    `json:"current_status"`
		Confidence       float64   `json:"confidence"`
		Message          string    `json:"message"`
		Reason           string    `json:"reason"`
		GeneratedAt      time.Time `json:"generated_at"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return AlertInput{}, false
	}
	if raw.RecommendationID == "" {
		return AlertInput{}, false
	}
	return AlertInput{
		AlertID:          raw.AlertID,
		RecommendationID: raw.RecommendationID,
		Symbol:           raw.Symbol,
		Timeframe:        raw.Timeframe,
		AlertType:        raw.AlertType,
		CurrentStatus:    Status(strings.ToUpper(strings.TrimSpace(raw.CurrentStatus))),
		Confidence:       raw.Confidence,
		Message:          raw.Message,
		Reason:           raw.Reason,
		GeneratedAt:      raw.GeneratedAt,
	}, true
}
