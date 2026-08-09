package ema_pullback

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "ema_pullback"

var (
	defaultParams = map[string]any{"fast": 9, "slow": 21, "tolerance": 0.002}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "fast", Values: []any{5, 8, 9, 10, 12}},
		{Name: "slow", Values: []any{20, 21, 26, 34, 50, 55, 100, 200}},
	}
)

// Strategy buys pullbacks to fast EMA in an established slow EMA trend.
type Strategy struct {
	fastPeriod int
	slowPeriod int
	tolerance  float64
	fastEMA    *indicators.EMA
	slowEMA    *indicators.EMA
	prevSlow   float64
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	fast := strategylib.IntParam(merged, "fast", 9)
	slow := strategylib.IntParam(merged, "slow", 21)
	tolerance := strategylib.FloatParam(merged, "tolerance", 0.002)
	return &Strategy{
		fastPeriod: fast,
		slowPeriod: slow,
		tolerance:  tolerance,
		fastEMA:    indicators.NewEMA(fast),
		slowEMA:    indicators.NewEMA(slow),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Enters on pullbacks to fast EMA when slow EMA confirms trend direction."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "EMA pullback trend entry", strategylib.CategoryTrend)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeTrending}
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
func (s *Strategy) WarmupBars() int { return s.slowPeriod + 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"fast":      s.fastPeriod,
		"slow":      s.slowPeriod,
		"tolerance": s.tolerance,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	close := ctx.Candle.Close
	low := ctx.Candle.Low
	high := ctx.Candle.High
	fastRes := indaccess.EMA(ctx, s.fastPeriod, s.fastEMA)
	slowRes := indaccess.EMA(ctx, s.slowPeriod, s.slowEMA)
	ind := map[string]float64{
		fmt.Sprintf("ema_%d", s.fastPeriod): fastRes.Value,
		fmt.Sprintf("ema_%d", s.slowPeriod): slowRes.Value,
	}
	if !fastRes.WarmedUp || !slowRes.WarmedUp {
		if !ctx.HasIndicatorStore() && slowRes.WarmedUp {
			s.prevSlow = slowRes.Value
		}
		return builder.IgnoreWithIndicators(ind)
	}

	var prevSlow float64
	if ctx.HasIndicatorStore() {
		ps, ok := indaccess.EMAAt(ctx, s.slowPeriod, ctx.BarIndex-1)
		if ok && ps.WarmedUp {
			prevSlow = ps.Value
		}
	} else {
		prevSlow = s.prevSlow
	}

	uptrend := slowRes.Value > prevSlow && close > slowRes.Value
	downtrend := slowRes.Value < prevSlow && close < slowRes.Value
	if !ctx.HasIndicatorStore() {
		s.prevSlow = slowRes.Value
	}

	fast := fastRes.Value
	pullbackLong := low <= fast*(1+s.tolerance) && close > fast
	pullbackShort := high >= fast*(1-s.tolerance) && close < fast

	strength := clamp01((close - slowRes.Value) / close * 100)
	if strength < 0 {
		strength = -strength
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if close < slowRes.Value {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("Close below slow EMA%d", s.slowPeriod)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if close > slowRes.Value {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("Close above slow EMA%d", s.slowPeriod)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if uptrend && pullbackLong {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Pullback to EMA%d in uptrend", s.fastPeriod)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if downtrend && pullbackShort {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Pullback to EMA%d in downtrend", s.fastPeriod)},
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
