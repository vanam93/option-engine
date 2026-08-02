package backtest

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// LoadOptions filters loaded candles by symbol and time range.
type LoadOptions struct {
	Symbols   []string
	StartTime time.Time
	EndTime   time.Time
	Timeframe market.Timeframe
}

// Load reads historical candles from a JSON file and applies filters.
func Load(path string, opts LoadOptions) ([]market.Candle, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("backtest data path required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read backtest data: %w", err)
	}
	var candles []market.Candle
	if err := json.Unmarshal(data, &candles); err != nil {
		return nil, fmt.Errorf("decode backtest data: %w", err)
	}
	return FilterCandles(candles, opts), nil
}

// FilterCandles returns candles matching symbol, timeframe, and time bounds in timestamp order.
func FilterCandles(candles []market.Candle, opts LoadOptions) []market.Candle {
	symbolSet := make(map[string]struct{}, len(opts.Symbols))
	for _, s := range opts.Symbols {
		symbolSet[strings.ToUpper(strings.TrimSpace(s))] = struct{}{}
	}

	filtered := make([]market.Candle, 0, len(candles))
	for _, c := range candles {
		if len(symbolSet) > 0 {
			if _, ok := symbolSet[strings.ToUpper(c.Symbol)]; !ok {
				continue
			}
		}
		if opts.Timeframe != "" && c.Timeframe != opts.Timeframe {
			continue
		}
		ts := candleTime(c)
		if !opts.StartTime.IsZero() && ts.Before(opts.StartTime) {
			continue
		}
		if !opts.EndTime.IsZero() && ts.After(opts.EndTime) {
			continue
		}
		filtered = append(filtered, c)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return candleTime(filtered[i]).Before(candleTime(filtered[j]))
	})
	return filtered
}

func candleTime(c market.Candle) time.Time {
	if !c.CloseTime.IsZero() {
		return c.CloseTime
	}
	return c.OpenTime
}
