package risk

import (
	"time"

	"github.com/google/uuid"
)

// Status identifies whether a trade intent was approved or rejected.
type Status string

const (
	Approved Status = "APPROVED"
	Rejected Status = "REJECTED"
)

// ApprovedTradeIntent is the payload published on ApprovedTradeIntent events.
type ApprovedTradeIntent struct {
	ID         uuid.UUID `json:"id"`
	Symbol     string    `json:"symbol"`
	Timeframe  string    `json:"timeframe"`
	Status     Status    `json:"status"`
	Action     string    `json:"action"`
	Quantity   int       `json:"quantity"`
	Strategy   string    `json:"strategy"`
	Confidence float64   `json:"confidence"`
	Reason     string    `json:"reason"`
	Timestamp  time.Time `json:"timestamp"`
}

func newTradeIntent(
	status Status,
	action, symbol, timeframe, strategyName string,
	quantity int,
	confidence float64,
	at time.Time,
	reason string,
) ApprovedTradeIntent {
	return ApprovedTradeIntent{
		ID:         uuid.New(),
		Symbol:     symbol,
		Timeframe:  timeframe,
		Status:     status,
		Action:     action,
		Quantity:   quantity,
		Strategy:   strategyName,
		Confidence: confidence,
		Reason:     reason,
		Timestamp:  at,
	}
}
