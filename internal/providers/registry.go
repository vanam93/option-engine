package providers

import (
	"fmt"
	"sync"

	"github.com/option-engine/option-engine/internal/core/clock"
	"github.com/option-engine/option-engine/internal/core/metrics"
	symbolregistry "github.com/option-engine/option-engine/internal/market/registry"
)

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
