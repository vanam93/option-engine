package researchengine

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// ScoreComponents holds normalized ranking factor contributions.
type ScoreComponents struct {
	ProfitFactor float64 `json:"profit_factor"`
	Sharpe       float64 `json:"sharpe"`
	Drawdown     float64 `json:"drawdown"`
	WinRate      float64 `json:"win_rate"`
	Expectancy   float64 `json:"expectancy"`
	TradeCount   float64 `json:"trade_count"`
}

// PerformanceReport is the full research output for one strategy on a dataset.
type PerformanceReport struct {
	Strategy        string             `json:"strategy"`
	StrategyVersion string             `json:"strategy_version"`
	Category        strategylib.Category `json:"category"`
	Symbol          string             `json:"symbol"`
	Timeframe       string             `json:"timeframe"`
	Parameters      map[string]any     `json:"parameters"`
	WarmupBars      int                `json:"warmup_bars"`
	Statistics      Statistics         `json:"statistics"`
	OverallScore    float64            `json:"overall_score"`
	ScoreComponents ScoreComponents    `json:"score_components"`
	Rank            int                `json:"rank"`
	Qualification   Qualification      `json:"qualification"`
	GeneratedAt     time.Time          `json:"generated_at"`
}

// RankingWeights configures the championship overall score formula.
type RankingWeights struct {
	ProfitFactor float64 `json:"profit_factor"`
	Sharpe       float64 `json:"sharpe"`
	Drawdown     float64 `json:"drawdown"`
	WinRate      float64 `json:"win_rate"`
	Expectancy   float64 `json:"expectancy"`
	TradeCount   float64 `json:"trade_count"`
}

// DefaultRankingWeights returns industry-style championship weights.
func DefaultRankingWeights() RankingWeights {
	return RankingWeights{
		ProfitFactor: 0.30,
		Sharpe:       0.20,
		Drawdown:     0.20,
		WinRate:      0.15,
		Expectancy:   0.10,
		TradeCount:   0.05,
	}
}

func (w RankingWeights) withDefaults() RankingWeights {
	out := w
	def := DefaultRankingWeights()
	if out.ProfitFactor == 0 && out.Sharpe == 0 && out.Drawdown == 0 &&
		out.WinRate == 0 && out.Expectancy == 0 && out.TradeCount == 0 {
		return def
	}
	return out
}

func (w RankingWeights) weightSum() float64 {
	return w.ProfitFactor + w.Sharpe + w.Drawdown + w.WinRate + w.Expectancy + w.TradeCount
}
