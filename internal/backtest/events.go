package backtest

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
)

const providerName = "backtest"

// PayloadFromCandle maps a historical candle to a provider-normalizer payload.
func PayloadFromCandle(c market.Candle) normalizer.Payload {
	ts := c.CloseTime
	if ts.IsZero() {
		ts = c.OpenTime
	}
	return normalizer.Payload{
		Symbol:         c.Symbol,
		Exchange:       "NSE",
		InstrumentType: market.InstrumentIndex,
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
func EventFromCandle(c market.Candle, clk clock.Clock) (events.Event, error) {
	payload := PayloadFromCandle(c)
	at := payload.Timestamp
	if clk != nil {
		at = clk.Now()
	}
	return events.NewEventWithTime(events.MarketDataReceived, providerName, payload, at)
}

// EventFromCandleAt publishes a MarketDataReceived event at an explicit timestamp.
func EventFromCandleAt(c market.Candle, at time.Time) (events.Event, error) {
	payload := PayloadFromCandle(c)
	if at.IsZero() {
		at = payload.Timestamp
	}
	return events.NewEventWithTime(events.MarketDataReceived, providerName, payload, at)
}
