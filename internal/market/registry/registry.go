package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/domain/option"
)

// Instrument is a fully typed market instrument — no magic strings downstream.
type Instrument struct {
	Symbol         string                `json:"symbol" yaml:"symbol"`
	Token          string                `json:"token" yaml:"token"`
	Exchange       string                `json:"exchange" yaml:"exchange"`
	InstrumentType market.InstrumentType `json:"instrument_type" yaml:"instrument_type"`
	Underlying     string                `json:"underlying,omitempty" yaml:"underlying,omitempty"`
	Strike         float64               `json:"strike,omitempty" yaml:"strike,omitempty"`
	Expiry         *time.Time            `json:"expiry,omitempty" yaml:"expiry,omitempty"`
	OptionType     option.OptionType     `json:"option_type,omitempty" yaml:"option_type,omitempty"`
	LotSize        int                   `json:"lot_size" yaml:"lot_size"`
	Segment        string                `json:"segment,omitempty" yaml:"segment,omitempty"`
}

// Registry resolves instruments by symbol or broker token.
type Registry struct {
	mu       sync.RWMutex
	bySymbol map[string]Instrument
	byToken  map[string]Instrument
}

// New builds an empty registry.
func New() *Registry {
	return &Registry{
		bySymbol: make(map[string]Instrument),
		byToken:  make(map[string]Instrument),
	}
}

// Load replaces registry contents from a slice of instruments.
func (r *Registry) Load(instruments []Instrument) error {
	bySymbol := make(map[string]Instrument, len(instruments))
	byToken := make(map[string]Instrument, len(instruments))

	for _, inst := range instruments {
		if inst.Symbol == "" {
			return fmt.Errorf("instrument missing symbol")
		}
		if inst.Token == "" {
			return fmt.Errorf("instrument %s missing token", inst.Symbol)
		}
		if _, dup := bySymbol[inst.Symbol]; dup {
			return fmt.Errorf("duplicate symbol: %s", inst.Symbol)
		}
		if _, dup := byToken[inst.Token]; dup {
			return fmt.Errorf("duplicate token: %s", inst.Token)
		}
		bySymbol[inst.Symbol] = inst
		byToken[inst.Token] = inst
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.bySymbol = bySymbol
	r.byToken = byToken
	return nil
}

// Register adds or updates a single instrument.
func (r *Registry) Register(inst Instrument) error {
	if inst.Symbol == "" || inst.Token == "" {
		return fmt.Errorf("instrument missing symbol or token")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.bySymbol[inst.Symbol]; ok {
		delete(r.byToken, old.Token)
	}
	if old, ok := r.byToken[inst.Token]; ok {
		delete(r.bySymbol, old.Symbol)
	}
	r.bySymbol[inst.Symbol] = inst
	r.byToken[inst.Token] = inst
	return nil
}

// BySymbol looks up an instrument by trading symbol.
func (r *Registry) BySymbol(symbol string) (Instrument, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.bySymbol[symbol]
	return inst, ok
}

// ByToken looks up an instrument by broker token.
func (r *Registry) ByToken(token string) (Instrument, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.byToken[token]
	return inst, ok
}

// All returns a snapshot of all registered instruments.
func (r *Registry) All() []Instrument {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Instrument, 0, len(r.bySymbol))
	for _, inst := range r.bySymbol {
		out = append(out, inst)
	}
	return out
}

// Count returns the number of registered instruments.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.bySymbol)
}
