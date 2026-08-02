package walkforward

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/experiments"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// WindowStatus describes walk-forward window lifecycle state.
type WindowStatus string

const (
	WindowStatusQueued     WindowStatus = "QUEUED"
	WindowStatusTraining   WindowStatus = "TRAINING"
	WindowStatusValidating WindowStatus = "VALIDATING"
	WindowStatusCompleted  WindowStatus = "COMPLETED"
	WindowStatusFailed     WindowStatus = "FAILED"
)

// WindowResult stores the outcome of a completed walk-forward window.
type WindowResult struct {
	WalkForwardID    string
	WindowIndex      int
	ExperimentID     string
	RunID            string
	TrainPeriod      Period
	ValidationPeriod Period
	BestParameters   experiments.ParameterSet
	TrainingScore    float64
	ValidationScore  float64
	Metrics          optimization.EvaluationMetrics
	CompletedAt      time.Time
}

// WalkForwardCompleted is the payload published on walkforward.completed events.
type WalkForwardCompleted struct {
	WalkForwardID      string                         `json:"walkforward_id"`
	ExperimentID       string                         `json:"experiment_id"`
	RunID              string                         `json:"run_id"`
	TrainPeriod        Period                         `json:"train_period"`
	ValidationPeriod   Period                         `json:"validation_period"`
	BestParameters     experiments.ParameterSet       `json:"best_parameters"`
	TrainingScore      float64                        `json:"training_score"`
	ValidationScore    float64                        `json:"validation_score"`
	PerformanceMetrics optimization.EvaluationMetrics `json:"performance_metrics"`
	Timestamp          time.Time                      `json:"timestamp"`
}

// AggregatedValidation holds cross-window summary metrics.
type AggregatedValidation struct {
	WindowCount           int                `json:"window_count"`
	MeanValidationScore   float64            `json:"mean_validation_score"`
	ScoreStdDev           float64            `json:"score_std_dev"`
	MeanTrainingScore     float64            `json:"mean_training_score"`
	TrainingValidationGap float64            `json:"training_validation_gap"`
	StabilityScore        float64            `json:"stability_score"`
	ParameterDrift        map[string]float64 `json:"parameter_drift"`
}

// StateSnapshot is an immutable read model of walk-forward state.
type StateSnapshot struct {
	WalkForwardID string
	Windows       []WindowRecord
	Completed     []WindowResult
	Aggregated    AggregatedValidation
}

// WindowRecord tracks window lifecycle in the cache.
type WindowRecord struct {
	Window Window
	Status WindowStatus
}
