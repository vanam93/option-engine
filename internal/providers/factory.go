package providers

import (
	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/providers/groww"
	"github.com/vanam-gangireddy/option-engine/internal/providers/mock"
	"github.com/vanam-gangireddy/option-engine/internal/providers/replay"
)

// DefaultRegistry returns a registry with built-in providers.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	mock.Register(r)
	replay.Register(r)
	backtest.Register(r)
	groww.Register(r)
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
