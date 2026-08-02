package performance

import "time"

// InputUpdate mirrors the portfolio.updated payload consumed by the performance engine.
type InputUpdate struct {
	Symbol        string
	PositionOpen  bool
	RealizedPnL   float64
	UnrealizedPnL float64
	Timestamp     time.Time
}

// TradeResult records a completed round-trip trade.
type TradeResult struct {
	Symbol    string
	PnL       float64
	Timestamp time.Time
}

// EquityPoint is a point on the equity curve.
type EquityPoint struct {
	Equity    float64
	Timestamp time.Time
}

// PerformanceUpdated is the payload published on performance.updated events.
type PerformanceUpdated struct {
	TotalTrades   int       `json:"total_trades"`
	WinRate       float64   `json:"win_rate"`
	RealizedPnL   float64   `json:"realized_pnl"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	Drawdown      float64   `json:"drawdown"`
	Timestamp     time.Time `json:"timestamp"`
}

// PerformanceSnapshot is an immutable snapshot of all performance metrics.
type PerformanceSnapshot struct {
	TotalTrades     int
	WinningTrades   int
	LosingTrades    int
	WinRate         float64
	RealizedPnL     float64
	UnrealizedPnL   float64
	NetPnL          float64
	ProfitFactor    float64
	MaxDrawdown     float64
	CurrentDrawdown float64
	AverageTradePnL float64
	SharpeRatio     float64
	Trades          []TradeResult
	EquityCurve     []EquityPoint
}
