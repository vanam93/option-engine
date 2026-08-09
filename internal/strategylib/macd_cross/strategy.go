package macd_cross

import (
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "macd_cross"

var (
	defaultParams = map[string]any{"fast": 12, "slow": 26, "signal": 9}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "fast", Values: []any{12}},
		{Name: "slow", Values: []any{26}},
		{Name: "signal", Values: []any{9}},
	}
)

// Strategy implements MACD line crossing signal line.
type Strategy struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
	macd         *indicators.MACD
	prevMACD     float64
	prevSignal   float64
	prevWarmed   bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	fast := strategylib.IntParam(merged, "fast", 12)
	slow := strategylib.IntParam(merged, "slow", 26)
	signal := strategylib.IntParam(merged, "signal", 9)
	return &Strategy{
		fastPeriod:   fast,
		slowPeriod:   slow,
		signalPeriod: signal,
		macd:         indicators.NewMACD(fast, slow, signal),
	}
}

func (s *Strategy) Name() string { return Name }
func (s *Strategy) Description() string {
	return "Enters when MACD line crosses the signal line; exits on opposite cross."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "MACD signal line crossover", strategylib.CategoryTrend)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeTrending, strategylib.RegimeHighMomentum}
	meta.TradeFrequency = strategylib.TradeFrequencyMedium
	meta.HoldingPeriod = strategylib.HoldingMedium
	meta.RiskLevel = strategylib.RiskMedium
	meta.MinimumHistory = s.WarmupBars()
	return meta
}

func (s *Strategy) DefaultParameters() map[string]any { return strategylib.CloneParams(defaultParams) }
func (s *Strategy) ParameterRanges() []strategylib.ParameterRange {
	return append([]strategylib.ParameterRange(nil), paramRanges...)
}
func (s *Strategy) Validate(params map[string]any) error {
	return strategylib.ValidateAgainstRanges(params, paramRanges)
}
func (s *Strategy) WarmupBars() int { return s.slowPeriod + s.signalPeriod + 1 }
func (s *Strategy) Parameters() map[string]any {
	return map[string]any{"fast": s.fastPeriod, "slow": s.slowPeriod, "signal": s.signalPeriod}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}
	res := indaccess.MACD(ctx, s.fastPeriod, s.slowPeriod, s.signalPeriod, s.macd)
	ind := map[string]float64{"macd": res.MACD, "macd_signal": res.Signal, "macd_histogram": res.Histogram}
	if !res.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	var prevMACD, prevSignal float64
	var prevWarmed bool
	if ctx.HasIndicatorStore() {
		prev, ok := indaccess.MACDAt(ctx, s.fastPeriod, s.slowPeriod, s.signalPeriod, ctx.BarIndex-1)
		if ok && prev.WarmedUp {
			prevMACD, prevSignal, prevWarmed = prev.MACD, prev.Signal, true
		}
	} else {
		prevMACD, prevSignal, prevWarmed = s.prevMACD, s.prevSignal, s.prevWarmed
	}

	bullish := cross.Above(prevMACD, prevSignal, res.MACD, res.Signal, prevWarmed)
	bearish := cross.Below(prevMACD, prevSignal, res.MACD, res.Signal, prevWarmed)
	if !ctx.HasIndicatorStore() {
		s.prevMACD, s.prevSignal, s.prevWarmed = res.MACD, res.Signal, true
	}
	strength := clamp01(res.Histogram / ctx.Candle.Close * 100)

	switch ctx.Position {
	case strategylib.PositionLong:
		if bearish {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"MACD crossed below signal line"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if bullish {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"MACD crossed above signal line"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if bullish {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{"MACD bullish crossover"},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if bearish {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{"MACD bearish crossover"},
				strategylib.MergeTags(Name, "sell"), ind)
		}
	}
	return builder.IgnoreWithIndicators(ind)
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
