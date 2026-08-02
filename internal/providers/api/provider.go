package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/core/metrics"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
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

// FactoryFunc constructs a Provider from typed configuration.
type FactoryFunc func(cfg FactoryConfig) (Provider, error)

// FactoryConfig carries dependencies injected into provider factories.
type FactoryConfig struct {
	Name         string
	ProviderCfg  map[string]any
	Reconnect    ReconnectConfig
	Subscription SubscriptionConfig
	Heartbeat    HeartbeatConfig
	Deps         Dependencies
}

// Dependencies are shared services passed to every provider factory.
type Dependencies struct {
	Clock          clock.Clock
	SymbolRegistry *symbolregistry.Registry
	Metrics        metrics.Registry
}

// ReconnectConfig controls provider reconnect behaviour.
type ReconnectConfig struct {
	Interval   string `mapstructure:"interval"`
	MaxRetries int    `mapstructure:"max_retries"` // -1 = unlimited
}

// SubscriptionConfig controls batch subscription sizing.
type SubscriptionConfig struct {
	BatchSize int `mapstructure:"batch_size"`
}

// HeartbeatConfig controls heartbeat monitoring interval.
type HeartbeatConfig struct {
	Interval string `mapstructure:"interval"`
}

// Registry maps provider names to factory functions.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]FactoryFunc
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]FactoryFunc)}
}

// Register adds a factory for the given provider name.
func (r *Registry) Register(name string, factory FactoryFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Create instantiates a provider by name.
func (r *Registry) Create(name string, cfg FactoryConfig) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
	cfg.Name = name
	return factory(cfg)
}

// Names returns all registered provider names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// ParseDuration parses a duration string with a sensible default.
func ParseDuration(raw, fallback string) time.Duration {
	if raw == "" {
		raw = fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		d, _ = time.ParseDuration(fallback)
	}
	return d
}

// MaxRetries returns max retries; -1 means unlimited.
func MaxRetries(n int) int {
	if n == 0 {
		return -1
	}
	return n
}

// ValidateCapabilities ensures the provider meets minimum requirements.
func ValidateCapabilities(p Provider, checks ...func(Capabilities) bool) error {
	caps := p.Capabilities()
	for _, check := range checks {
		if !check(caps) {
			return fmt.Errorf("provider %q missing required capability", p.Name())
		}
	}
	return nil
}
