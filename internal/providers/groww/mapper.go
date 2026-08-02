package groww

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

// GrowwSymbol builds the Groww instrument identifier from registry metadata.
func GrowwSymbol(inst symbolregistry.Instrument) string {
	exchange := inst.Exchange
	if exchange == "" {
		exchange = "NSE"
	}
	symbol := inst.Symbol
	underlying := inst.Underlying
	if underlying == "" {
		underlying = symbol
	}

	switch inst.InstrumentType {
	case market.InstrumentFuture:
		if inst.Expiry != nil {
			return fmt.Sprintf("%s-%s-%s-FUT", exchange, underlying, formatGrowwExpiry(*inst.Expiry))
		}
	case market.InstrumentOption:
		if inst.Expiry != nil && inst.Strike > 0 && inst.OptionType != "" {
			strike := formatStrike(inst.Strike)
			return fmt.Sprintf("%s-%s-%s-%s-%s", exchange, underlying, formatGrowwExpiry(*inst.Expiry), strike, inst.OptionType)
		}
	}
	if inst.InstrumentType == market.InstrumentIndex || inst.InstrumentType == market.InstrumentSpot {
		return fmt.Sprintf("%s-%s", exchange, symbol)
	}
	return fmt.Sprintf("%s-%s", exchange, symbol)
}

// ResolveSegment maps registry metadata to Groww segment values.
func ResolveSegment(inst symbolregistry.Instrument) string {
	switch inst.Segment {
	case "NFO", "FNO", "DERIVATIVES":
		return "FNO"
	default:
		if inst.InstrumentType == market.InstrumentFuture || inst.InstrumentType == market.InstrumentOption {
			return "FNO"
		}
		return "CASH"
	}
}

func formatGrowwExpiry(t time.Time) string {
	return t.Format("02Jan06")
}

func formatStrike(strike float64) string {
	if strike == float64(int64(strike)) {
		return fmt.Sprintf("%d", int64(strike))
	}
	return fmt.Sprintf("%.2f", strike)
}

// mapGrowwCandles converts Groww OHLCV arrays into domain candles.
func mapGrowwCandles(raw [][]any, symbol string, tf market.Timeframe) ([]market.Candle, error) {
	if tf == "" {
		tf = market.TF5m
	}
	loc, _ := time.LoadLocation("Asia/Kolkata")
	if loc == nil {
		loc = time.UTC
	}

	out := make([]market.Candle, 0, len(raw))
	seen := make(map[int64]struct{}, len(raw))
	for _, row := range raw {
		if len(row) < 6 {
			continue
		}
		ts, err := parseGrowwTimestamp(row[0], loc)
		if err != nil {
			continue
		}
		if _, dup := seen[ts.Unix()]; dup {
			continue
		}
		seen[ts.Unix()] = struct{}{}

		open, _ := toFloat(row[1])
		high, _ := toFloat(row[2])
		low, _ := toFloat(row[3])
		closePx, _ := toFloat(row[4])
		volume, _ := toInt64(row[5])

		out = append(out, market.Candle{
			Symbol:    symbol,
			Timeframe: tf,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePx,
			Volume:    volume,
			OpenTime:  ts,
			CloseTime: ts,
		})
	}
	return out, nil
}

func parseGrowwTimestamp(raw any, loc *time.Location) (time.Time, error) {
	switch v := raw.(type) {
	case string:
		layouts := []string{
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			time.RFC3339,
		}
		for _, layout := range layouts {
			if t, err := time.ParseInLocation(layout, v, loc); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unsupported timestamp: %s", v)
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type")
	}
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("not an integer")
	}
}

// PayloadFromCandle maps a candle into the gateway normalizer payload.
func PayloadFromCandle(c market.Candle, inst symbolregistry.Instrument) normalizer.Payload {
	ts := c.CloseTime
	if ts.IsZero() {
		ts = c.OpenTime
	}
	instType := inst.InstrumentType
	if instType == "" {
		instType = market.InstrumentIndex
	}
	exchange := inst.Exchange
	if exchange == "" {
		exchange = "NSE"
	}
	return normalizer.Payload{
		Symbol:         c.Symbol,
		Exchange:       exchange,
		InstrumentType: instType,
		LTP:            c.Close,
		Open:           c.Open,
		High:           c.High,
		Low:            c.Low,
		Close:          c.Close,
		Volume:         c.Volume,
		Bid:            c.Close - 0.25,
		Ask:            c.Close + 0.25,
		BidQty:         50,
		AskQty:         50,
		Timestamp:      ts,
	}
}

// EventFromCandle publishes a MarketDataReceived event for gateway consumption.
func EventFromCandle(c market.Candle, inst symbolregistry.Instrument, clk clock.Clock) (events.Event, error) {
	payload := PayloadFromCandle(c, inst)
	at := payload.Timestamp
	if clk != nil {
		at = clk.Now()
	}
	return events.NewEventWithTime(events.MarketDataReceived, providerName, payload, at)
}

// EventFromCandleAt publishes a MarketDataReceived event at an explicit timestamp.
func EventFromCandleAt(c market.Candle, inst symbolregistry.Instrument, at time.Time) (events.Event, error) {
	payload := PayloadFromCandle(c, inst)
	if at.IsZero() {
		at = payload.Timestamp
	}
	return events.NewEventWithTime(events.MarketDataReceived, providerName, payload, at)
}
