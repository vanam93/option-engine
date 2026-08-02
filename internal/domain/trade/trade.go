package trade

import (
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/decision"
)

// Status tracks the lifecycle of a trade.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusOpen      Status = "OPEN"
	StatusPartial   Status = "PARTIAL"
	StatusClosed    Status = "CLOSED"
	StatusCancelled Status = "CANCELLED"
)

// Side is long or short.
type Side string

const (
	SideLong  Side = "LONG"
	SideShort Side = "SHORT"
)

// Trade represents an active or closed position.
type Trade struct {
	ID               uuid.UUID                `json:"id"`
	Symbol           string                   `json:"symbol"`
	Side             Side                     `json:"side"`
	Status           Status                   `json:"status"`
	Quantity         int                      `json:"quantity"`
	EntryPrice       float64                  `json:"entry_price"`
	CurrentPrice     float64                  `json:"current_price"`
	StopLoss         float64                  `json:"stop_loss"`
	Target           float64                  `json:"target"`
	TrailingStop     *float64                 `json:"trailing_stop,omitempty"`
	RealizedPnL      float64                  `json:"realized_pnl"`
	UnrealizedPnL    float64                  `json:"unrealized_pnl"`
	RecommendationID uuid.UUID                `json:"recommendation_id"`
	Recommendation   *decision.Recommendation `json:"recommendation,omitempty"`
	OpenedAt         time.Time                `json:"opened_at"`
	ClosedAt         *time.Time               `json:"closed_at,omitempty"`
	UpdatedAt        time.Time                `json:"updated_at"`
}
