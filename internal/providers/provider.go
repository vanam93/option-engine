package providers

import (
	"context"

	"github.com/option-engine/option-engine/internal/core/health"
	"github.com/option-engine/option-engine/internal/domain/events"
)

// Capabilities describes what a provider can supply.
type Capabilities struct {
	LiveTicks      bool `json:"live_ticks"`
	OptionChain    bool `json:"option_chain"`
	HistoricalData bool `json:"historical_data"`
	Replay         bool `json:"replay"`
	OrderPlacement bool `json:"order_placement"`
}

// HasAll returns true if the provider supports every listed capability.
func (c Capabilities) HasAll(required ...func(Capabilities) bool) bool {
	for _, check := range required {
		if !check(c) {
			return false
		}
	}
	return true
}

// RequiresLiveTicks checks live tick support.
func RequiresLiveTicks(c Capabilities) bool { return c.LiveTicks }

// RequiresOptionChain checks option chain support.
func RequiresOptionChain(c Capabilities) bool { return c.OptionChain }

// RequiresReplay checks replay support.
func RequiresReplay(c Capabilities) bool { return c.Replay }

// Provider is the plugin contract for all market data sources.
// Analysis modules must depend on this interface, never on a specific broker.
type Provider interface {
	Name() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string) error
	Unsubscribe(ctx context.Context, symbols []string) error
	Events() <-chan events.Event
	Health() health.Report
	Capabilities() Capabilities
}
