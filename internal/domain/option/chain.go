package option

import (
	"time"

	"github.com/google/uuid"
)

// OptionChainSnapshot captures the full chain at a moment in time.
type OptionChainSnapshot struct {
	ID           uuid.UUID        `json:"id"`
	Underlying   string           `json:"underlying"`
	SpotPrice    float64          `json:"spot_price"`
	Expiry       time.Time        `json:"expiry"`
	Contracts    []OptionContract `json:"contracts"`
	TotalCallOI  int64            `json:"total_call_oi"`
	TotalPutOI   int64            `json:"total_put_oi"`
	PCR          float64          `json:"pcr"`
	MaxPain      float64          `json:"max_pain"`
	ATMStrike    float64          `json:"atm_strike"`
	SnapshotTime time.Time        `json:"snapshot_time"`
}
