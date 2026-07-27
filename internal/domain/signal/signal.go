package signal

import (
	"time"

	"github.com/google/uuid"
)

// Source identifies which engine produced a signal.
type Source string

const (
	SourceTechnical Source = "TECHNICAL"
	SourceOption    Source = "OPTION"
	SourceContext   Source = "CONTEXT"
	SourceStrategy  Source = "STRATEGY"
)

// Direction indicates bullish, bearish, or neutral bias.
type Direction string

const (
	Bullish Direction = "BULLISH"
	Bearish Direction = "BEARISH"
	Neutral Direction = "NEUTRAL"
)

// Signal is a structured output from any analysis module.
type Signal struct {
	ID          uuid.UUID         `json:"id"`
	Source      Source            `json:"source"`
	Name        string            `json:"name"`
	Symbol      string            `json:"symbol"`
	Direction   Direction         `json:"direction"`
	Confidence  float64           `json:"confidence"` // 0.0 – 1.0
	Explanation string            `json:"explanation"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	GeneratedAt time.Time         `json:"generated_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
}
