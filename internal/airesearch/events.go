package airesearch

import "time"

// StudyAICompleted is published on study.ai.completed.
type StudyAICompleted struct {
	ReportID         string    `json:"report_id"`
	StudyID          string    `json:"study_id"`
	ResearchVersion  string    `json:"research_version"`
	Analyzer         string    `json:"analyzer"`
	ExecutiveSummary string    `json:"executive_summary"`
	OverallVerdict   string    `json:"overall_verdict"`
	CompletedAt      time.Time `json:"completed_at"`
}
