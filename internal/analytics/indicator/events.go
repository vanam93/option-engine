package indicator

import (
	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

func newIndicatorValue(name domainindicator.Name, candle market.Candle, period int, result indicators.Result) domainindicator.IndicatorValue {
	values := map[string]float64{
		"value":       result.Value,
		"period":      float64(period),
		"warmed_up":   1,
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
