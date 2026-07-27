package market

import (
	"time"

	"github.com/google/uuid"
)

// InstrumentType classifies tradable instruments on NSE.
type InstrumentType string

const (
	InstrumentSpot   InstrumentType = "SPOT"
	InstrumentFuture InstrumentType = "FUTURE"
	InstrumentOption InstrumentType = "OPTION"
	InstrumentIndex  InstrumentType = "INDEX"
)

// Tick represents a single normalized market price update.
type Tick struct {
	ID             uuid.UUID      `json:"id"`
	Symbol         string         `json:"symbol"`
	Exchange       string         `json:"exchange"`
	InstrumentType InstrumentType `json:"instrument_type"`
	LTP            float64        `json:"ltp"`
	Open           float64        `json:"open"`
	High           float64        `json:"high"`
	Low            float64        `json:"low"`
	Close          float64        `json:"close"`
	Volume         int64          `json:"volume"`
	Bid            float64        `json:"bid"`
	Ask            float64        `json:"ask"`
	BidQty         int64          `json:"bid_qty"`
	AskQty         int64          `json:"ask_qty"`
	OI             int64          `json:"oi"`
	ProviderTS     time.Time      `json:"provider_ts"`
	ReceivedAt     time.Time      `json:"received_at"`
	SequenceNum    int64          `json:"sequence_num"`
}
