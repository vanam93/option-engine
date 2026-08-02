package strategy

import (
	"time"

	"github.com/google/uuid"
)

// Decision identifies a strategy output state.
type Decision string

const (
	LongEntry  Decision = "LONG_ENTRY"
	ShortEntry Decision = "SHORT_ENTRY"
	LongExit   Decision = "LONG_EXIT"
	ShortExit  Decision = "SHORT_EXIT"
	Hold       Decision = "HOLD"
)

// StrategyDecision is the payload published on StrategyDecision events.
type StrategyDecision struct {
	ID         uuid.UUID `json:"id"`
	Symbol     string    `json:"symbol"`
	Timeframe  string    `json:"timeframe"`
	Decision   Decision  `json:"decision"`
	Strategy   string    `json:"strategy"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
	Reason     string    `json:"reason"`
}

func newStrategyDecision(
	strategyName string,
	decision Decision,
	symbol, timeframe string,
	confidence float64,
	at time.Time,
	reason string,
) StrategyDecision {
	return StrategyDecision{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  timeframe,
		Decision:   decision,
		Strategy:   strategyName,
		Confidence: confidence,
		Timestamp:  at,
		Reason:     reason,
	}
}
