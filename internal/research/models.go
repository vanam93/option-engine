package research

import (
	"encoding/json"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/montecarlo"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// ResearchExperiment is the persisted experiment metadata record.
type ResearchExperiment struct {
	ExperimentID string          `json:"experiment_id"`
	Strategy     string          `json:"strategy"`
	Symbol       string          `json:"symbol"`
	Timeframe    string          `json:"timeframe"`
	Parameters   json.RawMessage `json:"parameters"`
	CreatedAt    time.Time       `json:"created_at"`
}

// OptimizationResult is a persisted optimization evaluation.
type OptimizationResult struct {
	ID           int64           `json:"id"`
	ExperimentID string          `json:"experiment_id"`
	Score        float64         `json:"score"`
	WinRate      float64         `json:"win_rate"`
	Expectancy   float64         `json:"expectancy"`
	ProfitFactor float64         `json:"profit_factor"`
	Drawdown     float64         `json:"drawdown"`
	Metrics      json.RawMessage `json:"metrics"`
	CreatedAt    time.Time       `json:"created_at"`
}

// WalkForwardResult is a persisted walk-forward validation window.
type WalkForwardResult struct {
	ID                 int64           `json:"id"`
	WalkForwardID      string          `json:"walkforward_id"`
	ExperimentID       string          `json:"experiment_id"`
	RunID              string          `json:"run_id"`
	TrainScore         float64         `json:"train_score"`
	ValidationScore    float64         `json:"validation_score"`
	ParameterSet       json.RawMessage `json:"parameter_set"`
	PerformanceMetrics json.RawMessage `json:"performance_metrics"`
	CreatedAt          time.Time       `json:"created_at"`
}

// MonteCarloResult is a persisted Monte Carlo simulation batch.
type MonteCarloResult struct {
	SimulationID        string          `json:"simulation_id"`
	WalkForwardID       string          `json:"walkforward_id"`
	ExperimentID        string          `json:"experiment_id"`
	Simulations         int             `json:"simulations"`
	ConfidenceInterval  json.RawMessage `json:"confidence_interval"`
	ProbabilityOfProfit float64         `json:"probability_of_profit"`
	ProbabilityOfLoss   float64         `json:"probability_of_loss"`
	RiskOfRuin          float64         `json:"risk_of_ruin"`
	Distribution        json.RawMessage `json:"distribution"`
	CreatedAt           time.Time       `json:"created_at"`
}

// ResearchReport tracks exported report artifacts.
type ResearchReport struct {
	ResearchID   string    `json:"research_id"`
	ExperimentID string    `json:"experiment_id"`
	Version      int       `json:"version"`
	JSONPath     string    `json:"json_path,omitempty"`
	CSVPath      string    `json:"csv_path,omitempty"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// QueryFilter selects research experiments by dimension.
type QueryFilter struct {
	ExperimentID string
	Strategy     string
	Symbol       string
	Timeframe    string
}

// ResearchBundle aggregates all persisted artifacts for an experiment.
type ResearchBundle struct {
	Experiment   ResearchExperiment    `json:"experiment"`
	Optimization []OptimizationResult  `json:"optimization"`
	WalkForward  []WalkForwardResult   `json:"walkforward"`
	MonteCarlo   []MonteCarloResult    `json:"montecarlo"`
	Reports      []ResearchReport      `json:"reports"`
}

// UnifiedReport is the in-memory report model generated from PostgreSQL data.
type UnifiedReport struct {
	ResearchID   string                `json:"research_id"`
	ExperimentID string                `json:"experiment_id"`
	Version      int                   `json:"version"`
	Strategy     string                `json:"strategy"`
	Symbol       string                `json:"symbol"`
	Timeframe    string                `json:"timeframe"`
	Parameters   json.RawMessage       `json:"parameters"`
	Optimization []OptimizationResult  `json:"optimization"`
	WalkForward  []WalkForwardResult   `json:"walkforward"`
	MonteCarlo   []MonteCarloResult    `json:"montecarlo"`
	Summary      ReportSummary         `json:"summary"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

// ReportSummary holds headline metrics for dashboards and events.
type ReportSummary struct {
	BestScore           float64 `json:"best_score"`
	LatestValidation    float64 `json:"latest_validation_score"`
	ProbabilityOfProfit float64 `json:"probability_of_profit"`
	RiskOfRuin          float64 `json:"risk_of_ruin"`
}

// ReportLocation describes exported file paths.
type ReportLocation struct {
	JSONPath string `json:"json_path,omitempty"`
	CSVPath  string `json:"csv_path,omitempty"`
}

func optimizationFromUpdate(experimentID string, update optimization.OptimizationUpdated, at time.Time) OptimizationResult {
	metrics, _ := json.Marshal(update.Metrics)
	return OptimizationResult{
		ExperimentID: experimentID,
		Score:        update.Score,
		WinRate:      update.Metrics.WinRate,
		Expectancy:   update.Metrics.Expectancy,
		ProfitFactor: update.Metrics.ProfitFactor,
		Drawdown:     update.Metrics.MaxDrawdown,
		Metrics:      metrics,
		CreatedAt:    at,
	}
}

func walkForwardFromEvent(completed walkForwardCompletedInput) WalkForwardResult {
	params, _ := json.Marshal(completed.BestParameters)
	metrics, _ := json.Marshal(completed.PerformanceMetrics)
	return WalkForwardResult{
		WalkForwardID:      completed.WalkForwardID,
		ExperimentID:       completed.ExperimentID,
		RunID:              completed.RunID,
		TrainScore:         completed.TrainingScore,
		ValidationScore:    completed.ValidationScore,
		ParameterSet:       params,
		PerformanceMetrics: metrics,
		CreatedAt:          completed.Timestamp,
	}
}

func monteCarloFromEvent(completed monteCarloCompletedInput) MonteCarloResult {
	ci, _ := json.Marshal(completed.ConfidenceInterval)
	dist, _ := json.Marshal(completed.DistributionSummary)
	return MonteCarloResult{
		SimulationID:        completed.SimulationID,
		WalkForwardID:       completed.WalkForwardID,
		ExperimentID:        completed.ExperimentID,
		Simulations:         completed.Simulations,
		ConfidenceInterval:  ci,
		ProbabilityOfProfit: completed.ProbabilityOfProfit,
		ProbabilityOfLoss:   completed.ProbabilityOfLoss,
		RiskOfRuin:          completed.RiskOfRuin,
		Distribution:        dist,
		CreatedAt:           completed.Timestamp,
	}
}

func buildSummary(bundle ResearchBundle) ReportSummary {
	summary := ReportSummary{}
	for _, opt := range bundle.Optimization {
		if opt.Score > summary.BestScore {
			summary.BestScore = opt.Score
		}
	}
	if len(bundle.WalkForward) > 0 {
		latest := bundle.WalkForward[len(bundle.WalkForward)-1]
		summary.LatestValidation = latest.ValidationScore
	}
	if len(bundle.MonteCarlo) > 0 {
		latest := bundle.MonteCarlo[len(bundle.MonteCarlo)-1]
		summary.ProbabilityOfProfit = latest.ProbabilityOfProfit
		summary.RiskOfRuin = latest.RiskOfRuin
	}
	return summary
}

func buildUnifiedReport(researchID string, version int, bundle ResearchBundle, at time.Time) UnifiedReport {
	return UnifiedReport{
		ResearchID:   researchID,
		ExperimentID: bundle.Experiment.ExperimentID,
		Version:      version,
		Strategy:     bundle.Experiment.Strategy,
		Symbol:       bundle.Experiment.Symbol,
		Timeframe:    bundle.Experiment.Timeframe,
		Parameters:   bundle.Experiment.Parameters,
		Optimization: bundle.Optimization,
		WalkForward:  bundle.WalkForward,
		MonteCarlo:   bundle.MonteCarlo,
		Summary:      buildSummary(bundle),
		GeneratedAt:  at,
	}
}

// walkForwardCompletedInput and monteCarloCompletedInput avoid import cycles in models.
type walkForwardCompletedInput struct {
	WalkForwardID      string
	ExperimentID       string
	RunID              string
	BestParameters     map[string]any
	TrainingScore      float64
	ValidationScore    float64
	PerformanceMetrics optimization.EvaluationMetrics
	Timestamp          time.Time
}

type monteCarloCompletedInput struct {
	SimulationID        string
	WalkForwardID       string
	ExperimentID        string
	Simulations         int
	ConfidenceInterval  montecarlo.ConfidenceInterval
	ProbabilityOfProfit float64
	ProbabilityOfLoss   float64
	RiskOfRuin          float64
	DistributionSummary montecarlo.DistributionSummary
	Timestamp           time.Time
}
