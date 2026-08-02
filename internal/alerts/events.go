package alerts

import "time"

// AlertType identifies the kind of lifecycle alert.
type AlertType string

const (
	AlertRecommendationCreated AlertType = "RECOMMENDATION_CREATED"
	AlertConfidenceIncreased   AlertType = "CONFIDENCE_INCREASED"
	AlertConfidenceDecreased   AlertType = "CONFIDENCE_DECREASED"
	AlertStatusChanged         AlertType = "STATUS_CHANGED"
	AlertEntryZoneReached      AlertType = "ENTRY_ZONE_REACHED"
	AlertExitRecommended       AlertType = "EXIT_RECOMMENDED"
	AlertRecommendationClosed  AlertType = "RECOMMENDATION_CLOSED"
)

// Status mirrors recommendation lifecycle states consumed from state updates.
type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusActive          Status = "ACTIVE"
	StatusWatch           Status = "WATCH"
	StatusExitRecommended Status = "EXIT_RECOMMENDED"
	StatusClosed          Status = "CLOSED"
)

// TimelineEntry mirrors the latest timeline entry from recommendation.state.updated.
type TimelineEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Event         string    `json:"event"`
	Reason        string    `json:"reason"`
	PreviousValue string    `json:"previous_value"`
	NewValue      string    `json:"new_value"`
}

// StateUpdate mirrors the recommendation.state.updated payload consumed by the engine.
type StateUpdate struct {
	RecommendationID    string
	Symbol              string
	Timeframe           string
	Strategy            string
	CurrentStatus       Status
	Confidence          float64
	LatestTimelineEntry TimelineEntry
	Summary             string
}

// AlertGenerated is the payload published on alert.generated events.
type AlertGenerated struct {
	AlertID          string    `json:"alert_id"`
	RecommendationID string    `json:"recommendation_id"`
	Symbol           string    `json:"symbol"`
	Timeframe        string    `json:"timeframe"`
	AlertType        AlertType `json:"alert_type"`
	CurrentStatus    Status    `json:"current_status"`
	Confidence       float64   `json:"confidence"`
	Message          string    `json:"message"`
	Reason           string    `json:"reason"`
	GeneratedAt      time.Time `json:"generated_at"`
}

// candidateAlert is an internal alert before deduplication and ID assignment.
type candidateAlert struct {
	AlertType AlertType
	Message   string
	Reason    string
}
