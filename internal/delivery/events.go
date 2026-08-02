package delivery

import "time"

// RecommendationDeliveryUpdated is published on recommendation.delivery.updated.
type RecommendationDeliveryUpdated struct {
	RecommendationID string           `json:"recommendation_id"`
	Symbol           string           `json:"symbol"`
	Timeframe        string           `json:"timeframe"`
	Strategy         string           `json:"strategy"`
	Document         DeliveryDocument `json:"document"`
	GeneratedAt      time.Time        `json:"generated_at"`
}

// StateInput mirrors recommendation.state.updated payload fields.
type StateInput struct {
	RecommendationID     string
	Symbol               string
	Timeframe            string
	Strategy             string
	Recommendation       Level
	CurrentStatus        Status
	Confidence           float64
	LatestTimelineEntry  stateTimelineEntry
	Summary              string
	Reasons              []string
	SupportingIndicators []string
	SupportingStrategies []string
	OptimizationSummary  string
	WalkForwardSummary   string
	MonteCarloSummary    string
	Components           map[string]float64
	ValidationStatus     string
	RejectionReasons     []string
	ScannerMatches       []string
	OpportunityRank      int
}

type stateTimelineEntry struct {
	Timestamp     time.Time
	Event         string
	Reason        string
	PreviousValue string
	NewValue      string
}

// IntelligenceInput mirrors recommendation.intelligence.updated payload fields.
type IntelligenceInput struct {
	RecommendationID string
	Symbol           string
	Timeframe        string
	Strategy         string
	Document         intelligenceDocumentInput
	GeneratedAt      time.Time
}

type intelligenceDocumentInput struct {
	RecommendationLevel        Level
	Confidence                 float64
	CurrentStatus              Status
	CurrentRecommendationState string
	ResearchSummary            string
	DecisionSummary            string
	Explanation                string
	ResearchEvidence           ResearchEvidence
	ConfidenceBreakdown        map[string]float64
}

// QualityInput mirrors recommendation.quality.updated payload fields.
type QualityInput struct {
	RecommendationID string
	Symbol           string
	Timeframe        string
	Strategy         string
	Report           qualityReportInput
	GeneratedAt      time.Time
}

type qualityReportInput struct {
	RecommendationLevel Level
	Confidence          float64
	CurrentStatus       Status
	Outcome             string
	Classification      string
	QualityScore        float64
	Completed           bool
	EvaluatedAt         time.Time
	EntryPrice          float64
	LatestPrice         float64
	HighestPrice        float64
	LowestPrice         float64
	PercentageReturn    float64
	AbsoluteReturn      float64
	HoldingDuration     time.Duration
	MFE                 float64
	MAE                 float64
}

// FeedbackInput mirrors recommendation.feedback.updated payload fields.
type FeedbackInput struct {
	Overall               overallStatsInput
	Strategies            []strategyStatsInput
	Symbols               []symbolStatsInput
	Timeframes            []timeframeStatsInput
	ConfidenceCalibration []confidenceBucketInput
	Timestamp             time.Time
}

type overallStatsInput struct {
	SuccessRate        float64 `json:"success_rate"`
	AverageReturn      float64 `json:"average_return"`
	AverageQuality     float64 `json:"average_quality"`
	ConfidenceAccuracy float64 `json:"confidence_accuracy"`
}

type strategyStatsInput struct {
	Strategy      string  `json:"strategy"`
	SuccessRate   float64 `json:"success_rate"`
	WinRate       float64 `json:"win_rate"`
	AverageReturn float64 `json:"average_return"`
}

type symbolStatsInput struct {
	Symbol        string  `json:"symbol"`
	AverageReturn float64 `json:"average_return"`
}

type timeframeStatsInput struct {
	Timeframe     string  `json:"timeframe"`
	WinRate       float64 `json:"win_rate"`
	AverageReturn float64 `json:"average_return"`
}

type confidenceBucketInput struct {
	Label       string  `json:"label"`
	LowerBound  float64 `json:"lower_bound"`
	UpperBound  float64 `json:"upper_bound"`
	SuccessRate float64 `json:"success_rate"`
}

// AlertInput mirrors alert.generated payload fields.
type AlertInput struct {
	AlertID          string
	RecommendationID string
	Symbol           string
	Timeframe        string
	AlertType        string
	CurrentStatus    Status
	Confidence       float64
	Message          string
	Reason           string
	GeneratedAt      time.Time
}
