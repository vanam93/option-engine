package market

import (
	"time"

	"github.com/google/uuid"
)

// Timeframe defines candle aggregation intervals.
type Timeframe string

const (
	TF1m  Timeframe = "1m"
	TF3m  Timeframe = "3m"
	TF5m  Timeframe = "5m"
	TF15m Timeframe = "15m"
	TF30m Timeframe = "30m"
	TF1h  Timeframe = "1h"
	TF1d  Timeframe = "1d"
)

// Candle is an OHLCV bar for a symbol and timeframe.
type Candle struct {
	ID         uuid.UUID `json:"id"`
	Symbol     string    `json:"symbol"`
	Timeframe  Timeframe `json:"timeframe"`
	Open       float64   `json:"open"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Close      float64   `json:"close"`
	Volume     int64     `json:"volume"`
	VWAP       float64   `json:"vwap"`
	TradeCount int64     `json:"trade_count"`
	OpenTime   time.Time `json:"open_time"`
	CloseTime  time.Time `json:"close_time"`
}
