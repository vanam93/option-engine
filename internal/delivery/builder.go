package delivery

import (
	"strings"
	"time"
)

// Builder incrementally assembles delivery documents from event inputs.
type Builder struct {
	fmt *Formatter
}

// NewBuilder creates a delivery document builder.
func NewBuilder() *Builder {
	return &Builder{fmt: NewFormatter()}
}

type storedDocument struct {
	document        DeliveryDocument
	hadEntryPrice   bool
	feedbackApplied bool
}

func (b *Builder) ensureStored(stored *storedDocument, id string) {
	if stored.document.RecommendationID != "" {
		return
	}
	stored.document.RecommendationID = id
}

// ApplyState merges a state update into the stored document.
func (b *Builder) ApplyState(stored *storedDocument, input StateInput, at time.Time) {
	b.ensureStored(stored, input.RecommendationID)

	doc := &stored.document
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = at
	}
	doc.Symbol = input.Symbol
	doc.Timeframe = input.Timeframe
	doc.Strategy = input.Strategy
	if input.Recommendation != "" {
		doc.Recommendation = string(input.Recommendation)
		doc.CurrentRecommendationLevel = input.Recommendation
	}
	doc.CurrentStatus = input.CurrentStatus
	doc.CurrentConfidence = input.Confidence
	doc.RecommendationState = b.fmt.StatusLabel(input.CurrentStatus)
	doc.UpdatedAt = at

	if input.CurrentStatus == StatusClosed {
		closedAt := at
		doc.ClosedAt = &closedAt
	}

	if input.ValidationStatus != "" {
		doc.ValidationResult = &ValidationResult{
			Status:           strings.ToUpper(strings.TrimSpace(input.ValidationStatus)),
			RejectionReasons: append([]string(nil), input.RejectionReasons...),
		}
	}

	if len(input.ScannerMatches) > 0 {
		doc.ScannerMatches = append([]string(nil), input.ScannerMatches...)
	}
	if input.OpportunityRank > 0 {
		doc.OpportunityRank = input.OpportunityRank
	}

	b.applyComponentScores(doc, input.Components)
	doc.Timeline = mergeStateTimeline(doc.Timeline, input.LatestTimelineEntry)
	doc.Timeline = sortTimeline(doc.Timeline)
}

// ApplyIntelligence merges an intelligence update into the stored document.
func (b *Builder) ApplyIntelligence(stored *storedDocument, input IntelligenceInput) {
	b.ensureStored(stored, input.RecommendationID)

	doc := &stored.document
	doc.Symbol = input.Symbol
	doc.Timeframe = input.Timeframe
	doc.Strategy = input.Strategy
	doc.CurrentRecommendationLevel = input.Document.RecommendationLevel
	doc.Recommendation = string(input.Document.RecommendationLevel)
	doc.CurrentConfidence = input.Document.Confidence
	doc.CurrentStatus = input.Document.CurrentStatus
	doc.RecommendationState = input.Document.CurrentRecommendationState
	if doc.RecommendationState == "" {
		doc.RecommendationState = b.fmt.StatusLabel(input.Document.CurrentStatus)
	}
	doc.ResearchSummary = input.Document.ResearchSummary
	doc.IntelligenceSummary = firstNonEmpty(input.Document.DecisionSummary, input.Document.Explanation)
	doc.Evidence = input.Document.ResearchEvidence
	doc.UpdatedAt = input.GeneratedAt

	b.applyComponentScores(doc, input.Document.ConfidenceBreakdown)
}

// ApplyQuality merges a quality update into the stored document.
func (b *Builder) ApplyQuality(stored *storedDocument, input QualityInput) {
	b.ensureStored(stored, input.RecommendationID)

	doc := &stored.document
	report := input.Report
	doc.Symbol = input.Symbol
	doc.Timeframe = input.Timeframe
	doc.Strategy = input.Strategy
	if report.RecommendationLevel != "" {
		doc.CurrentRecommendationLevel = report.RecommendationLevel
		doc.Recommendation = string(report.RecommendationLevel)
	}
	if report.Confidence > 0 {
		doc.CurrentConfidence = report.Confidence
	}
	if report.CurrentStatus != "" {
		doc.CurrentStatus = report.CurrentStatus
		doc.RecommendationState = b.fmt.StatusLabel(report.CurrentStatus)
	}

	doc.QualityEvaluation = &QualityEvaluation{
		Outcome:        report.Outcome,
		Classification: report.Classification,
		QualityScore:   report.QualityScore,
		Completed:      report.Completed,
		EvaluatedAt:    report.EvaluatedAt,
	}

	doc.EntryPrice = report.EntryPrice
	doc.LatestPrice = report.LatestPrice
	doc.High = report.HighestPrice
	doc.Low = report.LowestPrice
	doc.CurrentReturn = report.PercentageReturn
	doc.CurrentPnL = report.AbsoluteReturn
	doc.HoldingTime = report.HoldingDuration
	doc.MaximumFavorableExcursion = report.MFE
	doc.MaximumAdverseExcursion = report.MAE
	doc.UpdatedAt = input.GeneratedAt

	if report.EntryPrice > 0 && !stored.hadEntryPrice {
		stored.hadEntryPrice = true
		doc.Timeline = appendEntryTriggeredTimeline(doc.Timeline, report.EvaluatedAt, report.EntryPrice)
	}
	doc.Timeline = appendQualityTimeline(doc.Timeline, report.EvaluatedAt, report.Outcome)
	doc.Timeline = sortTimeline(doc.Timeline)
}

