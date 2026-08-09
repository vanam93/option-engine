package adx_trend

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "adx_trend"

var (
	defaultParams = map[string]any{"period": 14, "threshold": 25.0}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "period", Values: []any{14}},
		{Name: "threshold", Values: []any{25.0}},
	}
)

// Strategy trades when ADX confirms directional movement.
type Strategy struct {
	period    int
	threshold float64
	adx       *indicators.ADX
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	period := strategylib.IntParam(merged, "period", 14)
	threshold := strategylib.FloatParam(merged, "threshold", 25.0)
	return &Strategy{
		period:    period,
		threshold: threshold,
		adx:       indicators.NewADX(period),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Trend entries when ADX exceeds threshold and directional indicators align."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "ADX trend strength filter", strategylib.CategoryTrend)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeTrending, strategylib.RegimeHighMomentum}
	meta.TradeFrequency = strategylib.TradeFrequencyLow
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
func (s *Strategy) WarmupBars() int { return s.period*2 + 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"period":    s.period,
		"threshold": s.threshold,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	res := indaccess.ADX(ctx, s.period, s.adx)
	ind := map[string]float64{
		"adx":      res.ADX,
		"plus_di":  res.PlusDI,
		"minus_di": res.MinusDI,
	}
	if !res.WarmedUp || res.ADX < s.threshold {
		return builder.IgnoreWithIndicators(ind)
	}

	bullish := res.PlusDI > res.MinusDI
	bearish := res.MinusDI > res.PlusDI
	strength := clamp01(res.ADX / 100)

	switch ctx.Position {
	case strategylib.PositionLong:
		if bearish || res.ADX < s.threshold {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("ADX trend weakened (ADX=%.1f)", res.ADX)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if bullish || res.ADX < s.threshold {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("ADX trend weakened (ADX=%.1f)", res.ADX)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if bullish {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("ADX %.1f above threshold with +DI > -DI", res.ADX)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if bearish {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("ADX %.1f above threshold with -DI > +DI", res.ADX)},
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
