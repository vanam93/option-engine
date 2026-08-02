package research

import (
	"time"
)

// ResearchUpdated is the payload published on research.updated events.
type ResearchUpdated struct {
	ResearchID     string         `json:"research_id"`
	ExperimentID   string         `json:"experiment_id"`
	Strategy       string         `json:"strategy"`
	Metrics        ReportSummary  `json:"metrics"`
	ReportLocation ReportLocation `json:"report_location"`
	Timestamp      time.Time      `json:"timestamp"`
}

// ActiveReport tracks in-flight report generation per experiment.
type ActiveReport struct {
	ResearchID   string
	ExperimentID string
	StartedAt    time.Time
}
