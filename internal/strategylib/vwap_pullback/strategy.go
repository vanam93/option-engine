package vwap_pullback

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "vwap_pullback"

var (
	defaultParams = map[string]any{"tolerance": 0.003, "trend_period": 21}
	paramRanges   = []strategylib.ParameterRange{
		{Name: "trend_period", Values: []any{21}},
	}
)

// Strategy trades pullbacks to session VWAP in the direction of price vs VWAP.
type Strategy struct {
	tolerance   float64
	trendPeriod int
	vwap        *indicators.SessionVWAP
	trendEMA    *indicators.EMA
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	tolerance := strategylib.FloatParam(merged, "tolerance", 0.003)
	trendPeriod := strategylib.IntParam(merged, "trend_period", 21)
	return &Strategy{
		tolerance:   tolerance,
		trendPeriod: trendPeriod,
		vwap:        indicators.NewSessionVWAP(),
		trendEMA:    indicators.NewEMA(trendPeriod),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Session VWAP pullback entries with EMA trend confirmation."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "Session VWAP pullback with EMA filter", strategylib.CategoryMeanReversion)
	meta.DefaultParameters = s.DefaultParameters()
	meta.OptimizableParameters = strategylib.OptimizableNames(paramRanges)
	meta.ParameterRanges = append([]strategylib.ParameterRange(nil), paramRanges...)
	meta.SupportedTimeframes = []string{"1m", "3m", "5m", "15m", "30m"}
	meta.PreferredRegimes = []strategylib.Regime{strategylib.RegimeSideways, strategylib.RegimeTrending}
	meta.TradeFrequency = strategylib.TradeFrequencyHigh
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
func (s *Strategy) WarmupBars() int { return s.trendPeriod + 1 }

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"tolerance":    s.tolerance,
		"trend_period": s.trendPeriod,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	c := ctx.Candle
	vwapRes := s.vwap.Update(c.OpenTime, c.High, c.Low, c.Close, c.Volume)
	trendRes := s.trendEMA.Update(c.Close)
	ind := map[string]float64{
		"vwap":                  vwapRes.Value,
		fmt.Sprintf("ema_%d", s.trendPeriod): trendRes.Value,
	}
	if !vwapRes.WarmedUp || !trendRes.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	vwap := vwapRes.Value
	close := c.Close
	low := c.Low
	high := c.High
	uptrend := close > trendRes.Value
	downtrend := close < trendRes.Value
	pullbackToVWAPLong := low <= vwap*(1+s.tolerance) && close > vwap
	pullbackToVWAPShort := high >= vwap*(1-s.tolerance) && close < vwap

	strength := clamp01((close - vwap) / close * 100)
	if strength < 0 {
		strength = -strength
	}

	switch ctx.Position {
	case strategylib.PositionLong:
		if close < vwap*(1-s.tolerance) {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"Close below VWAP tolerance band"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if close > vwap*(1+s.tolerance) {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"Close above VWAP tolerance band"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if uptrend && pullbackToVWAPLong {
			return builder.Action(strategylib.DecisionBuy, 0.75, strength, 0.72,
				[]string{"VWAP pullback in uptrend"},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if downtrend && pullbackToVWAPShort {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{"VWAP pullback in downtrend"},
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
