package paper

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus identifies the outcome of a simulated order.
type ExecutionStatus string

const (
	Filled   ExecutionStatus = "FILLED"
	Rejected ExecutionStatus = "REJECTED"
)

// InputIntent mirrors the ApprovedTradeIntent payload consumed by the paper engine.
type InputIntent struct {
	ID             uuid.UUID
	Symbol         string
	Timeframe      string
	Status         string
	Action         string
	Quantity       int
	Strategy       string
	ReferencePrice float64
	Timestamp      time.Time
}

// ExecutionReport is the payload published on ExecutionReport events.
type ExecutionReport struct {
	OrderID         string          `json:"order_id"`
	Symbol          string          `json:"symbol"`
	Timeframe       string          `json:"timeframe"`
	Action          string          `json:"action"`
	Quantity        int             `json:"quantity"`
	ExecutionPrice  float64         `json:"execution_price"`
	Status          ExecutionStatus `json:"status"`
	Strategy        string          `json:"strategy"`
	Timestamp       time.Time       `json:"timestamp"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
}
