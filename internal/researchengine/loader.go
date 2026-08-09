package researchengine

import (
	"fmt"

	csvprovider "github.com/vanam-gangireddy/option-engine/internal/providers/csv"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// LoadCSVCandles reads all candles from a CSV file path.
func LoadCSVCandles(path, symbol string, timeframe market.Timeframe) ([]market.Candle, error) {
	reader := csvprovider.NewReader(path)
	it, err := reader.OpenIterator(1000)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var out []market.Candle
	for {
		row, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		c := csvprovider.RowToCandle(row, symbol, timeframe)
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no candles loaded from %s", path)
	}
	return out, nil
}
