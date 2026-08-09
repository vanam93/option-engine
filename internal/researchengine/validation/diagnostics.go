package validation

import (
	"fmt"
	"strings"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// StrategyDiagnostics summarizes one strategy simulation run.
type StrategyDiagnostics struct {
	Strategy           string
	Candles            int
	Trades             int
	BuySignals         int
	SellSignals        int
	ExitSignals        int
	OpensLong          int
	OpensShort         int
	Closes             int
	IgnoredBuyLong     int
	IgnoredSellShort   int
	IgnoredExitFlat    int
	AverageBarsHeld    float64
	AverageNetProfit   float64
	AverageWin         float64
	AverageLoss        float64
	MaxDrawdownPct     float64
	ProfitFactor       float64
	Qualification      string
	Issues             []string
}

// DiagnoseStrategy runs a strategy with metrics collection and invariant checks.
func DiagnoseStrategy(strategy strategylib.Strategy, candles []market.Candle, cfg researchengine.SimulatorConfig) StrategyDiagnostics {
	d := StrategyDiagnostics{Strategy: strategy.Name(), Candles: len(candles)}
	if strategy == nil || len(candles) == 0 {
		return d
	}

	dataset := researchengine.NewDataset(candles)
	sim := researchengine.NewSimulator(cfg)
	var metrics researchengine.SimulationMetrics
	journal := sim.RunWithMetrics(strategy, dataset.Candles, dataset.IndicatorSource(), &metrics)

	d.Trades = journal.Len()
	d.BuySignals = metrics.BuySignals
	d.SellSignals = metrics.SellSignals
	d.ExitSignals = metrics.ExitSignals
	d.OpensLong = metrics.OpensLong
	d.OpensShort = metrics.OpensShort
	d.Closes = metrics.Closes
	d.IgnoredBuyLong = metrics.IgnoredBuyLong
	d.IgnoredSellShort = metrics.IgnoredSellShort
	d.IgnoredExitFlat = metrics.IgnoredExitFlat

	stats := researchengine.ComputeStatistics(journal, cfg.InitialCapital)
	d.AverageBarsHeld = stats.AverageBarsHeld
	d.MaxDrawdownPct = stats.MaxDrawdownPercent
	d.ProfitFactor = stats.ProfitFactor
	if stats.TotalTrades > 0 {
		d.AverageNetProfit = stats.NetProfit / float64(stats.TotalTrades)
	}
	d.AverageWin = stats.AverageWin
	d.AverageLoss = stats.AverageLoss
	d.Qualification = string(researchengine.Qualify(stats).Status)

	d.Issues = append(d.Issues, researchengine.ValidateJournal(journal)...)
	d.Issues = append(d.Issues, researchengine.ValidateStatistics(stats, cfg.InitialCapital)...)
	d.Issues = append(d.Issues, researchengine.ValidateSimulationMetrics(metrics)...)

	// Heuristic sanity warnings for large datasets
	if len(candles) > 1000 {
		tradesPer1000 := float64(d.Trades) / float64(len(candles)) * 1000
		if tradesPer1000 > 200 {
			d.Issues = append(d.Issues, fmt.Sprintf("high trade frequency: %.1f trades per 1000 bars", tradesPer1000))
		}
	}
	return d
}

// FormatDiagnostics renders strategy diagnostic lines.
func FormatDiagnostics(diags []StrategyDiagnostics) string {
	var b strings.Builder
	b.WriteString("Strategy Diagnostics\n")
	b.WriteString("--------------------\n")
	for _, d := range diags {
		b.WriteString(fmt.Sprintf("%s: trades=%d signals(b=%d s=%d e=%d) ignored(l=%d s=%d f=%d) avgBars=%.1f PF=%.2f DD=%.1f%% status=%s\n",
			d.Strategy, d.Trades, d.BuySignals, d.SellSignals, d.ExitSignals,
			d.IgnoredBuyLong, d.IgnoredSellShort, d.IgnoredExitFlat,
			d.AverageBarsHeld, d.ProfitFactor, d.MaxDrawdownPct, d.Qualification))
		for _, issue := range d.Issues {
			b.WriteString("  ! " + issue + "\n")
		}
	}
	return b.String()
}
