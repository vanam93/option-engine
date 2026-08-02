package aicontext

import "time"

// StudyAIContextCompleted is published on study.ai.context.completed.
type StudyAIContextCompleted struct {
	ContextID        string    `json:"context_id"`
	ReportID         string    `json:"report_id"`
	StudyID          string    `json:"study_id"`
	ResearchVersion  string    `json:"research_version"`
	CompletedAt      time.Time `json:"completed_at"`
}