// ApplyFeedback merges platform feedback metrics relevant to the document.
func (b *Builder) ApplyFeedback(stored *storedDocument, input FeedbackInput) {
	doc := &stored.document
	if doc.RecommendationID == "" {
		return
	}

	metrics := &FeedbackMetrics{
		PlatformSuccessRate: input.Overall.SuccessRate,
		UpdatedAt:           input.Timestamp,
	}
	history := &HistoricalPerformance{
		PlatformAverageReturn:      input.Overall.AverageReturn,
		PlatformAverageQuality:     input.Overall.AverageQuality,
		PlatformConfidenceAccuracy: input.Overall.ConfidenceAccuracy,
		UpdatedAt:                  input.Timestamp,
	}

	for _, item := range input.Strategies {
		if item.Strategy != doc.Strategy {
			continue
		}
		metrics.StrategySuccessRate = item.SuccessRate
		metrics.StrategyWinRate = item.WinRate
		history.StrategyAverageReturn = item.AverageReturn
		break
	}
	for _, item := range input.Symbols {
		if item.Symbol != doc.Symbol {
			continue
		}
		history.SymbolAverageReturn = item.AverageReturn
		break
	}
	for _, item := range input.Timeframes {
		if item.Timeframe != doc.Timeframe {
			continue
		}
		metrics.TimeframeWinRate = item.WinRate
		history.TimeframeAverageReturn = item.AverageReturn
		break
	}
	bucket := ConfidenceBucket(doc.CurrentConfidence)
	for _, item := range input.ConfidenceCalibration {
		if item.Label != "" && item.Label != bucket {
			continue
		}
		if item.LowerBound > 0 || item.UpperBound > 0 {
			if doc.CurrentConfidence < item.LowerBound || doc.CurrentConfidence > item.UpperBound {
				continue
			}
		}
		metrics.ConfidenceSuccessRate = item.SuccessRate
		break
	}

	doc.FeedbackMetrics = metrics
	doc.HistoricalPerformance = history
	doc.UpdatedAt = input.Timestamp

	if !stored.feedbackApplied {
		stored.feedbackApplied = true
		doc.Timeline = appendFeedbackTimeline(doc.Timeline, input.Timestamp)
		doc.Timeline = sortTimeline(doc.Timeline)
	}
}

// ApplyAlert appends an alert to the document history.
func (b *Builder) ApplyAlert(stored *storedDocument, input AlertInput) {
	b.ensureStored(stored, input.RecommendationID)

	doc := &stored.document
	doc.AlertHistory = append(doc.AlertHistory, AlertRecord{
		AlertID:     input.AlertID,
		AlertType:   input.AlertType,
		Message:     input.Message,
		Reason:      input.Reason,
		GeneratedAt: input.GeneratedAt,
	})
	doc.UpdatedAt = input.GeneratedAt
	doc.Timeline = appendAlertTimeline(doc.Timeline, input)
	doc.Timeline = sortTimeline(doc.Timeline)
}

// Build returns an immutable copy of the delivery document.
func (b *Builder) Build(stored *storedDocument) DeliveryDocument {
	doc := stored.document
	doc.Timeline = append([]TimelineEntry(nil), doc.Timeline...)
	doc.AlertHistory = append([]AlertRecord(nil), doc.AlertHistory...)
	doc.ScannerMatches = append([]string(nil), doc.ScannerMatches...)
	if stored.document.ValidationResult != nil {
		vr := *stored.document.ValidationResult
		vr.RejectionReasons = append([]string(nil), vr.RejectionReasons...)
		doc.ValidationResult = &vr
	}
	if stored.document.QualityEvaluation != nil {
		qe := *stored.document.QualityEvaluation
		doc.QualityEvaluation = &qe
	}
	if stored.document.FeedbackMetrics != nil {
		fm := *stored.document.FeedbackMetrics
		doc.FeedbackMetrics = &fm
	}
	if stored.document.HistoricalPerformance != nil {
		hp := *stored.document.HistoricalPerformance
		doc.HistoricalPerformance = &hp
	}
	return doc
}

func (b *Builder) applyComponentScores(doc *DeliveryDocument, components map[string]float64) {
	if len(components) == 0 {
		return
	}
	if v, ok := components["optimization"]; ok {
		doc.OptimizationScore = v
	}
	if v, ok := components["walkforward"]; ok {
		doc.WalkForwardResult = v
	}
	if v, ok := components["walk_forward"]; ok && doc.WalkForwardResult == 0 {
		doc.WalkForwardResult = v
	}
	if v, ok := components["montecarlo"]; ok {
		doc.MonteCarloResult = v
	}
	if v, ok := components["monte_carlo"]; ok && doc.MonteCarloResult == 0 {
		doc.MonteCarloResult = v
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
