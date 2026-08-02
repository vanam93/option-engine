package experiments

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// RunStatus describes experiment run lifecycle state.
type RunStatus string

const (
	RunStatusCreated   RunStatus = "CREATED"
	RunStatusQueued    RunStatus = "QUEUED"
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusFailed    RunStatus = "FAILED"
)

// ParameterSet holds sweep parameters for a single experiment run.
type ParameterSet map[string]any

// ExperimentRun is a single backtest execution within an experiment batch.
type ExperimentRun struct {
	ExperimentID string
	RunID        string
	BacktestID   string
	Strategy     string
	Symbol       string
	Timeframe    string
	Parameters   ParameterSet
	Status       RunStatus
}

// RunResult stores optimization outcome for a completed run.
type RunResult struct {
	RunID             string
	ExperimentID      string
	Strategy          string
	Parameters        ParameterSet
	OptimizationScore float64
	Rank              int
	Metrics           optimization.EvaluationMetrics
	CompletedAt       time.Time
}

// ExperimentCompleted is the payload published on experiment.completed events.
type ExperimentCompleted struct {
	ExperimentID      string                         `json:"experiment_id"`
	RunID             string                         `json:"run_id"`
	Strategy          string                         `json:"strategy"`
	Parameters        ParameterSet                   `json:"parameters"`
	OptimizationScore float64                        `json:"optimization_score"`
	Rank              int                            `json:"rank"`
	Metrics           optimization.EvaluationMetrics `json:"metrics"`
	Timestamp         time.Time                      `json:"timestamp"`
}

// StateSnapshot is an immutable read model of experiment state.
type StateSnapshot struct {
	ExperimentID string
	Runs         []ExperimentRun
	Completed    []RunResult
	Rankings     []RunResult
}
