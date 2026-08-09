package mean_reversion

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/internal/stratutil"
)

const Name = "mean_reversion"

var (
	defaultParams = map[string]any{
		"rsi_period":       14,
		"oversold":         30.0,
		"overbought":       70.0,
		"bollinger_period": 20,
		"stddev":           2.0,
	}
	paramRanges = []strategylib.ParameterRange{
		{Name: "rsi_period", Values: []any{7, 14, 21}},
		{Name: "bollinger_period", Values: []any{20}},
		{Name: "stddev", Values: []any{2.0}},
	}
)

// Strategy combines RSI and Bollinger Band mean reversion signals.
type Strategy struct {
	rsiPeriod       int
	oversold        float64
	overbought      float64
	bollingerPeriod int
	stddev          float64
	rsi             *indicators.RSI
	bands           *indicators.Bollinger
}

func NewDefault() *Strategy { return New(nil) }

func New(params map[string]any) *Strategy {
	merged := stratutil.MergeParams(defaultParams, params)
	rsiPeriod := strategylib.IntParam(merged, "rsi_period", 14)
	oversold := strategylib.FloatParam(merged, "oversold", 30.0)
	overbought := strategylib.FloatParam(merged, "overbought", 70.0)
	bbPeriod := strategylib.IntParam(merged, "bollinger_period", 20)
	stddev := strategylib.FloatParam(merged, "stddev", 2.0)
	return &Strategy{
		rsiPeriod:       rsiPeriod,
		oversold:        oversold,
		overbought:      overbought,
		bollingerPeriod: bbPeriod,
		stddev:          stddev,
		rsi:             indicators.NewRSI(rsiPeriod),
		bands:           indicators.NewBollinger(bbPeriod, stddev),
	}
}

func (s *Strategy) Name() string { return Name }

func (s *Strategy) Description() string {
	return "Mean reversion when RSI and Bollinger Bands both signal stretched prices."
}

func (s *Strategy) Metadata() strategylib.Metadata {
	meta := strategylib.BaseMetadata(Name, s.Description(), "RSI and Bollinger combined mean reversion", strategylib.CategoryMeanReversion)
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
func (s *Strategy) WarmupBars() int {
	warmup := s.rsiPeriod + 1
	if s.bollingerPeriod > warmup {
		return s.bollingerPeriod
	}
	return warmup
}

func (s *Strategy) Parameters() map[string]any {
	return map[string]any{
		"rsi_period":       s.rsiPeriod,
		"oversold":         s.oversold,
		"overbought":       s.overbought,
		"bollinger_period": s.bollingerPeriod,
		"stddev":           s.stddev,
	}
}

func (s *Strategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	builder := strategylib.NewSignalBuilder(s.Parameters(), ctx.TimestampOrCandle())
	if !strategylib.ValidCandle(ctx.Candle) {
		return builder.Ignore()
	}

	close := ctx.Candle.Close
	rsiRes := s.rsi.Update(close)
	bbRes := s.bands.Update(close)
	ind := map[string]float64{
		"rsi":       rsiRes.Value,
		"bb_upper":  bbRes.Upper,
		"bb_middle": bbRes.Middle,
		"bb_lower":  bbRes.Lower,
	}
	if !rsiRes.WarmedUp || !bbRes.WarmedUp {
		return builder.IgnoreWithIndicators(ind)
	}

	oversold := rsiRes.Value <= s.oversold && close <= bbRes.Lower
	overbought := rsiRes.Value >= s.overbought && close >= bbRes.Upper
	normalized := rsiRes.Value > s.oversold && rsiRes.Value < s.overbought &&
		close > bbRes.Lower && close < bbRes.Upper

	strength := clamp01((rsiRes.Value - s.oversold) / (s.overbought - s.oversold))

	switch ctx.Position {
	case strategylib.PositionLong:
		if overbought || normalized {
			return builder.Action(strategylib.DecisionExit, 0.72, strength, 0.68,
				[]string{"RSI and Bollinger mean reversion target reached"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionShort:
		if oversold || normalized {
			return builder.Action(strategylib.DecisionExit, 0.72, 1-strength, 0.68,
				[]string{"RSI and Bollinger mean reversion target reached"},
				strategylib.MergeTags(Name, "exit"), ind)
		}
	case strategylib.PositionFlat:
		if oversold {
			return builder.Action(strategylib.DecisionBuy, 0.75, 1-strength, 0.72,
				[]string{fmt.Sprintf("RSI %.1f oversold with price at lower band", rsiRes.Value)},
				strategylib.MergeTags(Name, "buy"), ind)
		}
		if overbought {
			return builder.Action(strategylib.DecisionSell, 0.75, strength, 0.72,
				[]string{fmt.Sprintf("RSI %.1f overbought with price at upper band", rsiRes.Value)},
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
