package researchengine

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine/indicatorstore"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// Dataset holds an immutable candle slice and a shared lazy indicator cache.
type Dataset struct {
	Candles []market.Candle
	Store   *indicatorstore.Store
}

// NewDataset creates a research dataset with a shared indicator store.
func NewDataset(candles []market.Candle) *Dataset {
	return &Dataset{
		Candles: candles,
		Store:   indicatorstore.New(candles),
	}
}

// IndicatorSource returns the store as the strategylib indicator interface.
func (d *Dataset) IndicatorSource() strategylib.IndicatorSource {
	if d == nil || d.Store == nil {
		return nil
	}
	return d.Store
}
