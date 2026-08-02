package query

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/alerts"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
	"github.com/vanam-gangireddy/option-engine/internal/research"
	"github.com/vanam-gangireddy/option-engine/internal/scanner"
)

// Filter holds common query parameters.
type Filter struct {
	Symbol        string
	Strategy      string
	Timeframe     string
	Status        string
	ConfidenceMin float64
	Limit         int
	Offset        int
}

// Pagination describes list response paging metadata.
type Pagination struct {
	Limit   int `json:"limit"`
	Offset  int `json:"offset"`
	Total   int `json:"total"`
	HasMore bool `json:"has_more"`
}

// Metadata wraps query responses.
type Metadata struct {
	Timestamp time.Time `json:"timestamp"`
	Filters   Filter    `json:"filters"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// ListResponse is the standard list envelope.
type ListResponse[T any] struct {
	Metadata Metadata `json:"metadata"`
	Data     []T      `json:"data"`
}

// ItemResponse is the standard single-item envelope.
type ItemResponse[T any] struct {
	Metadata Metadata `json:"metadata"`
	Data     T        `json:"data"`
}

// RecommendationView is a query-layer recommendation DTO.
type RecommendationView struct {
	RecommendationID string                        `json:"recommendation_id"`
	Symbol           string                        `json:"symbol"`
	Timeframe        string                        `json:"timeframe"`
	Strategy         string                        `json:"strategy"`
	CurrentStatus    recommendationstate.Status    `json:"current_status"`
	Confidence       float64                       `json:"confidence"`
	CreatedAt        time.Time                     `json:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at"`
	ClosedAt         *time.Time                    `json:"closed_at,omitempty"`
}

// TimelineResponse wraps recommendation timeline entries.
type TimelineResponse struct {
	Metadata   Metadata                           `json:"metadata"`
	ID         string                             `json:"recommendation_id"`
	Timeline   []recommendationstate.TimelineEntry `json:"timeline"`
}

// AlertView is a query-layer alert DTO.
type AlertView = alerts.AlertGenerated

// OpportunityResponse wraps opportunity snapshot data.
type OpportunityResponse struct {
	Metadata Metadata                    `json:"metadata"`
	Snapshot opportunity.OpportunitySnapshot `json:"data"`
}

// ScannerResponse wraps scanner snapshot data.
type ScannerResponse struct {
	Metadata Metadata              `json:"metadata"`
	Snapshot scanner.ScannerSnapshot `json:"data"`
}

// PerformanceResponse wraps performance snapshot data.
type PerformanceResponse struct {
	Metadata Metadata                      `json:"metadata"`
	Snapshot performance.PerformanceSnapshot `json:"data"`
}

// OptimizationResponse wraps optimization snapshot data.
type OptimizationResponse struct {
	Metadata Metadata                   `json:"metadata"`
	Snapshot optimization.StateSnapshot `json:"data"`
}

// ResearchResponse wraps a research bundle.
type ResearchResponse struct {
	Metadata Metadata              `json:"metadata"`
	Bundle   research.ResearchBundle `json:"data"`
}

// IntelligenceHealthResponse aggregates intelligence component health.
type IntelligenceHealthResponse struct {
	Metadata   Metadata        `json:"metadata"`
	Components []health.Report `json:"components"`
}

func toRecommendationView(rec recommendationstate.Recommendation) RecommendationView {
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
	}
}

func paginate[T any](items []T, limit, offset int) ([]T, Pagination) {
	total := len(items)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := items[offset:end]
	return page, Pagination{
		Limit:   limit,
		Offset:  offset,
		Total:   total,
		HasMore: end < total,
	}
}
