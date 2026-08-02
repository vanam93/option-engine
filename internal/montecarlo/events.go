package montecarlo

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// SimulationStatus describes Monte Carlo job lifecycle state.
type SimulationStatus string

const (
	SimulationStatusActive    SimulationStatus = "ACTIVE"
	SimulationStatusCompleted SimulationStatus = "COMPLETED"
	SimulationStatusFailed    SimulationStatus = "FAILED"
)

// ConfidenceInterval holds lower/upper bounds for a confidence level.
type ConfidenceInterval struct {
	Level  float64 `json:"level"`
	Lower  float64 `json:"lower"`
	Upper  float64 `json:"upper"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
}

// DistributionSummary aggregates return and drawdown statistics.
type DistributionSummary struct {
	MeanReturn       float64 `json:"mean_return"`
	MedianReturn     float64 `json:"median_return"`
	StdDevReturn     float64 `json:"std_dev_return"`
	WorstDrawdown    float64 `json:"worst_drawdown"`
	BestDrawdown     float64 `json:"best_drawdown"`
	MeanMaxDrawdown  float64 `json:"mean_max_drawdown"`
	MedianMaxDrawdown float64 `json:"median_max_drawdown"`
	ReturnHistogram  []float64 `json:"return_histogram,omitempty"`
}

// SimulationResult stores the outcome of a completed Monte Carlo batch.
type SimulationResult struct {
	SimulationID       string
	WalkForwardID      string
	ExperimentID       string
	Simulations        int
	ConfidenceInterval ConfidenceInterval
	ProbabilityOfProfit float64
	ProbabilityOfLoss  float64
	RiskOfRuin         float64
	DistributionSummary DistributionSummary
	CompletedAt        time.Time
}

// MonteCarloCompleted is the payload published on montecarlo.completed events.
type MonteCarloCompleted struct {
	SimulationID        string              `json:"simulation_id"`
	WalkForwardID       string              `json:"walkforward_id"`
	ExperimentID        string              `json:"experiment_id"`
	Simulations         int                 `json:"simulations"`
	ConfidenceInterval  ConfidenceInterval  `json:"confidence_interval"`
	ProbabilityOfProfit float64             `json:"probability_of_profit"`
	ProbabilityOfLoss   float64             `json:"probability_of_loss"`
	RiskOfRuin          float64             `json:"risk_of_ruin"`
	DistributionSummary DistributionSummary `json:"distribution_summary"`
	Timestamp           time.Time           `json:"timestamp"`
}

// StateSnapshot is an immutable read model of Monte Carlo state.
type StateSnapshot struct {
	SimulationID string
	Active       []SimulationRecord
	Completed    []SimulationResult
}

// SimulationRecord tracks simulation lifecycle in the cache.
type SimulationRecord struct {
	SimulationID  string
	WalkForwardID string
	ExperimentID  string
	Status        SimulationStatus
}

// WalkForwardInput carries trade data extracted from walkforward.completed.
type WalkForwardInput struct {
	WalkForwardID string
	ExperimentID  string
	RunID         string
	Metrics       optimization.EvaluationMetrics
	Timestamp     time.Time
}
