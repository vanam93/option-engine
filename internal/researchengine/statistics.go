package researchengine

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Statistics summarizes institutional performance metrics from a trade journal.
type Statistics struct {
	TotalTrades         int                `json:"total_trades"`
	WinningTrades       int                `json:"winning_trades"`
	LosingTrades        int                `json:"losing_trades"`
	WinRate             float64            `json:"win_rate"`
	GrossProfit         float64            `json:"gross_profit"`
	GrossLoss           float64            `json:"gross_loss"`
	NetProfit           float64            `json:"net_profit"`
	AverageWin          float64            `json:"average_win"`
	AverageLoss         float64            `json:"average_loss"`
	LargestWin          float64            `json:"largest_win"`
	LargestLoss         float64            `json:"largest_loss"`
	ProfitFactor        float64            `json:"profit_factor"`
	Expectancy          float64            `json:"expectancy"`
	SharpeRatio         float64            `json:"sharpe_ratio"`
	SortinoRatio        float64            `json:"sortino_ratio"`
	CalmarRatio         float64            `json:"calmar_ratio"`
	RecoveryFactor      float64            `json:"recovery_factor"`
	MaxDrawdown         float64            `json:"max_drawdown"`
	MaxDrawdownPercent  float64            `json:"max_drawdown_percent"`
	LongestWinStreak    int                `json:"longest_win_streak"`
	LongestLossStreak   int                `json:"longest_loss_streak"`
	AverageBarsHeld     float64            `json:"average_bars_held"`
	AverageHoldDuration time.Duration      `json:"average_hold_duration"`
	Exposure            float64            `json:"exposure"`
	MonthlyReturns      map[string]float64 `json:"monthly_returns"`
	AnnualReturns       map[string]float64 `json:"annual_returns"`
	EquityCurve         []float64          `json:"equity_curve"`
}

// ComputeStatistics derives metrics from journal and initial capital.
func ComputeStatistics(journal *Journal, initialCapital float64) Statistics {
	if journal == nil || journal.Len() == 0 {
		return Statistics{
			MonthlyReturns: make(map[string]float64),
			AnnualReturns:  make(map[string]float64),
		}
	}

	stats := Statistics{
		TotalTrades:    journal.Len(),
		MonthlyReturns: make(map[string]float64),
		AnnualReturns:  make(map[string]float64),
	}

	var wins, losses []float64
	equity := initialCapital
	peak := initialCapital
	maxDD := 0.0
	curve := []float64{initialCapital}
	winStreak, lossStreak := 0, 0
	var totalBars time.Duration
	var totalHold time.Duration

	for _, t := range journal.Trades {
		pnl := t.NetProfit
		equity += pnl
		curve = append(curve, equity)
		if equity > peak {
			peak = equity
		}
		if dd := peak - equity; dd > maxDD {
			maxDD = dd
		}

		stats.NetProfit += pnl
		totalBars += time.Duration(t.BarsHeld)
		totalHold += t.HoldingDuration

		monthKey := t.ExitTime.Format("2006-01")
		yearKey := t.ExitTime.Format("2006")
		stats.MonthlyReturns[monthKey] += pnl
		stats.AnnualReturns[yearKey] += pnl

		if pnl > 0 {
			stats.WinningTrades++
			stats.GrossProfit += pnl
			wins = append(wins, pnl)
			if pnl > stats.LargestWin {
				stats.LargestWin = pnl
			}
			winStreak++
			if winStreak > stats.LongestWinStreak {
				stats.LongestWinStreak = winStreak
			}
			lossStreak = 0
		} else if pnl < 0 {
			stats.LosingTrades++
			stats.GrossLoss += -pnl
			losses = append(losses, pnl)
			if pnl < stats.LargestLoss {
				stats.LargestLoss = pnl
			}
			lossStreak++
			if lossStreak > stats.LongestLossStreak {
				stats.LongestLossStreak = lossStreak
			}
			winStreak = 0
		}
	}

	stats.EquityCurve = curve
	stats.MaxDrawdown = maxDD
	if initialCapital > 0 {
		stats.MaxDrawdownPercent = (maxDD / initialCapital) * 100
		if stats.MaxDrawdownPercent > 100 {
			stats.MaxDrawdownPercent = 100
		}
	} else if peak > 0 {
		stats.MaxDrawdownPercent = (maxDD / peak) * 100
		if stats.MaxDrawdownPercent > 100 {
			stats.MaxDrawdownPercent = 100
		}
	}
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades)
		stats.Expectancy = stats.NetProfit / float64(stats.TotalTrades)
		stats.AverageBarsHeld = float64(totalBars) / float64(stats.TotalTrades)
		stats.AverageHoldDuration = totalHold / time.Duration(stats.TotalTrades)
	}
	if len(wins) > 0 {
		stats.AverageWin = mean(wins)
	}
	if len(losses) > 0 {
		stats.AverageLoss = mean(losses)
	}
	if stats.GrossLoss > 0 {
		stats.ProfitFactor = stats.GrossProfit / stats.GrossLoss
	}
	if maxDD > 0 {
		stats.RecoveryFactor = stats.NetProfit / maxDD
		stats.CalmarRatio = stats.NetProfit / maxDD
	}
	if initialCapital > 0 {
		stats.Exposure = stats.NetProfit / initialCapital
	}

	returns := journal.PnLSeries()
	stats.SharpeRatio = sharpe(returns)
	stats.SortinoRatio = sortino(returns)

	return stats
}

