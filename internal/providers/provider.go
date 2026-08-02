package providers

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// Capabilities describes what a provider can supply.
type Capabilities = api.Capabilities

// Provider is the plugin contract for all market data sources.
type Provider = api.Provider

// FactoryFunc constructs a Provider from typed configuration.
type FactoryFunc = api.FactoryFunc

// FactoryConfig carries dependencies injected into provider factories.
type FactoryConfig = api.FactoryConfig

// Dependencies are shared services passed to every provider factory.
type Dependencies = api.Dependencies

// ReconnectConfig controls provider reconnect behaviour.
type ReconnectConfig = api.ReconnectConfig

// SubscriptionConfig controls batch subscription sizing.
type SubscriptionConfig = api.SubscriptionConfig

// HeartbeatConfig controls heartbeat monitoring interval.
type HeartbeatConfig = api.HeartbeatConfig

// Registry maps provider names to factory functions.
type Registry = api.Registry

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return api.NewRegistry()
}

// ParseDuration parses a duration string with a sensible default.
func ParseDuration(raw, fallback string) time.Duration {
	return api.ParseDuration(raw, fallback)
}

// MaxRetries returns max retries; -1 means unlimited.
func MaxRetries(n int) int {
	return api.MaxRetries(n)
}

// ValidateCapabilities ensures the provider meets minimum requirements.
func ValidateCapabilities(p Provider, checks ...func(Capabilities) bool) error {
	return api.ValidateCapabilities(p, checks...)
}

// RequiresLiveTicks checks live tick support.
func RequiresLiveTicks(c Capabilities) bool { return api.RequiresLiveTicks(c) }

// RequiresOptionChain checks option chain support.
func RequiresOptionChain(c Capabilities) bool { return api.RequiresOptionChain(c) }

// RequiresReplay checks replay support.
func RequiresReplay(c Capabilities) bool { return api.RequiresReplay(c) }
