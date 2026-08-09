package trend_following

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "trend_following"

var (
	defaultParams = map[string]any{
		"fast":          9,
		"slow":          21,
		"adx_period":    14,
		"adx_threshold": 25.0,
	}
	paramRanges = []strategylib.ParameterRange{
		{Name: "fast", Values: []any{5, 8, 9, 10, 12}},
		{Name: "slow", Values: []any{20, 21, 26, 34, 50, 55, 100, 200}},
		{Name: "adx_period", Values: []any{14}},
	}
)

// Strategy combines EMA trend with ADX confirmation.
type Strategy struct {
	fastPeriod int
	slowPeriod int
	adxPeriod  int
	threshold  float64
	fastEMA    *indicators.EMA
	slowEMA    *indicators.EMA
	adx        *indicators.ADX
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	fast := strategylib.IntParam(merged, "fast", 9)
	slow := strategylib.IntParam(merged, "slow", 21)
	adxPeriod := strategylib.IntParam(merged, "adx_period", 14)
	threshold := strategylib.FloatParam(merged, "adx_threshold", 25.0)
	return &Strategy{
		fastPeriod: fast,
		slowPeriod: slow,
		adxPeriod:  adxPeriod,
		threshold:  threshold,
		fastEMA:    indicators.NewEMA(fast),
		slowEMA:    indicators.NewEMA(slow),
		adx:        indicators.NewADX(adxPeriod),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Trend entries when fast EMA leads slow EMA and ADX confirms strength."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "EMA trend with ADX confirmation", strategylib.CategoryTrend)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeTrending, strategylib.RegimeHighMomentum}
	meta.TradeFrequency = strategylib.TradeFrequencyLow
	meta.HoldingPeriod = strategylib.HoldingLong
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
func (s *Strategy) WarmupBars() int {
	warmup := s.slowPeriod + 1
	adxWarmup := s.adxPeriod*2 + 1
	if adxWarmup > warmup {
		return adxWarmup
	}
	return warmup
}

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"fast":          s.fastPeriod,
		"slow":          s.slowPeriod,
		"adx_period":    s.adxPeriod,
		"adx_threshold": s.threshold,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	c := ctx.Candle
	fastRes := s.fastEMA.Update(c.Close)
	slowRes := s.slowEMA.Update(c.Close)
	adxRes := s.adx.Update(c.High, c.Low, c.Close)
	ind := map[string]float64{
		fmt.Sprintf("ema_%d", s.fastPeriod): fastRes.Value,
		fmt.Sprintf("ema_%d", s.slowPeriod): slowRes.Value,
		"adx":      adxRes.ADX,
		"plus_di":  adxRes.PlusDI,
		"minus_di": adxRes.MinusDI,
	}
	if !fastRes.WarmedUp || !slowRes.WarmedUp || !adxRes.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	bullish := fastRes.Value > slowRes.Value && adxRes.ADX >= s.threshold && adxRes.PlusDI > adxRes.MinusDI
	bearish := fastRes.Value < slowRes.Value && adxRes.ADX >= s.threshold && adxRes.MinusDI > adxRes.PlusDI

	spread := fastRes.Value - slowRes.Value
	strength := clamp01(spread / c.Close * 100)
	if strength < 0 {
		strength = -strength
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if !bullish {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"EMA/ADX bullish trend no longer intact"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if !bearish {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"EMA/ADX bearish trend no longer intact"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if bullish {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("EMA%d > EMA%d with ADX %.1f", s.fastPeriod, s.slowPeriod, adxRes.ADX)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if bearish {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("EMA%d < EMA%d with ADX %.1f", s.fastPeriod, s.slowPeriod, adxRes.ADX)},
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
