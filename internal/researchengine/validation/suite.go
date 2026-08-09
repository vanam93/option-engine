package validation

import (
	"fmt"
	"math"

	"github.com/vanam-gangireddy/option-engine/internal/researchengine"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/donchian"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/macd_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/opening_range"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func checkSimulatorLongPnL() CheckResult {
	name := "simulator long PnL and costs"
	candles := LongWinSeries()
	strat := newScripted("long_win", 1, map[int]strategylib.Signal{
		1: buySignal(),
		15: exitSignal(),
	})
	sim := researchengine.NewSimulator(researchengine.SimulatorConfig{
		InitialCapital: 100000,
		Quantity:       1,
		Commission:     1,
		SlippagePct:    0,
		TaxRate:        0,
	})
	journal := sim.Run(strat, candles)
	if journal.Len() != 1 {
		return fail(name, fmt.Sprintf("expected 1 trade, got %d", journal.Len()), nil)
	}
	t := journal.Trades[0]
	grossWant := 10.0
	if math.Abs(t.GrossProfit-grossWant) > 0.01 {
		return fail(name, fmt.Sprintf("gross profit %.4f, want %.2f", t.GrossProfit, grossWant), nil)
	}
	netWant := 8.0 // 10 - 2 commission (1 per side)
	if math.Abs(t.NetProfit-netWant) > 0.01 {
		return fail(name, fmt.Sprintf("net profit %.4f, want %.2f", t.NetProfit, netWant), nil)
	}
	return pass(name, fmt.Sprintf("gross=%.2f net=%.2f commission=%.2f", t.GrossProfit, t.NetProfit, t.Commission))
}

func checkSimulatorPositionRules() CheckResult {
	name := "simulator position management"
	candles := testutil.ClosesToCandles([]float64{100, 100, 100, 100, 100, 100, 100, 100})
	strat := newScripted("position_rules", 1, map[int]strategylib.Signal{
		1:  buySignal(),
		2:  buySignal(), // duplicate long
		3:  exitSignal(),
		4:  exitSignal(), // flat exit
		5:  sellSignal(),
		6:  sellSignal(), // duplicate short
		7:  exitSignal(),
	})
	sim := researchengine.NewSimulator(researchengine.SimulatorConfig{InitialCapital: 100000, Quantity: 1, Commission: 0, SlippagePct: 0})
	var metrics researchengine.SimulationMetrics
	sim.RunWithMetrics(strat, candles, nil, &metrics)

	issues := researchengine.ValidateSimulationMetrics(metrics)
	if metrics.IgnoredBuyLong != 1 {
		issues = append(issues, fmt.Sprintf("ignored buy while long: got %d want 1", metrics.IgnoredBuyLong))
	}
	if metrics.IgnoredSellShort != 1 {
		issues = append(issues, fmt.Sprintf("ignored sell while short: got %d want 1", metrics.IgnoredSellShort))
	}
	if metrics.IgnoredExitFlat != 1 {
		issues = append(issues, fmt.Sprintf("ignored exit while flat: got %d want 1", metrics.IgnoredExitFlat))
	}
	if metrics.OpensLong != 1 || metrics.OpensShort != 1 {
		issues = append(issues, fmt.Sprintf("opens long=%d short=%d want 1 each", metrics.OpensLong, metrics.OpensShort))
	}
	if len(issues) > 0 {
		return fail(name, "", issues)
	}
	return pass(name, fmt.Sprintf("signals buy=%d sell=%d exit=%d ignored long=%d short=%d flat=%d",
		metrics.BuySignals, metrics.SellSignals, metrics.ExitSignals,
		metrics.IgnoredBuyLong, metrics.IgnoredSellShort, metrics.IgnoredExitFlat))
}

func checkStatisticsDrawdown() CheckResult {
	name := "statistics drawdown cap"
	// Simulate losing streak to drive equity negative
	candles := testutil.ClosesToCandles([]float64{100, 90, 80, 70, 60, 50, 40, 30, 20, 10})
	strat := newScripted("loser", 1, map[int]strategylib.Signal{
		1: buySignal(),
		2: exitSignal(),
		3: buySignal(),
		4: exitSignal(),
		5: buySignal(),
		6: exitSignal(),
	})
	sim := researchengine.NewSimulator(researchengine.SimulatorConfig{InitialCapital: 100, Quantity: 1, Commission: 0, SlippagePct: 0})
	journal := sim.Run(strat, candles)
	stats := researchengine.ComputeStatistics(journal, 100)
	issues := researchengine.ValidateStatistics(stats, 100)
	if stats.MaxDrawdownPercent > 100 {
		issues = append(issues, fmt.Sprintf("drawdown %.2f%% > 100%%", stats.MaxDrawdownPercent))
	}
	if len(issues) > 0 {
		return fail(name, fmt.Sprintf("max DD %.2f%%", stats.MaxDrawdownPercent), issues)
	}
	return pass(name, fmt.Sprintf("max drawdown %.2f%% (capped)", stats.MaxDrawdownPercent))
}

func checkStatisticsKnownTrades() CheckResult {
	name := "statistics win rate and profit factor"
	journal := researchengine.NewJournal()
	journal.Add(researchengine.SimulatedTrade{NetProfit: 100})
	journal.Add(researchengine.SimulatedTrade{NetProfit: -50})
	journal.Add(researchengine.SimulatedTrade{NetProfit: 50})
	journal.Add(researchengine.SimulatedTrade{NetProfit: -25})

	stats := researchengine.ComputeStatistics(journal, 10000)
	if stats.TotalTrades != 4 {
		return fail(name, fmt.Sprintf("trades %d want 4", stats.TotalTrades), nil)
	}
	if stats.WinningTrades != 2 || stats.LosingTrades != 2 {
		return fail(name, fmt.Sprintf("wins=%d losses=%d", stats.WinningTrades, stats.LosingTrades), nil)
	}
	if math.Abs(stats.WinRate-0.5) > 0.001 {
		return fail(name, fmt.Sprintf("win rate %.4f want 0.5", stats.WinRate), nil)
	}
	wantPF := 150.0 / 75.0
	if math.Abs(stats.ProfitFactor-wantPF) > 0.01 {
		return fail(name, fmt.Sprintf("PF %.4f want %.2f", stats.ProfitFactor, wantPF), nil)
	}
	if math.Abs(stats.NetProfit-75) > 0.01 {
		return fail(name, fmt.Sprintf("net %.2f want 75", stats.NetProfit), nil)
	}
	return pass(name, fmt.Sprintf("win rate=%.0f%% PF=%.2f net=%.2f", stats.WinRate*100, stats.ProfitFactor, stats.NetProfit))
}

func checkStrategyEMAcross() CheckResult {
	name := "strategy ema_cross bullish cross"
	s := ema_cross.New(map[string]any{"fast": 3, "slow": 5})
	signals := testutil.EvaluateSeries(s, testutil.ClosesToCandles(EMABullishCrossCloses()), strategylib.PositionFlat)
	if !testutil.HasDecision(signals, strategylib.DecisionBuy) {
		return fail(name, "expected BUY on bullish cross fixture", nil)
	}
	return pass(name, "BUY signal observed on known bullish cross fixture")
}

func checkStrategyMACD() CheckResult {
	name := "strategy macd_cross produces signals"
	s := macd_cross.NewDefault()
	closes := make([]float64, 40)
	for i := range closes {
		closes[i] = 100 + float64(i%5)
	}
	signals := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	hasAction := testutil.HasDecision(signals, strategylib.DecisionBuy) ||
		testutil.HasDecision(signals, strategylib.DecisionSell)
	if !hasAction {
		return fail(name, "no BUY/SELL on oscillating fixture", nil)
	}
	return pass(name, "action signal observed on oscillating fixture")
}

func checkStrategyDonchian() CheckResult {
	name := "strategy donchian breakout"
	s := donchian.New(map[string]any{"period": 3})
	closes := []float64{10, 11, 12, 16, 17, 18, 8, 9, 10, 11, 12, 20}
	signals := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	if !testutil.HasDecision(signals, strategylib.DecisionBuy) {
		return fail(name, "expected BUY on channel breakout fixture", nil)
	}
	return pass(name, "BUY on Donchian breakout fixture")
}

func checkStrategyOpeningRangeSessions() CheckResult {
	name := "strategy opening_range multi-session entries"
	s := opening_range.New(map[string]any{"window_minutes": 15})
	candles := MultiSessionBreakoutDays(3)
	buys := 0
	for i, c := range candles {
		ctx := strategylib.Context{
			Candle:    c,
			Position:  strategylib.PositionFlat,
			BarIndex:  i,
			Timestamp: c.CloseTime,
		}
		sig := s.Evaluate(ctx)
		if sig.Decision == strategylib.DecisionBuy {
			buys++
		}
	}
	if buys < 2 {
		return fail(name, fmt.Sprintf("expected BUY on multiple sessions, got %d buy signals", buys), nil)
	}
	return pass(name, fmt.Sprintf("%d BUY signals across %d session days", buys, 3))
}

func checkJournalInvariants() CheckResult {
	name := "journal structural invariants"
	candles := LongWinSeries()
	strat := newScripted("journal_check", 1, map[int]strategylib.Signal{1: buySignal(), 10: exitSignal()})
	sim := researchengine.NewSimulator(researchengine.SimulatorConfig{InitialCapital: 100000, Quantity: 1, Commission: 1, SlippagePct: 0})
	journal := sim.Run(strat, candles)
	issues := researchengine.ValidateJournal(journal)
	stats := researchengine.ComputeStatistics(journal, 100000)
	issues = append(issues, researchengine.ValidateStatistics(stats, 100000)...)
	if len(issues) > 0 {
		return fail(name, "", issues)
	}
	return pass(name, "all journal and statistics invariants satisfied")
}

func pass(name, details string) CheckResult {
	return CheckResult{Name: name, Passed: true, Details: details}
}

func fail(name, details string, issues []string) CheckResult {
	if details != "" && len(issues) == 0 {
		issues = []string{details}
		details = ""
	}
	return CheckResult{Name: name, Passed: false, Details: details, Issues: issues}
}
