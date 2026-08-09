package breakout

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "breakout"

var (
	defaultParams = map[string]any{"period": 20, "atr_period": 14, "atr_multiple": 1.0}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "period", Values: []any{20, 55}},
		{Name: "atr_period", Values: []any{14}},
	}
)

// Strategy trades range breakouts confirmed by ATR expansion.
type Strategy struct {
	period      int
	atrPeriod   int
	atrMultiple float64
	channel     *indicators.DonchianChannel
	atr         *indicators.ATR
	baseATR     float64
	baseATRSet  bool
	prevUpper   float64
	prevLower   float64
	prevReady   bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	period := strategylib.IntParam(merged, "period", 20)
	atrPeriod := strategylib.IntParam(merged, "atr_period", 14)
	atrMultiple := strategylib.FloatParam(merged, "atr_multiple", 1.0)
	return &Strategy{
		period:      period,
		atrPeriod:   atrPeriod,
		atrMultiple: atrMultiple,
		channel:     indicators.NewDonchianChannel(period),
		atr:         indicators.NewATR(atrPeriod),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Range breakout entries when price breaks Donchian channel with ATR expansion."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "Donchian breakout with ATR expansion filter", strategylib.CategoryBreakout)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeVolatile, strategylib.RegimeHighMomentum}
	meta.TradeFrequency = strategylib.TradeFrequencyLow
	meta.HoldingPeriod = strategylib.HoldingMedium
	meta.RiskLevel = strategylib.RiskHigh
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
	if s.period > s.atrPeriod {
		return s.period + 1
	}
	return s.atrPeriod + 1
}

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"period":       s.period,
		"atr_period":   s.atrPeriod,
		"atr_multiple": s.atrMultiple,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	c := ctx.Candle
	var prevUpper, prevLower float64
	var prevReady bool

	if ctx.HasIndicatorStore() {
		prev, ok := indaccess.DonchianAt(ctx, s.period, ctx.BarIndex-1)
		if ok && prev.WarmedUp {
			prevUpper, prevLower, prevReady = prev.Upper, prev.Lower, true
		}
	} else {
		prevUpper, prevLower, prevReady = s.prevUpper, s.prevLower, s.prevReady
	}

	chRes := indaccess.Donchian(ctx, s.period, s.channel)
	atrRes := indaccess.ATR(ctx, s.atrPeriod, s.atr)
	ind := map[string]float64{
		"donchian_upper": prevUpper,
		"donchian_lower": prevLower,
		"atr":            atrRes.Value,
	}
	if chRes.WarmedUp {
		if !ctx.HasIndicatorStore() {
			s.prevUpper = chRes.Upper
			s.prevLower = chRes.Lower
			s.prevReady = true
		}
		ind["donchian_upper"] = chRes.Upper
		ind["donchian_lower"] = chRes.Lower
	}
	if !prevReady || !atrRes.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	if !s.baseATRSet {
		s.baseATR = atrRes.Value
		s.baseATRSet = true
	}
	expanded := atrRes.Value >= s.baseATR*s.atrMultiple
	ind["atr_expanded"] = 0
	if expanded {
		ind["atr_expanded"] = 1
	}
	close := c.Close
	upper := prevUpper
	lower := prevLower

	strength := 0.5
	if s.baseATR > 0 {
		strength = clamp01(atrRes.Value / s.baseATR / 2)
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if close < lower {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("Close below Donchian lower %.2f", lower)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if close > upper {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("Close above Donchian upper %.2f", upper)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if expanded && close > upper {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Bullish breakout above %.2f with ATR expansion", upper)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if expanded && close < lower {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Bearish breakout below %.2f with ATR expansion", lower)},
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
