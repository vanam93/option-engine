package mock

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

const providerName = "mock"

// Provider emits synthetic ticks for development and testing.
type Provider struct {
	mu             sync.RWMutex
	connected      bool
	subscribed     map[string]struct{}
	events         chan events.Event
	reconnectCount int64
	lastEvent      *time.Time
	clk            clock.Clock
	tickInterval   time.Duration
	rng            *rand.Rand
	stop           chan struct{}
	wg             sync.WaitGroup
}

// Register adds the mock provider factory to the registry.
func Register(reg *api.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a mock provider from factory configuration.
func NewFromConfig(cfg api.FactoryConfig) (api.Provider, error) {
	interval := api.ParseDuration(getString(cfg.ProviderCfg, "tick_interval"), "1s")
	clk := clock.NewSystem()
	if c, ok := cfg.Deps.Clock.(clock.Clock); ok && c != nil {
		clk = c
	}
	seed := uint64(getInt(cfg.ProviderCfg, "seed", 1))
	return NewSeeded(clk, interval, seed), nil
}

// New creates a mock provider with the given clock and tick interval.
func New(clk clock.Clock, tickInterval time.Duration) *Provider {
	return NewSeeded(clk, tickInterval, 1)
}

// NewSeeded creates a repeatable synthetic NSE feed. Equal seed and subscription
// order produce equal price sequences.
func NewSeeded(clk clock.Clock, tickInterval time.Duration, seed uint64) *Provider {
	if tickInterval <= 0 {
		tickInterval = time.Second
	}
	return &Provider{
		subscribed:   make(map[string]struct{}),
		events:       make(chan events.Event, 256),
		clk:          clk,
		tickInterval: tickInterval,
		rng:          rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		stop:         make(chan struct{}),
	}
}

func (p *Provider) Name() string { return providerName }

func (p *Provider) Capabilities() api.Capabilities {
	return api.Capabilities{
		LiveTicks:      true,
		OptionChain:    true,
		HistoricalData: false,
		Replay:         false,
	}
}

func (p *Provider) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connected {
		return nil
	}
	p.connected = true
	p.stop = make(chan struct{})
	p.wg = sync.WaitGroup{}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.emitLoop(p.stop)
	}()
	return nil
}

func (p *Provider) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	if !p.connected {
		p.mu.Unlock()
		return nil
	}
	close(p.stop)
	p.connected = false
	p.mu.Unlock()
	p.wg.Wait()
	return nil
}

func (p *Provider) Subscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range symbols {
		p.subscribed[s] = struct{}{}
	}
	return nil
}

func (p *Provider) Unsubscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range symbols {
		delete(p.subscribed, s)
	}
	return nil
}

func (p *Provider) Events() <-chan events.Event { return p.events }

func (p *Provider) SetRawPayloadMode(enabled bool) {
	_ = enabled
}

func (p *Provider) Health() health.Report {
	p.mu.RLock()
	defer p.mu.RUnlock()
	status := health.StatusHealthy
	if !p.connected {
		status = health.StatusUnhealthy
	}
	return health.Report{
		Component:      providerName,
		Status:         status,
		Connected:      p.connected,
		ReconnectCount: atomic.LoadInt64(&p.reconnectCount),
		LastEventTime:  p.lastEvent,
		Message:        "mock provider",
	}
}

func (p *Provider) emitLoop(stop <-chan struct{}) {
	price := 22500.0
	for {
		select {
		case <-stop:
			return
		default:
		}
		p.mu.RLock()
		tickInterval := p.tickInterval
		p.mu.RUnlock()
		timer := time.NewTimer(tickInterval)
		select {
		case <-stop:
			timer.Stop()
			return
		case <-timer.C:
		}
		p.mu.RLock()
		symbols := make([]string, 0, len(p.subscribed))
		for s := range p.subscribed {
			symbols = append(symbols, s)
		}
		p.mu.RUnlock()
		sort.Strings(symbols)

		if len(symbols) == 0 {
			continue
		}

		now := p.clk.Now()
		for _, symbol := range symbols {
			p.mu.Lock()
			price += (p.rng.Float64() - 0.48) * 8
			p.mu.Unlock()
			if price < 1 {
				price = 1
			}
			payload := normalizer.Payload{
				Symbol:         symbol,
				Exchange:       "NSE",
				InstrumentType: market.InstrumentIndex,
				LTP:            price,
				Open:           price - 2, High: price + 3, Low: price - 3, Close: price - 1,
				Bid: price - .25, Ask: price + .25, BidQty: 50, AskQty: 60, Volume: 1000, OI: 500,
				Timestamp: now,
			}
			var evt events.Event
			var err error
			evt, err = events.NewEventWithClock(p.clk, events.MarketDataReceived, providerName, payload)
			if err != nil {
				continue
			}
			slog.Debug("provider emitted event",
				"provider", providerName,
				"symbol", symbol,
				"timestamp", now,
			)
			select {
			case p.events <- evt:
				p.mu.Lock()
				p.lastEvent = &now
				p.mu.Unlock()
			default:
			}
		}
	}
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

func getInt(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return def
}