func sharpe(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	m := mean(returns)
	s := stdDev(returns, m)
	if s == 0 {
		return 0
	}
	return (m / s) * math.Sqrt(float64(len(returns)))
}

func sortino(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}
	m := mean(returns)
	var down []float64
	for _, r := range returns {
		if r < 0 {
			down = append(down, r)
		}
	}
	if len(down) == 0 {
		return 0
	}
	dm := mean(down)
	ds := stdDev(down, dm)
	if ds == 0 {
		return 0
	}
	return (m / ds) * math.Sqrt(float64(len(returns)))
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func stdDev(v []float64, mean float64) float64 {
	if len(v) < 2 {
		return 0
	}
	var sumSq float64
	for _, x := range v {
		d := x - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(v)))
}

// FormatReport renders a human-readable performance summary.
func FormatReport(strategyName, symbol, timeframe string, stats Statistics) string {
	var lines []string
	lines = append(lines, "Strategy Research Report", "========================", "")
	lines = append(lines, "Strategy: "+strategyName)
	lines = append(lines, "Symbol: "+symbol)
	lines = append(lines, "Timeframe: "+timeframe, "")
	lines = append(lines, fmt.Sprintf("Trades: %d", stats.TotalTrades))
	lines = append(lines, fmt.Sprintf("Win Rate: %.1f%%", stats.WinRate*100))
	lines = append(lines, fmt.Sprintf("Net Profit: %.2f", stats.NetProfit))
	lines = append(lines, fmt.Sprintf("Profit Factor: %.2f", stats.ProfitFactor))
	lines = append(lines, fmt.Sprintf("Expectancy: %.2f", stats.Expectancy))
	lines = append(lines, fmt.Sprintf("Sharpe: %.2f", stats.SharpeRatio))
	lines = append(lines, fmt.Sprintf("Sortino: %.2f", stats.SortinoRatio))
	lines = append(lines, fmt.Sprintf("Max Drawdown: %.2f (%.2f%%)", stats.MaxDrawdown, stats.MaxDrawdownPercent))
	lines = append(lines, fmt.Sprintf("Calmar: %.2f", stats.CalmarRatio))
	lines = append(lines, fmt.Sprintf("Recovery Factor: %.2f", stats.RecoveryFactor))
	return strings.Join(lines, "\n")
}
