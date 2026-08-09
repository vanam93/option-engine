package ema_cross

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/cross"
)

const Name = "ema_cross"

var parameterRanges = []strategylib.ParameterRange{
	{Name: "fast", Values: []any{5, 8, 9, 10, 12}},
	{Name: "slow", Values: []any{20, 21, 26, 34, 50, 55, 100, 200}},
}

// Strategy implements EMA crossover entries and exits.
type Strategy struct {
	fastPeriod int
	slowPeriod int
	fastEMA    *indicators.EMA
	slowEMA    *indicators.EMA
	prevFast   float64
	prevSlow   float64
	prevWarmed bool
}

// NewDefault creates EMA cross with industry defaults (9/21).
func NewDefault() *Strategy {
	return New(nil)
}

// New creates EMA cross with optional parameters.
func New(params map[string]any) *Strategy {
	merged := strategylib.CloneParams(map[string]any{"fast": 9, "slow": 21})
	for k, v := range params {
		merged[k] = v
	}
	fast := strategylib.IntParam(merged, "fast", 9)
	slow := strategylib.IntParam(merged, "slow", 21)
	return &Strategy{
		fastPeriod: fast,
		slowPeriod: slow,
		fastEMA:    indicators.NewEMA(fast),
		slowEMA:    indicators.NewEMA(slow),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Enters on fast EMA crossing slow EMA; exits on opposite cross."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "EMA crossover trend following", strategylib.CategoryTrend)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(parameterRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), parameterRanges...)
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeTrending, strategylib.RegimeHighMomentum}
	meta.TradeFrequency = strategylib.TradeFrequencyMedium
	meta.HoldingPeriod = strategylib.HoldingMedium
	meta.RiskLevel = strategylib.RiskMedium
	meta.MinimumHistory = s.WarmupBars()
	return meta
}

func (s *Strategy) DefaultParameters() map[string]any {
	return map[string]any{"fast": 9, "slow": 21}
}

func (s *Strategy) ParameterRanges() []strategylib.ParameterRange {
	return append([]strategylib.ParameterRange(nil), parameterRanges...)
}

func (s *Strategy) Validate(params map[string]any) error {
	if err := strategylib.ValidateAgainstRanges(params, parameterRanges); err != nil {
		return err
	}
	fast := strategylib.IntParam(params, "fast", 9)
	slow := strategylib.IntParam(params, "slow", 21)
	if fast >= slow {
		return fmt.Errorf("fast period must be less than slow period")
	}
	return nil
}

func (s *Strategy) WarmupBars() int {
	return s.slowPeriod + 1
}

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{"fast": s.fastPeriod, "slow": s.slowPeriod}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	close := ctx.Candle.Close
	fastRes := s.fastEMA.Update(close)
	slowRes := s.slowEMA.Update(close)
	ind := map[string]float64{
		fmt.Sprintf("ema_%d", s.fastPeriod): fastRes.Value,
		fmt.Sprintf("ema_%d", s.slowPeriod): slowRes.Value,
	}
	if !fastRes.WarmedUp || !slowRes.WarmedUp {
		if fastRes.WarmedUp {
			s.prevFast = fastRes.Value
		}
		if slowRes.WarmedUp {
			s.prevSlow = slowRes.Value
		}
		return builder.IgnoreWithIndicators(ind)
	}

	currFast := fastRes.Value
	currSlow := slowRes.Value
	ind["ema_spread"] = currFast - currSlow
	bullish := cross.Above(s.prevFast, s.prevSlow, currFast, currSlow, s.prevWarmed)
	bearish := cross.Below(s.prevFast, s.prevSlow, currFast, currSlow, s.prevWarmed)
	s.prevFast = currFast
	s.prevSlow = currSlow
	s.prevWarmed = true

	strength := clampSpreadStrength(ind["ema_spread"], close)

	switch ctx.Position {
	case strategylib.PositionLong:
		if bearish {
			return builder.Action(strategylib.DecisionExit, 0.7, strength, 0.65,
				[]string{fmt.Sprintf("EMA%d crossed below EMA%d", s.fastPeriod, s.slowPeriod)},
				strategylib.MergeTags(Name, "exit", "bearish_cross"),
				ind,
			)
		}
	case strategylib.PositionShort:
		if bullish {
			return builder.Action(strategylib.DecisionExit, 0.7, strength, 0.65,
				[]string{fmt.Sprintf("EMA%d crossed above EMA%d", s.fastPeriod, s.slowPeriod)},
				strategylib.MergeTags(Name, "exit", "bullish_cross"),
				ind,
			)
		}
	case strategylib.PositionFlat:
		if bullish {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("EMA%d crossed above EMA%d", s.fastPeriod, s.slowPeriod)},
				strategylib.MergeTags(Name, "buy", "bullish_cross"),
				ind,
			)
		}
		if bearish {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("EMA%d crossed below EMA%d", s.fastPeriod, s.slowPeriod)},
				strategylib.MergeTags(Name, "sell", "bearish_cross"),
				ind,
			)
		}
	}
	return builder.IgnoreWithIndicators(ind)
}

func clampSpreadStrength(spread, price float64) float64 {
	if price <= 0 {
		return 0
	}
	v := spread / price * 100
	if v < 0 {
		v = -v
	}
	if v > 1 {
		return 1
	}
	return v
}
