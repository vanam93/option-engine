package portfolio

import "time"

// Side identifies a position direction.
type Side string

const (
	SideLong  Side = "LONG"
	SideShort Side = "SHORT"
)

const (
	actionLongEntry  = "LONG_ENTRY"
	actionShortEntry = "SHORT_ENTRY"
	actionLongExit   = "LONG_EXIT"
	actionShortExit  = "SHORT_EXIT"
)

const statusFilled = "FILLED"

// Position is an open portfolio position.
type Position struct {
	Symbol       string    `json:"symbol"`
	Timeframe    string    `json:"timeframe"`
	Side         Side      `json:"side"`
	Quantity     int       `json:"quantity"`
	AveragePrice float64   `json:"average_price"`
	EntryTime    time.Time `json:"entry_time"`
}

// TradeRecord captures a processed filled execution.
type TradeRecord struct {
	OrderID        string    `json:"order_id"`
	Symbol         string    `json:"symbol"`
	Timeframe      string    `json:"timeframe"`
	Action         string    `json:"action"`
	Quantity       int       `json:"quantity"`
	ExecutionPrice float64   `json:"execution_price"`
	Strategy       string    `json:"strategy"`
	Timestamp      time.Time `json:"timestamp"`
}

// PortfolioUpdated is the payload published on portfolio.updated events.
type PortfolioUpdated struct {
	Symbol        string     `json:"symbol"`
	Position      *Position  `json:"position,omitempty"`
	RealizedPnL   float64    `json:"realized_pnl"`
	UnrealizedPnL float64    `json:"unrealized_pnl"`
	Timestamp     time.Time  `json:"timestamp"`
}

// PortfolioState is an immutable snapshot of portfolio state.
type PortfolioState struct {
	Positions     []Position
	Trades        []TradeRecord
	RealizedPnL   float64
	UnrealizedPnL float64
	Exposure      float64
}

// InputReport mirrors the ExecutionReport payload consumed by the portfolio engine.
type InputReport struct {
	OrderID        string
	Symbol         string
	Timeframe      string
	Action         string
	Quantity       int
	ExecutionPrice float64
	Status         string
	Strategy       string
	Timestamp      time.Time
}
