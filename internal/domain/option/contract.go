package option

import (
	"time"

	"github.com/google/uuid"
)

// OptionType is CALL or PUT.
type OptionType string

const (
	Call OptionType = "CE"
	Put  OptionType = "PE"
)

// OptionContract represents a single option strike.
type OptionContract struct {
	ID         uuid.UUID  `json:"id"`
	Underlying string     `json:"underlying"`
	Strike     float64    `json:"strike"`
	Expiry     time.Time  `json:"expiry"`
	OptionType OptionType `json:"option_type"`
	Symbol     string     `json:"symbol"`
	LTP        float64    `json:"ltp"`
	Bid        float64    `json:"bid"`
	Ask        float64    `json:"ask"`
	Volume     int64      `json:"volume"`
	OI         int64      `json:"oi"`
	OIChange   int64      `json:"oi_change"`
	IV         float64    `json:"iv"`
	Delta      float64    `json:"delta"`
	Gamma      float64    `json:"gamma"`
	Theta      float64    `json:"theta"`
	Vega       float64    `json:"vega"`
	ProviderTS time.Time  `json:"provider_ts"`
}
