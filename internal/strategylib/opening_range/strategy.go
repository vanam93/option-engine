package opening_range

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "opening_range"

var (
	defaultParams = map[string]any{"window_minutes": 15}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "window_minutes", Values: []any{5, 15, 30}},
	}
)

// Strategy trades breakouts of the session opening range.
type Strategy struct {
	windowMinutes int
	orTracker     *indicators.OpeningRange
	brokeHigh     bool
	brokeLow      bool
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	window := strategylib.IntParam(merged, "window_minutes", 15)
	return &Strategy{
		windowMinutes: window,
		orTracker:     indicators.NewOpeningRange(window),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Breakout above or below the session opening range after the window closes."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "Opening range breakout", strategylib.CategoryBreakout)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.SupportedTimeframes = []string{"1m", "3m", "5m", "15m"}
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeGapDay, strategylib.RegimeHighMomentum, strategylib.RegimeVolatile}
	meta.TradeFrequency = strategylib.TradeFrequencyLow
	meta.HoldingPeriod = strategylib.HoldingShort
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
func (s *Strategy) WarmupBars() int { return 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"window_minutes": s.windowMinutes,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	c := ctx.Candle
	orRes := s.orTracker.Update(c.OpenTime, c.High, c.Low)
	ind := map[string]float64{
		"or_high": orRes.High,
		"or_low":  orRes.Low,
	}
	if !orRes.WindowClosed {
		return builder.IgnoreWithIndicators(ind)
	}

	close := c.Close
	orHigh := orRes.High
	orLow := orRes.Low

	rangeWidth := orHigh - orLow
	strength := 0.5
	if rangeWidth > 0 {
		strength = clamp01((close - orLow) / rangeWidth)
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if close < orLow {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{fmt.Sprintf("Close below opening range low %.2f", orLow)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if close > orHigh {
			return builder.Action(strategylib.DecisionExit, 0.72, 1-strength, 0.68,
				[]string{fmt.Sprintf("Close above opening range high %.2f", orHigh)},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if close > orHigh && !s.brokeHigh {
			s.brokeHigh = true
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("Breakout above opening range high %.2f", orHigh)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if close < orLow && !s.brokeLow {
			s.brokeLow = true
			return builder.Action(strategylib.DecisionSell, 0.75, 1-strength, 0.72,
				[]string{fmt.Sprintf("Breakdown below opening range low %.2f", orLow)},
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
