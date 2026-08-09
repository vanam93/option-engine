package supertrend

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/indaccess"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "supertrend"

var (
	defaultParams = map[string]any{"atr_period": 10, "multiplier": 3.0}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "atr_period", Values: []any{10}},
		{Name: "multiplier", Values: []any{3.0}},
	}
)

// Strategy trades SuperTrend direction changes.
type Strategy struct {
	atrPeriod   int
	multiplier  float64
	st          *indicators.SuperTrend
	prevDir     int
	initialized bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	atrPeriod := strategylib.IntParam(merged, "atr_period", 10)
	multiplier := strategylib.FloatParam(merged, "multiplier", 3.0)
	return &Strategy{
		atrPeriod:  atrPeriod,
		multiplier: multiplier,
		st:         indicators.NewSuperTrend(atrPeriod, multiplier),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Enters on SuperTrend direction flips; exits on opposite flip."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "SuperTrend direction flip", strategylib.CategoryTrend)
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
func (s *Strategy) WarmupBars() int { return s.atrPeriod + 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"atr_period": s.atrPeriod,
		"multiplier": s.multiplier,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	c := ctx.Candle
	res := indaccess.SuperTrend(ctx, s.atrPeriod, s.multiplier, s.st)
	ind := map[string]float64{
		"supertrend":            res.Value,
		"supertrend_direction": float64(res.Direction),
	}
	if !res.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	dir := res.Direction
	var prevDir int
	var initialized bool
	if ctx.HasIndicatorStore() {
		prev, ok := indaccess.SuperTrendAt(ctx, s.atrPeriod, s.multiplier, ctx.BarIndex-1)
		if ok && prev.WarmedUp {
			prevDir, initialized = prev.Direction, true
		}
	} else {
		prevDir, initialized = s.prevDir, s.initialized
	}

	if !initialized {
		if !ctx.HasIndicatorStore() {
			s.prevDir = dir
			s.initialized = true
		}
		return builder.IgnoreWithIndicators(ind)
	}

	flipUp := prevDir < 0 && dir > 0
	flipDown := prevDir > 0 && dir < 0
	if !ctx.HasIndicatorStore() {
		s.prevDir = dir
	}

	strength := clamp01((c.Close - res.Value) / c.Close * 100)
	if strength < 0 {
		strength = -strength
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if flipDown {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"SuperTrend flipped bearish"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if flipUp {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"SuperTrend flipped bullish"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if flipUp {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("SuperTrend bullish flip at %.2f", res.Value)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if flipDown {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("SuperTrend bearish flip at %.2f", res.Value)},
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
