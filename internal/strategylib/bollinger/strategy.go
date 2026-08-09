package bollinger

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "bollinger"

var (
	defaultParams = map[string]any{"period": 20, "stddev": 2.0}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "period", Values: []any{20}},
		{Name: "stddev", Values: []any{2.0}},
	}
)

// Strategy implements Bollinger Band mean reversion.
type Strategy struct {
	period       int
	stddev       float64
	bands        *indicators.Bollinger
	touchedLower bool
	touchedUpper bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	period := strategylib.IntParam(merged, "period", 20)
	stddev := strategylib.FloatParam(merged, "stddev", 2.0)
	return &Strategy{
		period: period,
		stddev: stddev,
		bands:  indicators.NewBollinger(period, stddev),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Mean reversion from Bollinger Band extremes back toward the middle band."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "Bollinger Band mean reversion", strategylib.CategoryMeanReversion)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeSideways, strategylib.RegimeLowVolatility}
	meta.TradeFrequency = strategylib.TradeFrequencyMedium
	meta.HoldingPeriod = strategylib.HoldingShort
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
func (s *Strategy) WarmupBars() int { return s.period }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"period": s.period,
		"stddev": s.stddev,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	close := ctx.Candle.Close
	res := indaccess.Bollinger(ctx, s.period, s.stddev, s.bands)
	ind := map[string]float64{
		"bb_upper":  res.Upper,
		"bb_middle": res.Middle,
		"bb_lower":  res.Lower,
	}
	if !res.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	if close <= res.Lower {
		s.touchedLower = true
	}
	if close >= res.Upper {
		s.touchedUpper = true
	}

	bandWidth := res.Upper - res.Lower
	strength := 0.5
	if bandWidth > 0 {
		strength = clamp01((close - res.Lower) / bandWidth)
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if close >= res.Middle || close >= res.Upper {
			s.touchedLower = false
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"Price reached middle or upper Bollinger band"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if close <= res.Middle || close <= res.Lower {
			s.touchedUpper = false
			return builder.Action(strategylib.DecisionExit, 0.72, 1-strength, 0.68,
				[]string{"Price reached middle or lower Bollinger band"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if s.touchedLower && close > res.Lower {
			s.touchedLower = false
			return builder.Action(strategylib.DecisionBuy, 0.75, 1-strength, 0.72,
				[]string{fmt.Sprintf("Bounce from lower band at %.2f", res.Lower)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if s.touchedUpper && close < res.Upper {
			s.touchedUpper = false
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Rejection from upper band at %.2f", res.Upper)},
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
