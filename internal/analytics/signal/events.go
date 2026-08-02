package signal

import (
	"time"

	"github.com/google/uuid"
)

// Type identifies a trading signal direction.
type Type string

const (
	Buy       Type = "BUY"
	Sell      Type = "SELL"
	ExitLong  Type = "EXIT_LONG"
	ExitShort Type = "EXIT_SHORT"
	Neutral   Type = "NEUTRAL"
)

// GeneratedSignal is the payload published on SignalGenerated events.
type GeneratedSignal struct {
	ID          uuid.UUID          `json:"id"`
	Symbol      string             `json:"symbol"`
	Timeframe   string             `json:"timeframe"`
	Signal      Type               `json:"signal"`
	Strategy    string             `json:"strategy"`
	Confidence  float64            `json:"confidence"`
	Timestamp   time.Time          `json:"timestamp"`
	Indicators  map[string]float64 `json:"indicators"`
}

func newGeneratedSignal(
	strategy string,
	signalType Type,
	symbol, timeframe string,
	confidence float64,
	at time.Time,
	indicators map[string]float64,
) GeneratedSignal {
	values := make(map[string]float64, len(indicators))
	for k, v := range indicators {
		values[k] = v
	}
	return GeneratedSignal{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  timeframe,
		Signal:     signalType,
		Strategy:   strategy,
		Confidence: confidence,
		Timestamp:  at,
		Indicators: values,
	}
}
