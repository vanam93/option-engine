package mock

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/option-engine/option-engine/internal/core/clock"
	"github.com/option-engine/option-engine/internal/core/health"
	"github.com/option-engine/option-engine/internal/domain/events"
	"github.com/option-engine/option-engine/internal/domain/market"
	"github.com/option-engine/option-engine/internal/providers"
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
	stop           chan struct{}
}

// Register adds the mock provider factory to the registry.
func Register(reg *providers.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a mock provider from factory configuration.
func NewFromConfig(cfg providers.FactoryConfig) (providers.Provider, error) {
	interval := providers.ParseDuration(getString(cfg.ProviderCfg, "tick_interval"), "1s")
	clk := clock.NewSystem()
	if c, ok := cfg.Deps.Clock.(clock.Clock); ok && c != nil {
		clk = c
	}
	return New(clk, interval), nil
}

// New creates a mock provider with the given clock and tick interval.
func New(clk clock.Clock, tickInterval time.Duration) *Provider {
	if tickInterval <= 0 {
		tickInterval = time.Second
	}
	return &Provider{
		subscribed:   make(map[string]struct{}),
		events:       make(chan events.Event, 256),
		clk:          clk,
		tickInterval: tickInterval,
		stop:         make(chan struct{}),
	}
}

func (p *Provider) Name() string { return providerName }

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		LiveTicks:      true,
		OptionChain:    false,
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
	go p.emitLoop()
	return nil
}

func (p *Provider) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.connected {
		return nil
	}
	close(p.stop)
	p.connected = false
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

func (p *Provider) emitLoop() {
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

	price := 22500.0
	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.mu.RLock()
			symbols := make([]string, 0, len(p.subscribed))
			for s := range p.subscribed {
				symbols = append(symbols, s)
			}
			p.mu.RUnlock()

			if len(symbols) == 0 {
				continue
			}

			now := p.clk.Now()
			for _, symbol := range symbols {
				price += 0.5
				tick := market.Tick{
					ID:             uuid.New(),
					Symbol:         symbol,
					Exchange:       "NSE",
					InstrumentType: market.InstrumentIndex,
					LTP:            price,
					ProviderTS:     now,
					ReceivedAt:     now,
				}
				evt, err := events.NewEventWithClock(p.clk, events.MarketDataReceived, providerName, tick)
				if err != nil {
					continue
				}
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
