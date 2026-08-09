package researchengine

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportLeaderboardJSON writes the leaderboard to a JSON file.
func ExportLeaderboardJSON(board StrategyLeaderboard, path string) error {
	if path == "" {
		return fmt.Errorf("json export path required")
	}
	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ExportLeaderboardCSV writes leaderboard entries to a CSV file.
func ExportLeaderboardCSV(board StrategyLeaderboard, path string) error {
	if path == "" {
		return fmt.Errorf("csv export path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"rank", "strategy", "category", "status", "overall_score",
		"total_trades", "win_rate", "profit_factor", "sharpe",
		"max_drawdown_percent", "net_profit", "expectancy",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, e := range board.Entries {
		row := []string{
			fmt.Sprintf("%d", e.Rank),
			e.Strategy,
			e.Category,
			string(e.Qualification.Status),
			fmt.Sprintf("%.4f", e.OverallScore),
			fmt.Sprintf("%d", e.TotalTrades),
			fmt.Sprintf("%.4f", e.WinRate),
			fmt.Sprintf("%.4f", e.ProfitFactor),
			fmt.Sprintf("%.4f", e.SharpeRatio),
			fmt.Sprintf("%.4f", e.MaxDrawdownPct),
			fmt.Sprintf("%.2f", e.NetProfit),
			fmt.Sprintf("%.4f", e.Expectancy),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}

// FormatLeaderboard renders a human-readable championship table.
func FormatLeaderboard(board StrategyLeaderboard) string {
	var b strings.Builder
	b.WriteString("Strategy Championship Leaderboard\n")
	b.WriteString("===================================\n\n")
	b.WriteString(fmt.Sprintf("Symbol: %s\n", board.Symbol))
	b.WriteString(fmt.Sprintf("Timeframe: %s\n", board.Timeframe))
	b.WriteString(fmt.Sprintf("Data: %s\n", board.DataSource))
	b.WriteString(fmt.Sprintf("Candles: %d\n", board.CandleCount))
	b.WriteString(fmt.Sprintf("Strategies: %d\n\n", board.StrategyCount))

	b.WriteString(fmt.Sprintf("%-4s %-22s %-14s %-10s %-8s %-6s %-7s %-7s %-7s %-10s %-12s\n",
		"Rank", "Strategy", "Category", "Status", "Score", "Trades", "Win%", "PF", "Sharpe", "MaxDD%", "Net Profit"))
	b.WriteString(strings.Repeat("-", 110) + "\n")

	for _, e := range board.Entries {
		b.WriteString(fmt.Sprintf("%-4d %-22s %-14s %-10s %-8.3f %-6d %-7.1f %-7.2f %-7.2f %-10.1f %-12.2f\n",
			e.Rank,
			e.Strategy,
			e.Category,
			e.Qualification.Status,
			e.OverallScore,
			e.TotalTrades,
			e.WinRate*100,
			e.ProfitFactor,
			e.SharpeRatio,
			e.MaxDrawdownPct,
			e.NetProfit,
		))
	}

	if len(board.Entries) > 0 {
		winner := board.Entries[0]
		b.WriteString("\nBest strategy on this dataset: ")
		b.WriteString(winner.Strategy)
		b.WriteString(fmt.Sprintf(" (score %.3f, net profit %.2f)\n", winner.OverallScore, winner.NetProfit))
	}
	return b.String()
}
