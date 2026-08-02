// Package validator rejects malformed, stale, duplicate, and unordered ticks.
package validator

import (
	"fmt"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

type ErrorCode string

const (
	Duplicate       ErrorCode = "duplicate"
	OldTimestamp    ErrorCode = "old_timestamp"
	NegativePrice   ErrorCode = "negative_price"
	InvalidQuantity ErrorCode = "invalid_quantity"
	OutOfOrder      ErrorCode = "out_of_order"
	InvalidSymbol   ErrorCode = "invalid_symbol"
)

type Error struct {
	Code   ErrorCode
	Symbol string
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("tick validation %s for %s: %s", e.Code, e.Symbol, e.Detail)
}

type Config struct {
	MaxAge                  time.Duration
	RequireRegisteredSymbol bool
}
type Validator struct {
	mu      sync.Mutex
	cfg     Config
	symbols *symbolregistry.Registry
	last    map[string]market.Tick
}

func New(cfg Config, symbols *symbolregistry.Registry) *Validator {
	return &Validator{cfg: cfg, symbols: symbols, last: make(map[string]market.Tick)}
}
func (v *Validator) Validate(t market.Tick, now time.Time) error {
	if t.Symbol == "" {
		return &Error{InvalidSymbol, t.Symbol, "empty symbol"}
	}
	if v.cfg.RequireRegisteredSymbol && (v.symbols == nil || func() bool { _, ok := v.symbols.BySymbol(t.Symbol); return ok }() == false) {
		return &Error{InvalidSymbol, t.Symbol, "not registered"}
	}
	if t.LTP < 0 || t.Open < 0 || t.High < 0 || t.Low < 0 || t.Close < 0 || t.Bid < 0 || t.Ask < 0 {
		return &Error{NegativePrice, t.Symbol, "negative price"}
	}
	if t.Volume < 0 || t.BidQty < 0 || t.AskQty < 0 || t.OI < 0 {
		return &Error{InvalidQuantity, t.Symbol, "negative quantity"}
	}
	if t.ProviderTS.IsZero() || (v.cfg.MaxAge > 0 && now.Sub(t.ProviderTS) > v.cfg.MaxAge) {
		return &Error{OldTimestamp, t.Symbol, "timestamp is stale"}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	prev, ok := v.last[t.Symbol]
	if ok {
		if t.ProviderTS.Before(prev.ProviderTS) || (t.SequenceNum > 0 && prev.SequenceNum > 0 && t.SequenceNum <= prev.SequenceNum) {
			return &Error{OutOfOrder, t.Symbol, "timestamp or sequence regressed"}
		}
		if t.ProviderTS.Equal(prev.ProviderTS) && t.LTP == prev.LTP && t.Volume == prev.Volume && t.OI == prev.OI {
			return &Error{Duplicate, t.Symbol, "unchanged tick"}
		}
	}
	v.last[t.Symbol] = t
	return nil
}
