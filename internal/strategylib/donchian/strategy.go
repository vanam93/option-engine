package donchian

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "donchian"

var (
	defaultParams = map[string]any{"period": 20}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "period", Values: []any{20, 55}},
	}
)

// Strategy trades Donchian channel breakouts.
type Strategy struct {
	period    int
	channel   *indicators.DonchianChannel
	prevUpper float64
	prevLower float64
	prevReady bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	period := strategylib.IntParam(merged, "period", 20)
	return &Strategy{
		period:  period,
		channel: indicators.NewDonchianChannel(period),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Enters on close breaking the Donchian channel; exits on opposite break."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "Donchian channel breakout", strategylib.CategoryBreakout)
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
func (s *Strategy) WarmupBars() int { return s.period + 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"period": s.period,
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

	res := indaccess.Donchian(ctx, s.period, s.channel)
	ind := map[string]float64{
		"donchian_upper": prevUpper,
		"donchian_lower": prevLower,
	}
	if res.WarmedUp {
		if !ctx.HasIndicatorStore() {
			s.prevUpper = res.Upper
			s.prevLower = res.Lower
			s.prevReady = true
		}
		ind["donchian_upper"] = res.Upper
		ind["donchian_lower"] = res.Lower
	}
	if !prevReady {
		return builder.IgnoreWithIndicators(ind)
	}

	close := c.Close
	upper := prevUpper
	lower := prevLower

	rangeWidth := upper - lower
	strength := 0.5
	if rangeWidth > 0 {
		strength = clamp01((close - lower) / rangeWidth)
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
			return builder.Action(strategylib.DecisionExit, 0.72, 1-strength, 0.68,
				[]string{fmt.Sprintf("Close above Donchian upper %.2f", upper)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if close > upper {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Close broke above Donchian upper %.2f", upper)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if close < lower {
			return builder.Action(strategylib.DecisionSell, 0.75, 1-strength, 0.72,
				[]string{fmt.Sprintf("Close broke below Donchian lower %.2f", lower)},
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
