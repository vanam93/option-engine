package delivery

import "time"

// Status mirrors recommendation lifecycle states.
type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusActive          Status = "ACTIVE"
	StatusWatch           Status = "WATCH"
	StatusExitRecommended Status = "EXIT_RECOMMENDED"
	StatusClosed          Status = "CLOSED"
)

// Level identifies the recommendation tier.
type Level string

const (
	LevelStrongBuy Level = "STRONG_BUY"
	LevelBuy       Level = "BUY"
	LevelWatch     Level = "WATCH"
	LevelAvoid     Level = "AVOID"
)

// TimelineEventType identifies delivery timeline entry kinds.
type TimelineEventType string

const (
	TimelineCreated             TimelineEventType = "Created"
	TimelineConfidenceIncreased TimelineEventType = "Confidence Increased"
	TimelineConfidenceDecreased TimelineEventType = "Confidence Decreased"
	TimelineStatusChanged       TimelineEventType = "Status Changed"
	TimelineAlertGenerated      TimelineEventType = "Alert Generated"
	TimelineEntryTriggered      TimelineEventType = "Entry Triggered"
	TimelineExitRecommended     TimelineEventType = "Exit Recommended"
	TimelineClosed              TimelineEventType = "Closed"
	TimelineQualityEvaluated    TimelineEventType = "Quality Evaluated"
	TimelineFeedbackUpdated     TimelineEventType = "Feedback Updated"
)

// TimelineEntry records a chronological delivery event.
type TimelineEntry struct {
	Timestamp     time.Time         `json:"timestamp"`
	Event         TimelineEventType `json:"event"`
	Reason        string            `json:"reason,omitempty"`
	PreviousValue string            `json:"previous_value,omitempty"`
	NewValue      string            `json:"new_value,omitempty"`
}

// ResearchEvidence holds structured research supporting a recommendation.
type ResearchEvidence struct {
	Signal       string `json:"signal,omitempty"`
	Strategy     string `json:"strategy,omitempty"`
	Risk         string `json:"risk,omitempty"`
	Performance  string `json:"performance,omitempty"`
	Optimization string `json:"optimization,omitempty"`
	WalkForward  string `json:"walk_forward,omitempty"`
	MonteCarlo   string `json:"monte_carlo,omitempty"`
	Drawdown     string `json:"drawdown,omitempty"`
	Freshness    string `json:"freshness,omitempty"`
}

// AlertRecord captures one alert in the delivery document history.
type AlertRecord struct {
	AlertID     string    `json:"alert_id"`
	AlertType   string    `json:"alert_type"`
	Message     string    `json:"message"`
	Reason      string    `json:"reason,omitempty"`
	GeneratedAt time.Time `json:"generated_at"`
}

// QualityEvaluation summarizes the latest quality report.
type QualityEvaluation struct {
	Outcome        string    `json:"outcome,omitempty"`
	Classification string    `json:"classification,omitempty"`
	QualityScore   float64   `json:"quality_score"`
	Completed      bool      `json:"completed"`
	EvaluatedAt    time.Time `json:"evaluated_at,omitempty"`
}

// FeedbackMetrics holds dimensional learning metrics relevant to a recommendation.
type FeedbackMetrics struct {
	StrategySuccessRate   float64   `json:"strategy_success_rate,omitempty"`
	StrategyWinRate       float64   `json:"strategy_win_rate,omitempty"`
	SymbolSuccessRate     float64   `json:"symbol_success_rate,omitempty"`
	TimeframeWinRate      float64   `json:"timeframe_win_rate,omitempty"`
	ConfidenceSuccessRate float64   `json:"confidence_success_rate,omitempty"`
	PlatformSuccessRate   float64   `json:"platform_success_rate,omitempty"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
}

// HistoricalPerformance holds historical learning context for a recommendation.
type HistoricalPerformance struct {
	StrategyAverageReturn      float64   `json:"strategy_average_return,omitempty"`
	SymbolAverageReturn        float64   `json:"symbol_average_return,omitempty"`
	TimeframeAverageReturn     float64   `json:"timeframe_average_return,omitempty"`
	PlatformAverageReturn      float64   `json:"platform_average_return,omitempty"`
	PlatformAverageQuality     float64   `json:"platform_average_quality,omitempty"`
	PlatformConfidenceAccuracy float64   `json:"platform_confidence_accuracy,omitempty"`
	UpdatedAt                  time.Time `json:"updated_at,omitempty"`
}

// ValidationResult summarizes validation outcome when available.
type ValidationResult struct {
	Status           string   `json:"status,omitempty"`
	RejectionReasons []string `json:"rejection_reasons,omitempty"`
}

// DeliveryDocument is the consolidated read model for one recommendation.
type DeliveryDocument struct {
	RecommendationID           string                 `json:"recommendation_id"`
	Symbol                     string                 `json:"symbol"`
	Timeframe                  string                 `json:"timeframe"`
	Strategy                   string                 `json:"strategy"`
	Recommendation             string                 `json:"recommendation"`
	RecommendationState        string                 `json:"recommendation_state"`
	CurrentStatus              Status                 `json:"current_status"`
	CurrentConfidence          float64                `json:"current_confidence"`
	CurrentRecommendationLevel Level                  `json:"current_recommendation_level"`
	Timeline                   []TimelineEntry        `json:"timeline"`
	ResearchSummary            string                 `json:"research_summary,omitempty"`
	IntelligenceSummary        string                 `json:"intelligence_summary,omitempty"`
	Evidence                   ResearchEvidence       `json:"evidence"`
	AlertHistory               []AlertRecord          `json:"alert_history"`
	QualityEvaluation          *QualityEvaluation     `json:"quality_evaluation,omitempty"`
	FeedbackMetrics            *FeedbackMetrics       `json:"feedback_metrics,omitempty"`
	HistoricalPerformance      *HistoricalPerformance `json:"historical_performance,omitempty"`
	EntryPrice                 float64                `json:"entry_price"`
	LatestPrice                float64                `json:"latest_price"`
	High                       float64                `json:"high"`
	Low                        float64                `json:"low"`
	CurrentReturn              float64                `json:"current_return"`
	HoldingTime                time.Duration          `json:"holding_time"`
	CurrentPnL                 float64                `json:"current_pnl"`
	MaximumFavorableExcursion  float64                `json:"maximum_favorable_excursion"`
	MaximumAdverseExcursion    float64                `json:"maximum_adverse_excursion"`
	ValidationResult           *ValidationResult      `json:"validation_result,omitempty"`
	ScannerMatches             []string               `json:"scanner_matches,omitempty"`
	OpportunityRank            int                    `json:"opportunity_rank,omitempty"`
	OptimizationScore          float64                `json:"optimization_score,omitempty"`
	WalkForwardResult          float64                `json:"walk_forward_result,omitempty"`
	MonteCarloResult           float64                `json:"monte_carlo_result,omitempty"`
	CreatedAt                  time.Time              `json:"created_at"`
	UpdatedAt                  time.Time              `json:"updated_at"`
	ClosedAt                   *time.Time             `json:"closed_at,omitempty"`
}

// Filter selects delivery documents for list queries.
type Filter struct {
	Symbol           string
	Strategy         string
	Timeframe        string
	Status           Status
	Level            Level
	ConfidenceMin    float64
	ConfidenceBucket string
	CreatedAfter     time.Time
	UpdatedAfter     time.Time
	Limit            int
}
