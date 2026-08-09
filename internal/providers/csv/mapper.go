package csv

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

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

// EventFromCandleAt publishes a MarketDataReceived event at an explicit timestamp.
func EventFromCandleAt(c market.Candle, inst symbolregistry.Instrument, at time.Time) (events.Event, error) {
	payload := PayloadFromCandle(c, inst)
	if at.IsZero() {
		at = payload.Timestamp
	}
	return events.NewEventWithTime(events.MarketDataReceived, providerName, payload, at)
}

// EventFromCandle publishes a MarketDataReceived event using the injected clock.
func EventFromCandle(c market.Candle, inst symbolregistry.Instrument, clk clock.Clock) (events.Event, error) {
	payload := PayloadFromCandle(c, inst)
	at := payload.Timestamp
	if clk != nil {
		at = clk.Now()
	}
	return events.NewEventWithTime(events.MarketDataReceived, providerName, payload, at)
}

// SymbolsMatch reports whether a subscribed symbol maps to the configured data symbol.
func SymbolsMatch(subscribed, dataSymbol string) bool {
	a := normalizeSymbol(subscribed)
	b := normalizeSymbol(dataSymbol)
	if a == b {
		return true
	}
	if a+"50" == b || b+"50" == a {
		return true
	}
	return false
}

func normalizeSymbol(symbol string) string {
	out := make([]byte, 0, len(symbol))
	for i := 0; i < len(symbol); i++ {
		c := symbol[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c != ' ' {
			out = append(out, c)
		}
	}
	return string(out)
}

func candleTime(c market.Candle) time.Time {
	if !c.CloseTime.IsZero() {
		return c.CloseTime
	}
	return c.OpenTime
}
