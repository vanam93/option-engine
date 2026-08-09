package rsi_reversal

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "rsi_reversal"

var (
	defaultParams = map[string]any{"period": 14, "overbought": 70.0, "oversold": 30.0}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "period", Values: []any{7, 14, 21}},
		{Name: "overbought", Values: []any{70.0}},
		{Name: "oversold", Values: []any{30.0}},
	}
)

// Strategy implements RSI oversold/overbought reversals.
type Strategy struct {
	period        int
	overbought    float64
	oversold      float64
	rsi           *indicators.RSI
	wasOversold   bool
	wasOverbought bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	period := strategylib.IntParam(merged, "period", 14)
	overbought := strategylib.FloatParam(merged, "overbought", 70)
	oversold := strategylib.FloatParam(merged, "oversold", 30)
	return &Strategy{
		period:     period,
		overbought: overbought,
		oversold:   oversold,
		rsi:        indicators.NewRSI(period),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Buys oversold RSI reversals and sells overbought reversals."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "RSI oversold/overbought reversal", strategylib.CategoryMeanReversion)
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
func (s *Strategy) WarmupBars() int { return s.period + 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"period":     s.period,
		"overbought": s.overbought,
		"oversold":   s.oversold,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	res := indaccess.RSI(ctx, s.period, s.rsi)
	ind := map[string]float64{"rsi": res.Value}
	if !res.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	rsi := res.Value
	if rsi <= s.oversold {
		s.wasOversold = true
	}
	if rsi >= s.overbought {
		s.wasOverbought = true
	}

	strength := clamp01((rsi - s.oversold) / (s.overbought - s.oversold))

	switch ctx.Position {
	case strategylib.PositionLong:
		if rsi >= s.overbought || (s.wasOversold && rsi >= 50) {
			s.wasOversold = false
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("RSI exit at %.1f", rsi)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if rsi <= s.oversold || (s.wasOverbought && rsi <= 50) {
			s.wasOverbought = false
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("RSI exit at %.1f", rsi)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if s.wasOversold && rsi > s.oversold {
			s.wasOversold = false
			return builder.Action(strategylib.DecisionBuy, 0.75, 1-strength, 0.72,
				[]string{fmt.Sprintf("RSI oversold reversal above %.0f", s.oversold)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if s.wasOverbought && rsi < s.overbought {
			s.wasOverbought = false
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("RSI overbought reversal below %.0f", s.overbought)},
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
