package api

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/alerts"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

// Filter holds common query parameters for list and snapshot endpoints.
type Filter struct {
	Symbol        string    `json:"symbol,omitempty"`
	Strategy      string    `json:"strategy,omitempty"`
	Timeframe     string    `json:"timeframe,omitempty"`
	Status        string    `json:"status,omitempty"`
	ConfidenceMin float64   `json:"confidence_min,omitempty"`
	From          time.Time `json:"from,omitempty"`
	To            time.Time `json:"to,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	Offset        int       `json:"offset,omitempty"`
	Page          int       `json:"page,omitempty"`
	Sort          string    `json:"sort,omitempty"`
	Order         string    `json:"order,omitempty"`
}

// RecommendationView is the API-layer recommendation DTO.
type RecommendationView struct {
	RecommendationID string                     `json:"recommendation_id"`
	Symbol           string                     `json:"symbol"`
	Timeframe        string                     `json:"timeframe"`
	Strategy         string                     `json:"strategy"`
	CurrentStatus    recommendationstate.Status `json:"current_status"`
	Confidence       float64                    `json:"confidence"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
	ClosedAt         *time.Time                 `json:"closed_at,omitempty"`
	Source           string                     `json:"source,omitempty"`
}

// TimelineView wraps recommendation timeline entries.
type TimelineView struct {
	RecommendationID string                              `json:"recommendation_id"`
	Timeline         []recommendationstate.TimelineEntry `json:"timeline"`
}

// AlertView is the API-layer alert DTO.
type AlertView = alerts.AlertGenerated

// OpportunityView wraps the current opportunity snapshot.
type OpportunityView struct {
	Snapshot opportunity.OpportunitySnapshot `json:"snapshot"`
}

// PerformanceView wraps the performance analytics snapshot.
type PerformanceView struct {
	Snapshot performance.PerformanceSnapshot `json:"snapshot"`
}

// OptimizationView wraps persisted optimization data for an experiment.
type OptimizationView struct {
	Experiment   research.ResearchExperiment   `json:"experiment"`
	Optimization []research.OptimizationResult `json:"optimization"`
}

// ResearchView wraps a full research bundle.
type ResearchView = research.ResearchBundle

func toRecommendationView(rec recommendationstate.Recommendation, source string) RecommendationView {
	return RecommendationView{
		RecommendationID: rec.RecommendationID,
		Symbol:           rec.Symbol,
		Timeframe:        rec.Timeframe,
		Strategy:         rec.Strategy,
		CurrentStatus:    rec.CurrentStatus,
		Confidence:       rec.Confidence,
		CreatedAt:        rec.CreatedAt,
		UpdatedAt:        rec.UpdatedAt,
		ClosedAt:         rec.ClosedAt,
		Source:           source,
	}
}
