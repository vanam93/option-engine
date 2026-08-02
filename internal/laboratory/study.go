package laboratory

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/backtestrunner"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// StudyStatus describes research study lifecycle state.
type StudyStatus string

const (
	StudyStatusPending   StudyStatus = "PENDING"
	StudyStatusRunning   StudyStatus = "RUNNING"
	StudyStatusCompleted StudyStatus = "COMPLETED"
	StudyStatusFailed    StudyStatus = "FAILED"
)

// StudyRequest configures a new research study.
type StudyRequest struct {
	Name        string
	Description string
	Strategy    string
	Parameters  map[string]string
	Symbols     []string
	Timeframes  []market.Timeframe
	StartTime   time.Time
	EndTime     time.Time
	Expiries    []string
	Speed       float64
	DataPath    string
}

// Study is an immutable research study record.
type Study struct {
	StudyID            string
	Name               string
	Description        string
	Strategy           string
	Parameters         map[string]string
	Symbols            []string
	Timeframes         []market.Timeframe
	StartTime          time.Time
	EndTime            time.Time
	CreatedAt          time.Time
	CompletedAt        *time.Time
	Status             StudyStatus
	ResearchVersion    string
	BacktestSessionIDs []string
	Output             *StudyOutput
	Error              string
}

// StudyOutput aggregates downstream summaries from completed backtest sessions.
type StudyOutput struct {
	BacktestSummaries       []backtestrunner.SessionSummary `json:"backtest_summaries"`
	OptimizationSummaries   []OptimizationSummary           `json:"optimization_summaries"`
	WalkForwardSummaries    []WalkForwardSummary            `json:"walk_forward_summaries"`
	MonteCarloSummaries     []MonteCarloSummary             `json:"monte_carlo_summaries"`
	RecommendationSummaries []RecommendationSummary         `json:"recommendation_summaries"`
	QualitySummaries        []QualitySummary                `json:"quality_summaries"`
	FeedbackSummaries       []backtestrunner.FeedbackSummary `json:"feedback_summaries"`
	ResearchReports         int                             `json:"research_reports"`
}

// OptimizationSummary captures optimization activity from a backtest session.
type OptimizationSummary struct {
	BacktestID       string `json:"backtest_id"`
	OptimizationRuns int    `json:"optimization_runs"`
}

// WalkForwardSummary captures walk-forward activity from a backtest session.
type WalkForwardSummary struct {
	BacktestID      string `json:"backtest_id"`
	WalkForwardRuns int    `json:"walk_forward_runs"`
}

// MonteCarloSummary captures Monte Carlo activity from a backtest session.
type MonteCarloSummary struct {
	BacktestID     string `json:"backtest_id"`
	MonteCarloRuns int    `json:"monte_carlo_runs"`
}

// RecommendationSummary captures recommendation aggregates from a backtest session.
type RecommendationSummary struct {
	BacktestID               string  `json:"backtest_id"`
	RecommendationsGenerated int     `json:"recommendations_generated"`
	RecommendationsClosed    int     `json:"recommendations_closed"`
	BuyCount                 int     `json:"buy_count"`
	WatchCount               int     `json:"watch_count"`
	AvoidCount               int     `json:"avoid_count"`
	AverageConfidence        float64 `json:"average_confidence"`
	AverageReturn            float64 `json:"average_return"`
	WinRate                  float64 `json:"win_rate"`
}

// QualitySummary captures quality distribution from a backtest session.
type QualitySummary struct {
	BacktestID          string         `json:"backtest_id"`
	QualityDistribution map[string]int `json:"quality_distribution"`
}

// StudyStarted is published on study.started.
type StudyStarted struct {
	StudyID         string    `json:"study_id"`
	Name            string    `json:"name"`
	Strategy        string    `json:"strategy"`
	Symbols         []string  `json:"symbols"`
	ResearchVersion string    `json:"research_version"`
	StartedAt       time.Time `json:"started_at"`
}

// StudyCompleted is published on study.completed.
type StudyCompleted struct {
	StudyID            string      `json:"study_id"`
	Status             StudyStatus `json:"status"`
	ResearchVersion    string      `json:"research_version"`
	BacktestSessionIDs []string    `json:"backtest_session_ids"`
	CompletedAt        time.Time   `json:"completed_at"`
	Error              string      `json:"error,omitempty"`
}

