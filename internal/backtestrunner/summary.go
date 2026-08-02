package backtestrunner

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/delivery"
	"github.com/vanam-gangireddy/option-engine/internal/feedback"
)

// SessionSummary is the immutable aggregate output of a completed backtest session.
type SessionSummary struct {
	BacktestID                string             `json:"backtest_id"`
	RecommendationsGenerated  int                `json:"recommendations_generated"`
	RecommendationsClosed     int                `json:"recommendations_closed"`
	BuyCount                  int                `json:"buy_count"`
	WatchCount                int                `json:"watch_count"`
	AvoidCount                int                `json:"avoid_count"`
	AverageConfidence         float64            `json:"average_confidence"`
	HighestConfidence         float64            `json:"highest_confidence"`
	LowestConfidence          float64            `json:"lowest_confidence"`
	AverageHoldingTime        time.Duration      `json:"average_holding_time"`
	BestRecommendation        string             `json:"best_recommendation,omitempty"`
	WorstRecommendation       string             `json:"worst_recommendation,omitempty"`
	AverageReturn             float64            `json:"average_return"`
	WinRate                   float64            `json:"win_rate"`
	LossRate                  float64            `json:"loss_rate"`
	QualityDistribution       map[string]int     `json:"quality_distribution"`
	FeedbackSummary           FeedbackSummary    `json:"feedback_summary"`
	StrategyDistribution      map[string]int     `json:"strategy_distribution"`
	SymbolDistribution        map[string]int     `json:"symbol_distribution"`
	TimeframeDistribution     map[string]int     `json:"timeframe_distribution"`
	AlertsGenerated           int                `json:"alerts_generated"`
	ResearchReportsGenerated  int                `json:"research_reports_generated"`
	OptimizationRuns          int                `json:"optimization_runs"`
	WalkForwardRuns           int                `json:"walk_forward_runs"`
	MonteCarloRuns            int                `json:"monte_carlo_runs"`
	GeneratedAt               time.Time          `json:"generated_at"`
}

// FeedbackSummary captures platform feedback at session completion.
type FeedbackSummary struct {
	TotalRecommendations int     `json:"total_recommendations"`
	SuccessRate          float64 `json:"success_rate"`
	WinRate              float64 `json:"win_rate"`
	AverageReturn        float64 `json:"average_return"`
	AverageQuality       float64 `json:"average_quality"`
	ConfidenceAccuracy   float64 `json:"confidence_accuracy"`
}

// CollectorSnapshot holds session-scoped event aggregates.
type CollectorSnapshot struct {
	Documents            map[string]delivery.DeliveryDocument
	ResearchReports      int
	AlertsGenerated      int
	OptimizationRuns     int
	WalkForwardRuns      int
	MonteCarloRuns       int
	Feedback             *feedback.RecommendationFeedbackUpdated
}

// BuildSummary aggregates delivery documents and collector metrics into a session summary.
func BuildSummary(backtestID string, docs []delivery.DeliveryDocument, snap CollectorSnapshot, at time.Time) SessionSummary {
	summary := SessionSummary{
		BacktestID:               backtestID,
		QualityDistribution:      make(map[string]int),
		StrategyDistribution:     make(map[string]int),
		SymbolDistribution:       make(map[string]int),
		TimeframeDistribution:    make(map[string]int),
		ResearchReportsGenerated: snap.ResearchReports,
		AlertsGenerated:          snap.AlertsGenerated,
		OptimizationRuns:         snap.OptimizationRuns,
		WalkForwardRuns:          snap.WalkForwardRuns,
		MonteCarloRuns:           snap.MonteCarloRuns,
		GeneratedAt:              at,
	}

	if snap.Feedback != nil {
		summary.FeedbackSummary = FeedbackSummary{
			TotalRecommendations: snap.Feedback.Overall.TotalRecommendations,
			SuccessRate:          snap.Feedback.Overall.SuccessRate,
			WinRate:              snap.Feedback.Overall.WinRate,
			AverageReturn:        snap.Feedback.Overall.AverageReturn,
			AverageQuality:       snap.Feedback.Overall.AverageQuality,
			ConfidenceAccuracy:   snap.Feedback.Overall.ConfidenceAccuracy,
		}
	}

	if len(docs) == 0 {
		return summary
	}

	summary.RecommendationsGenerated = len(docs)

	var (
		confidenceSum   float64
		returnSum       float64
		holdingSum      time.Duration
		holdingCount    int
		wins            int
		losses          int
		bestID          string
		worstID         string
		bestReturn      float64
		worstReturn     float64
		highestConf     float64
		lowestConf      float64
		firstConfidence = true
	)

	for _, doc := range docs {
		if doc.CurrentStatus == delivery.StatusClosed {
			summary.RecommendationsClosed++
		}

		switch doc.CurrentRecommendationLevel {
		case delivery.LevelBuy, delivery.LevelStrongBuy:
			summary.BuyCount++
		case delivery.LevelWatch:
			summary.WatchCount++
		case delivery.LevelAvoid:
			summary.AvoidCount++
		}

		confidenceSum += doc.CurrentConfidence
		if firstConfidence || doc.CurrentConfidence > highestConf {
			highestConf = doc.CurrentConfidence
		}
		if firstConfidence || doc.CurrentConfidence < lowestConf {
			lowestConf = doc.CurrentConfidence
		}
		firstConfidence = false

		if doc.HoldingTime > 0 {
			holdingSum += doc.HoldingTime
			holdingCount++
		}

		returnSum += doc.CurrentReturn
		if doc.CurrentReturn > bestReturn || bestID == "" {
			bestReturn = doc.CurrentReturn
			bestID = doc.RecommendationID
		}
		if doc.CurrentReturn < worstReturn || worstID == "" {
			worstReturn = doc.CurrentReturn
			worstID = doc.RecommendationID
		}

		if doc.QualityEvaluation != nil {
			class := doc.QualityEvaluation.Classification
			if class == "" {
				class = doc.QualityEvaluation.Outcome
			}
			if class != "" {
				summary.QualityDistribution[class]++
			}
			switch doc.QualityEvaluation.Outcome {
			case "SUCCESS":
				wins++
			case "FAILED":
				losses++
			}
		}

		if doc.Strategy != "" {
			summary.StrategyDistribution[doc.Strategy]++
		}
		if doc.Symbol != "" {
			summary.SymbolDistribution[doc.Symbol]++
		}
		if doc.Timeframe != "" {
			summary.TimeframeDistribution[doc.Timeframe]++
		}
	}

	summary.AverageConfidence = confidenceSum / float64(len(docs))
	summary.HighestConfidence = highestConf
	summary.LowestConfidence = lowestConf
	if holdingCount > 0 {
		summary.AverageHoldingTime = holdingSum / time.Duration(holdingCount)
	}
	summary.AverageReturn = returnSum / float64(len(docs))
	summary.BestRecommendation = bestID
	summary.WorstRecommendation = worstID

	evaluated := wins + losses
	if evaluated > 0 {
		summary.WinRate = float64(wins) / float64(evaluated)
		summary.LossRate = float64(losses) / float64(evaluated)
	}

	return summary
}
