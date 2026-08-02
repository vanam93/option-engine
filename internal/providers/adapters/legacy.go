package adapters

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/domain/option"
	"github.com/vanam-gangireddy/option-engine/internal/providers"
)

// LegacyProviderAdapter bridges the new Provider interface to the Stage 1 MarketDataProvider.
type LegacyProviderAdapter struct {
	inner     providers.Provider
	mu        sync.RWMutex
	connected bool
	ticks     chan market.Tick
	chains    chan option.OptionChainSnapshot
	stop      chan struct{}
}

// NewLegacyProviderAdapter wraps a Provider for backward-compatible consumers.
func NewLegacyProviderAdapter(p providers.Provider) *LegacyProviderAdapter {
	return &LegacyProviderAdapter{
		inner:  p,
		ticks:  make(chan market.Tick, 256),
		chains: make(chan option.OptionChainSnapshot, 64),
		stop:   make(chan struct{}),
	}
}

func (a *LegacyProviderAdapter) Name() string { return a.inner.Name() }

func (a *LegacyProviderAdapter) Connect(ctx context.Context) error {
	if err := a.inner.Connect(ctx); err != nil {
		return err
	}
	a.mu.Lock()
	a.connected = true
	a.stop = make(chan struct{})
	a.mu.Unlock()
	go a.fanOut()
	return nil
}

func (a *LegacyProviderAdapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	if a.connected {
		close(a.stop)
		a.connected = false
	}
	a.mu.Unlock()
	return a.inner.Disconnect(ctx)
}

func (a *LegacyProviderAdapter) Subscribe(ctx context.Context, symbols []string) error {
	return a.inner.Subscribe(ctx, symbols)
}

func (a *LegacyProviderAdapter) Unsubscribe(ctx context.Context, symbols []string) error {
	return a.inner.Unsubscribe(ctx, symbols)
}

func (a *LegacyProviderAdapter) Ticks() <-chan market.Tick { return a.ticks }

func (a *LegacyProviderAdapter) OptionChains() <-chan option.OptionChainSnapshot {
	return a.chains
}

func (a *LegacyProviderAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected
}

func (a *LegacyProviderAdapter) fanOut() {
	eventCh := a.inner.Events()
	for {
		select {
		case <-a.stop:
			return
		case evt, ok := <-eventCh:
			if !ok {
				return
			}
			a.dispatch(evt)
		}
	}
}

func (a *LegacyProviderAdapter) dispatch(evt events.Event) {
	switch evt.Type {
	case events.MarketDataReceived:
		var tick market.Tick
		if err := json.Unmarshal(evt.Payload, &tick); err == nil {
			select {
			case a.ticks <- tick:
			default:
			}
		}
	case events.OptionChainUpdated:
		var chain option.OptionChainSnapshot
		if err := json.Unmarshal(evt.Payload, &chain); err == nil {
			select {
			case a.chains <- chain:
			default:
			}
		}
	}
}