// StudyCompared is published on study.compared.
type StudyCompared struct {
	ComparisonID string             `json:"comparison_id"`
	Criteria     ComparisonCriteria `json:"criteria"`
	StudyIDs     []string           `json:"study_ids"`
	ComparedAt   time.Time          `json:"compared_at"`
}

// Validate checks study request fields.
func (r StudyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("%w: name required", ErrInvalidStudy)
	}
	if r.Strategy == "" {
		return fmt.Errorf("%w: strategy required", ErrInvalidStudy)
	}
	if r.StartTime.IsZero() {
		return fmt.Errorf("%w: start_time required", ErrInvalidStudy)
	}
	if r.EndTime.IsZero() {
		return fmt.Errorf("%w: end_time required", ErrInvalidStudy)
	}
	if r.EndTime.Before(r.StartTime) {
		return fmt.Errorf("%w: end_time must be after start_time", ErrInvalidStudy)
	}
	if len(r.Symbols) == 0 {
		return fmt.Errorf("%w: symbols required", ErrInvalidStudy)
	}
	return nil
}

func (r StudyRequest) withDefaults() StudyRequest {
	out := r
	if out.Speed <= 0 {
		out.Speed = 1.0
	}
	if len(out.Timeframes) == 0 {
		out.Timeframes = []market.Timeframe{market.TF1m}
	}
	out.Symbols = append([]string(nil), out.Symbols...)
	out.Timeframes = append([]market.Timeframe(nil), out.Timeframes...)
	out.Expiries = append([]string(nil), out.Expiries...)
	if out.Parameters == nil {
		out.Parameters = make(map[string]string)
	} else {
		params := make(map[string]string, len(out.Parameters))
		for k, v := range out.Parameters {
			params[k] = v
		}
		out.Parameters = params
	}
	return out
}

func newStudy(req StudyRequest, version string, at time.Time) Study {
	req = req.withDefaults()
	return Study{
		StudyID:         generateStudyID(at),
		Name:            req.Name,
		Description:     req.Description,
		Strategy:        req.Strategy,
		Parameters:      copyParameters(req.Parameters),
		Symbols:         append([]string(nil), req.Symbols...),
		Timeframes:      append([]market.Timeframe(nil), req.Timeframes...),
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		Status:          StudyStatusPending,
		ResearchVersion: version,
		CreatedAt:       at,
	}
}

func generateStudyID(at time.Time) string {
	return fmt.Sprintf("STUDY-%s-%s", at.UTC().Format("20060102T150405"), uuid.NewString()[:8])
}

func copyParameters(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func buildStudyOutput(sessions []backtestrunner.Session) StudyOutput {
	out := StudyOutput{}
	for _, session := range sessions {
		if session.Summary == nil {
			continue
		}
		summary := *session.Summary
		out.BacktestSummaries = append(out.BacktestSummaries, summary)
		out.OptimizationSummaries = append(out.OptimizationSummaries, OptimizationSummary{
			BacktestID:       session.BacktestID,
			OptimizationRuns: summary.OptimizationRuns,
		})
		out.WalkForwardSummaries = append(out.WalkForwardSummaries, WalkForwardSummary{
			BacktestID:      session.BacktestID,
			WalkForwardRuns: summary.WalkForwardRuns,
		})
		out.MonteCarloSummaries = append(out.MonteCarloSummaries, MonteCarloSummary{
			BacktestID:     session.BacktestID,
			MonteCarloRuns: summary.MonteCarloRuns,
		})
		out.RecommendationSummaries = append(out.RecommendationSummaries, RecommendationSummary{
			BacktestID:               session.BacktestID,
			RecommendationsGenerated: summary.RecommendationsGenerated,
			RecommendationsClosed:    summary.RecommendationsClosed,
			BuyCount:                 summary.BuyCount,
			WatchCount:               summary.WatchCount,
			AvoidCount:               summary.AvoidCount,
			AverageConfidence:        summary.AverageConfidence,
			AverageReturn:            summary.AverageReturn,
			WinRate:                  summary.WinRate,
		})
		out.QualitySummaries = append(out.QualitySummaries, QualitySummary{
			BacktestID:          session.BacktestID,
			QualityDistribution: copyIntMap(summary.QualityDistribution),
		})
		if summary.FeedbackSummary.TotalRecommendations > 0 {
			out.FeedbackSummaries = append(out.FeedbackSummaries, summary.FeedbackSummary)
		}
		out.ResearchReports += summary.ResearchReportsGenerated
	}
	return out
}

func copyIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return make(map[string]int)
	}
	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
