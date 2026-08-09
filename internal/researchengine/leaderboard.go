package researchengine

import (
	"math"
	"sort"
	"time"
)

const (
	scoreCapProfitFactor = 5.0
	scoreCapSharpe       = 3.0
	scoreCapExpectancy   = 100.0
	scoreCapTrades       = 2000.0
	scoreCapDrawdownPct  = 50.0
)

// ComputeOverallScore derives a weighted championship score from statistics.
func ComputeOverallScore(stats Statistics, weights RankingWeights) (float64, ScoreComponents) {
	weights = weights.withDefaults()
	if stats.TotalTrades == 0 {
		return 0, ScoreComponents{}
	}

	components := ScoreComponents{
		ProfitFactor: normalizeMetric(stats.ProfitFactor, scoreCapProfitFactor),
		Sharpe:       normalizeMetric(stats.SharpeRatio, scoreCapSharpe),
		Drawdown:     normalizeMetric(stats.MaxDrawdownPercent, scoreCapDrawdownPct),
		WinRate:      clamp01(stats.WinRate),
		Expectancy:   normalizeMetric(math.Abs(stats.Expectancy), scoreCapExpectancy),
		TradeCount:   normalizeMetric(float64(stats.TotalTrades), scoreCapTrades),
	}

	sum := weights.weightSum()
	if sum <= 0 {
		return 0, components
	}

	score := (weights.ProfitFactor*components.ProfitFactor +
		weights.Sharpe*components.Sharpe +
		weights.WinRate*components.WinRate +
		weights.Expectancy*components.Expectancy +
		weights.TradeCount*components.TradeCount -
		weights.Drawdown*components.Drawdown) / sum

	return clamp01(score), components
}

func normalizeMetric(value, cap float64) float64 {
	if value <= 0 || cap <= 0 {
		return 0
	}
	if value >= cap {
		return 1
	}
	return value / cap
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// LeaderboardEntry is one ranked row in a strategy championship.
type LeaderboardEntry struct {
	Rank            int              `json:"rank"`
	Strategy        string           `json:"strategy"`
	Category        string           `json:"category"`
	OverallScore    float64          `json:"overall_score"`
	TotalTrades     int              `json:"total_trades"`
	WinRate         float64          `json:"win_rate"`
	ProfitFactor    float64          `json:"profit_factor"`
	SharpeRatio     float64          `json:"sharpe_ratio"`
	MaxDrawdownPct  float64          `json:"max_drawdown_percent"`
	NetProfit       float64          `json:"net_profit"`
	Expectancy      float64          `json:"expectancy"`
	ScoreComponents ScoreComponents  `json:"score_components"`
	Qualification   Qualification    `json:"qualification"`
}

// StrategyLeaderboard ranks strategies on identical historical data.
type StrategyLeaderboard struct {
	Symbol       string             `json:"symbol"`
	Timeframe    string             `json:"timeframe"`
	DataSource   string             `json:"data_source"`
	CandleCount  int                `json:"candle_count"`
	StrategyCount int               `json:"strategy_count"`
	Weights      RankingWeights     `json:"weights"`
	Entries      []LeaderboardEntry `json:"entries"`
	Reports      []PerformanceReport `json:"reports"`
	GeneratedAt  time.Time          `json:"generated_at"`
}

// BuildLeaderboard ranks performance reports by overall score descending.
func BuildLeaderboard(reports []PerformanceReport, weights RankingWeights, meta ChampionshipMeta) StrategyLeaderboard {
	weights = weights.withDefaults()
	scored := append([]PerformanceReport(nil), reports...)
	for i := range scored {
		score, comps := ComputeOverallScore(scored[i].Statistics, weights)
		scored[i].OverallScore = score
		scored[i].ScoreComponents = comps
	}

	sortReportsByScore(scored)

	entries := make([]LeaderboardEntry, len(scored))
	for i, r := range scored {
		rank := i + 1
		scored[i].Rank = rank
		entries[i] = LeaderboardEntry{
			Rank:            rank,
			Strategy:        r.Strategy,
			Category:        string(r.Category),
			OverallScore:    r.OverallScore,
			TotalTrades:     r.Statistics.TotalTrades,
			WinRate:         r.Statistics.WinRate,
			ProfitFactor:    r.Statistics.ProfitFactor,
			SharpeRatio:     r.Statistics.SharpeRatio,
			MaxDrawdownPct:  r.Statistics.MaxDrawdownPercent,
			NetProfit:       r.Statistics.NetProfit,
			Expectancy:      r.Statistics.Expectancy,
			ScoreComponents: r.ScoreComponents,
			Qualification:   r.Qualification,
		}
	}

	return StrategyLeaderboard{
		Symbol:        meta.Symbol,
		Timeframe:     meta.Timeframe,
		DataSource:    meta.DataSource,
		CandleCount:   meta.CandleCount,
		StrategyCount: len(scored),
		Weights:       weights,
		Entries:       entries,
		Reports:       scored,
		GeneratedAt:   time.Now().UTC(),
	}
}

func sortReportsByScore(reports []PerformanceReport) {
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].OverallScore != reports[j].OverallScore {
			return reports[i].OverallScore > reports[j].OverallScore
		}
		return reports[i].Strategy < reports[j].Strategy
	})
}

// ChampionshipMeta describes the dataset used for a championship run.
type ChampionshipMeta struct {
	Symbol      string
	Timeframe   string
	DataSource  string
	CandleCount int
}
