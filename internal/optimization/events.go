package optimization

import "time"

// InputUpdate mirrors the performance.updated payload consumed by the optimization engine.
type InputUpdate struct {
	Strategy      string
	Symbol        string
	Timeframe     string
	Parameters    string
	TotalTrades   int
	WinRate       float64
	RealizedPnL   float64
	UnrealizedPnL float64
	Drawdown      float64
	ProfitFactor  float64
	MaxDrawdown   float64
	Timestamp     time.Time
}

// EvaluationKey uniquely identifies a strategy evaluation dimension.
type EvaluationKey struct {
	Strategy   string
	Symbol     string
	Timeframe  string
	Parameters string
}

// EvaluationMetrics holds computed performance metrics for ranking.
type EvaluationMetrics struct {
	TotalTrades  int     `json:"total_trades"`
	NetPnL       float64 `json:"net_pnl"`
	RealizedPnL  float64 `json:"realized_pnl"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
	AverageTrade float64 `json:"average_trade"`
	Expectancy   float64 `json:"expectancy"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	RiskReward   float64 `json:"risk_reward"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
}

// OptimizationUpdated is the payload published on optimization.updated events.
type OptimizationUpdated struct {
	Strategy   string            `json:"strategy"`
	Symbol     string            `json:"symbol"`
	Timeframe  string            `json:"timeframe"`
	Parameters string            `json:"parameters,omitempty"`
	Metrics    EvaluationMetrics `json:"metrics"`
	Score      float64           `json:"score"`
	Rank       int               `json:"rank"`
	Timestamp  time.Time         `json:"timestamp"`
}

// EvaluationRecord is an immutable evaluation entry with score and rank.
type EvaluationRecord struct {
	Key       EvaluationKey
	Metrics   EvaluationMetrics
	Score     float64
	Rank      int
	UpdatedAt time.Time
}

// StateSnapshot is an immutable read model of optimization state.
type StateSnapshot struct {
	Evaluations []EvaluationRecord
	Rankings    []EvaluationRecord
}
