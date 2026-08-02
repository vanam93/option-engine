package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Type identifies the kind of domain event.
type Type string

const (
	MarketDataReceived       Type = "market.data.received"
	TickPersisted            Type = "market.tick.persisted"
	CandleClosed             Type = "market.candle.closed"
	OptionChainUpdated       Type = "option.chain.updated"
	IndicatorUpdated         Type = "indicator.updated"
	SignalGenerated          Type = "signal.generated"
	StrategyDecision         Type = "strategy.decision"
	ContextEvaluated         Type = "context.evaluated"
	StrategySignalGenerated  Type = "strategy.signal.generated"
	DecisionMade             Type = "decision.made"
	TradeOpened              Type = "trade.opened"
	TradeUpdated             Type = "trade.updated"
	TradeClosed              Type = "trade.closed"
	AlertFired               Type = "alert.fired"
)

// Event is the universal envelope for all system events.
type Event struct {
	ID        uuid.UUID       `json:"id"`
	Type      Type            `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent creates an event with a marshaled payload using the system clock.
// Prefer NewEventWithClock in business logic.
func NewEvent(eventType Type, source string, payload any) (Event, error) {
	return NewEventWithTime(eventType, source, payload, time.Now().UTC())
}

// NewEventWithClock creates an event using the injected clock.
func NewEventWithClock(clk interface{ Now() time.Time }, eventType Type, source string, payload any) (Event, error) {
	return NewEventWithTime(eventType, source, payload, clk.Now())
}

// NewEventWithTime creates an event at an explicit timestamp.
func NewEventWithTime(eventType Type, source string, payload any, at time.Time) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:        uuid.New(),
		Type:      eventType,
		Source:    source,
		Timestamp: at.UTC(),
		Payload:   data,
	}, nil
}
