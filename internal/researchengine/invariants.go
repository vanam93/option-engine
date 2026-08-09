package researchengine

import (
	"fmt"
	"strings"
)

// ValidateJournal checks trade records for structural integrity.
func ValidateJournal(journal *Journal) []string {
	if journal == nil {
		return nil
	}
	var issues []string
	for i, t := range journal.Trades {
		prefix := fmt.Sprintf("trade[%d]", i)
		if t.ExitTime.IsZero() {
			issues = append(issues, prefix+": missing exit time")
		}
		if !t.EntryTime.Before(t.ExitTime) && !t.EntryTime.Equal(t.ExitTime) {
			issues = append(issues, prefix+": entry must precede or equal exit time")
		}
		if t.EntryPrice <= 0 || t.ExitPrice <= 0 {
			issues = append(issues, prefix+": invalid entry or exit price")
		}
		if t.Quantity <= 0 {
			issues = append(issues, prefix+": invalid quantity")
		}
		if t.BarsHeld < 1 {
			issues = append(issues, prefix+": bars held must be >= 1")
		}
	}
	return issues
}

// ValidateStatistics checks computed metrics for impossible values.
func ValidateStatistics(stats Statistics, initialCapital float64) []string {
	var issues []string
	if stats.MaxDrawdownPercent > 100 {
		issues = append(issues, fmt.Sprintf("max drawdown %.2f%% exceeds 100%%", stats.MaxDrawdownPercent))
	}
	if stats.ProfitFactor < 0 {
		issues = append(issues, fmt.Sprintf("profit factor %.4f is negative", stats.ProfitFactor))
	}
	if stats.WinRate < 0 || stats.WinRate > 1 {
		issues = append(issues, fmt.Sprintf("win rate %.4f out of range [0,1]", stats.WinRate))
	}
	if stats.TotalTrades > 0 {
		if stats.WinningTrades+stats.LosingTrades > stats.TotalTrades {
			issues = append(issues, "winning+losing trades exceed total trades")
		}
	}
	if initialCapital > 0 && stats.MaxDrawdown > initialCapital*2 {
		issues = append(issues, fmt.Sprintf("max drawdown %.2f unrealistic vs capital %.2f", stats.MaxDrawdown, initialCapital))
	}
	return issues
}

// ValidateSimulationMetrics checks position management accounting.
func ValidateSimulationMetrics(metrics SimulationMetrics) []string {
	var issues []string
	opens := metrics.OpensLong + metrics.OpensShort
	closes := metrics.Closes
	if opens != closes {
		issues = append(issues, fmt.Sprintf("opens (%d) != closes (%d)", opens, closes))
	}
	expectedSignals := metrics.BuySignals + metrics.SellSignals + metrics.ExitSignals
	if expectedSignals == 0 && opens > 0 {
		issues = append(issues, "positions opened without action signals")
	}
	return issues
}

// FormatIssues renders validation issues for reports.
func FormatIssues(issues []string) string {
	if len(issues) == 0 {
		return "none"
	}
	var b strings.Builder
	for _, issue := range issues {
		b.WriteString("  - ")
		b.WriteString(issue)
		b.WriteString("\n")
	}
	return b.String()
}
