package indicator

import (
	"time"

	"github.com/google/uuid"
)

// Name identifies a technical indicator.
type Name string

const (
	EMA            Name = "EMA"
	SMA            Name = "SMA"
	RSI            Name = "RSI"
	MACD           Name = "MACD"
	ATR            Name = "ATR"
	VWAP           Name = "VWAP"
	Supertrend     Name = "SUPERTREND"
	ADX            Name = "ADX"
	BollingerBands Name = "BOLLINGER_BANDS"
	Ichimoku       Name = "ICHIMOKU"
	VolumeProfile  Name = "VOLUME_PROFILE"
)

// IndicatorValue is a typed output from the TA engine.
type IndicatorValue struct {
	ID         uuid.UUID          `json:"id"`
	Name       Name               `json:"name"`
	Symbol     string             `json:"symbol"`
	Timeframe  string             `json:"timeframe"`
	Values     map[string]float64 `json:"values"`
	ComputedAt time.Time          `json:"computed_at"`
}
