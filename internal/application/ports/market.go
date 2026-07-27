package ports

import (
	"context"

	"github.com/option-engine/option-engine/internal/core/health"
	"github.com/option-engine/option-engine/internal/domain/events"
	"github.com/option-engine/option-engine/internal/domain/market"
	"github.com/option-engine/option-engine/internal/domain/option"
	"github.com/option-engine/option-engine/internal/providers"
)

// Provider is the canonical market data source interface.
// Re-exported from providers for application-layer dependency injection.
type Provider = providers.Provider

// MarketDataProvider is the Stage 1 interface retained for backward compatibility.
// New code should depend on Provider instead.
type MarketDataProvider interface {
	Name() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string) error
	Unsubscribe(ctx context.Context, symbols []string) error
	Ticks() <-chan market.Tick
	OptionChains() <-chan option.OptionChainSnapshot
	IsConnected() bool
}

// MarketEventBus broadcasts normalized events to subscribers.
type MarketEventBus interface {
	Publish(ctx context.Context, event events.Event) error
	Subscribe(ctx context.Context, eventTypes ...events.Type) (<-chan events.Event, error)
	Close() error
}

// MarketDataStore persists and retrieves market data.
type MarketDataStore interface {
	SaveTick(ctx context.Context, tick market.Tick) error
	SaveCandle(ctx context.Context, candle market.Candle) error
	SaveOptionChain(ctx context.Context, chain option.OptionChainSnapshot) error
	GetCandles(ctx context.Context, symbol string, tf market.Timeframe, from, to interface{}) ([]market.Candle, error)
	Ping(ctx context.Context) error
}

// HealthChecker reports component health for readiness probes.
type HealthChecker interface {
	Check(ctx context.Context) error
}

// HealthReporter exposes detailed component health.
type HealthReporter interface {
	Health() health.Report
}
