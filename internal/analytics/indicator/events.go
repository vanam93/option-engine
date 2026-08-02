package indicator

import (
	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

func newIndicatorValue(name domainindicator.Name, candle market.Candle, period int, result indicators.Result) domainindicator.IndicatorValue {
	values := map[string]float64{
		"value":        result.Value,
		"period":       float64(period),
		"warmed_up":    1,
		"sample_count": float64(result.Samples),
	}
	return domainindicator.IndicatorValue{
		ID:         uuid.New(),
		Name:       name,
		Symbol:     candle.Symbol,
		Timeframe:  string(candle.Timeframe),
		Values:     values,
		ComputedAt: candle.CloseTime,
	}
}

func newMACDIndicatorValue(candle market.Candle, cfg *MACDConfig, result indicators.MACDResult) domainindicator.IndicatorValue {
	values := map[string]float64{
		"macd":           result.MACD,
		"signal":         result.Signal,
		"histogram":      result.Histogram,
		"fast_period":    float64(cfg.FastPeriod),
		"slow_period":    float64(cfg.SlowPeriod),
		"signal_period":  float64(cfg.SignalPeriod),
		"warmed_up":      1,
		"sample_count":   float64(result.Samples),
	}
	return domainindicator.IndicatorValue{
		ID:         uuid.New(),
		Name:       domainindicator.MACD,
		Symbol:     candle.Symbol,
		Timeframe:  string(candle.Timeframe),
		Values:     values,
		ComputedAt: candle.CloseTime,
	}
}

func newBollingerIndicatorValue(candle market.Candle, cfg *BollingerConfig, result indicators.BollingerResult) domainindicator.IndicatorValue {
	values := map[string]float64{
		"upper":        result.Upper,
		"middle":       result.Middle,
		"lower":        result.Lower,
		"bandwidth":    result.BandWidth,
		"percent_b":    result.PercentB,
		"period":       float64(cfg.Period),
		"stddev":       cfg.StdDev,
		"warmed_up":    1,
		"sample_count": float64(result.Samples),
	}
	return domainindicator.IndicatorValue{
		ID:         uuid.New(),
		Name:       domainindicator.BollingerBands,
		Symbol:     candle.Symbol,
		Timeframe:  string(candle.Timeframe),
		Values:     values,
		ComputedAt: candle.CloseTime,
	}
}
