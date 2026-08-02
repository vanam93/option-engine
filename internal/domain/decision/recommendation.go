package decision

import (
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/signal"
)

// Action is the recommended trade action.
type Action string

const (
	ActionBuy     Action = "BUY"
	ActionSell    Action = "SELL"
	ActionHold    Action = "HOLD"
	ActionNoTrade Action = "NO_TRADE"
	ActionExit    Action = "EXIT"
)

// Recommendation is the aggregated output of the Decision Engine.
type Recommendation struct {
	ID           uuid.UUID        `json:"id"`
	Symbol       string           `json:"symbol"`
	Action       Action           `json:"action"`
	Direction    signal.Direction `json:"direction"`
	Confidence   float64          `json:"confidence"`
	EntryPrice   *float64         `json:"entry_price,omitempty"`
	StopLoss     *float64         `json:"stop_loss,omitempty"`
	Target       *float64         `json:"target,omitempty"`
	Rationale    string           `json:"rationale"`
	Contributors []signal.Signal  `json:"contributors"`
	AuditTrail   []AuditEntry     `json:"audit_trail"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

// AuditEntry records a step in the decision reasoning chain.
type AuditEntry struct {
	Step      string    `json:"step"`
	Detail    string    `json:"detail"`
	Score     float64   `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}
