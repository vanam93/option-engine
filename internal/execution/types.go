package execution

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus identifies the outcome of an order execution.
type ExecutionStatus string

const (
	Filled   ExecutionStatus = "FILLED"
	Rejected ExecutionStatus = "REJECTED"
)

// ApprovedTradeIntent is the broker-independent trade intent consumed by execution adapters.
type ApprovedTradeIntent struct {
	ID             uuid.UUID `json:"id"`
	Symbol         string    `json:"symbol"`
	Timeframe      string    `json:"timeframe"`
	Status         string    `json:"status"`
	Action         string    `json:"action"`
	Quantity       int       `json:"quantity"`
	Strategy       string    `json:"strategy"`
	ReferencePrice float64   `json:"reference_price"`
	Timestamp      time.Time `json:"timestamp"`
}

// ExecutionReport is the broker-independent execution outcome published on execution.report events.
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

// Capabilities describes what an execution adapter supports.
type Capabilities struct {
	Mode              string `json:"mode"`
	Simulated         bool   `json:"simulated"`
	MarketOrders      bool   `json:"market_orders"`
	LimitOrders       bool   `json:"limit_orders"`
	PartialFills      bool   `json:"partial_fills"`
	RealTimeExecution bool   `json:"real_time_execution"`
}
