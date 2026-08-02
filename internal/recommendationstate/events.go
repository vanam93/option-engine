package recommendationstate

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

// Status identifies the recommendation lifecycle state.
type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusActive          Status = "ACTIVE"
	StatusWatch           Status = "WATCH"
	StatusExitRecommended Status = "EXIT_RECOMMENDED"
	StatusClosed          Status = "CLOSED"
)

// TimelineEntry records a chronological state change.
type TimelineEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Event         string    `json:"event"`
	Reason        string    `json:"reason"`
	PreviousValue string    `json:"previous_value"`
	NewValue      string    `json:"new_value"`
}

// Recommendation holds the persistent lifecycle state for one recommendation.
type Recommendation struct {
	RecommendationID string    `json:"recommendation_id"`
	Symbol           string    `json:"symbol"`
	Timeframe        string    `json:"timeframe"`
	Strategy         string    `json:"strategy"`
	CurrentStatus    Status    `json:"current_status"`
	Confidence       float64   `json:"confidence"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
}

// RecommendationStateUpdated is the payload published on recommendation.state.updated events.
type RecommendationStateUpdated struct {
	RecommendationID    string        `json:"recommendation_id"`
	Symbol              string        `json:"symbol"`
	Timeframe           string        `json:"timeframe"`
	Strategy            string        `json:"strategy"`
	CurrentStatus       Status        `json:"current_status"`
	Confidence          float64       `json:"confidence"`
	LatestTimelineEntry TimelineEntry `json:"latest_timeline_entry"`
	Summary             string        `json:"summary"`
}

// InputValidated mirrors the validated.recommendation payload consumed by the engine.
type InputValidated struct {
	Symbol             string
	Timeframe          string
	Strategy           string
	Recommendation     recommendation.Level
	Confidence         float64
	ValidationStatus   string
	RejectionReasons   []string
	ValidatedAt        time.Time
}
