package providers

import (
	"fmt"
	"time"

	"github.com/option-engine/option-engine/internal/providers/mock"
	"github.com/option-engine/option-engine/internal/providers/replay"
)

// DefaultRegistry returns a registry with built-in mock and replay providers.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	mock.Register(r)
	replay.Register(r)
	return r
}

// CreateFromConfig builds the configured active provider.
func CreateFromConfig(reg *Registry, marketProvider string, reconnect ReconnectConfig, subscription SubscriptionConfig, heartbeat HeartbeatConfig, providerCfg map[string]any) (Provider, error) {
	if marketProvider == "" {
		marketProvider = "mock"
	}
	return reg.Create(marketProvider, FactoryConfig{
		ProviderCfg:  providerCfg,
		Reconnect:    reconnect,
		Subscription: subscription,
		Heartbeat:    heartbeat,
	})
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
