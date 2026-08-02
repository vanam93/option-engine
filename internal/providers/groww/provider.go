package groww

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

// Provider streams Groww historical candles through the market event pipeline.
type Provider struct {
	mu             sync.RWMutex
	cfg            Config
	client         *Client
	auth           *Authenticator
	hist           *HistoricalService
	coordinator    *streamCoordinator
	registry       *symbolregistry.Registry
	clk            clock.Clock
	replayClk      *clock.ReplayClock
	connected      bool
	subscribed     map[string]symbolregistry.Instrument
	events         chan events.Event
	metrics        *healthMetrics
	reconnectCount int64
	lastEvent      *time.Time
	stop           chan struct{}
	wg             sync.WaitGroup
}

// Register adds the Groww provider factory to the registry.
func Register(reg *api.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a Groww provider from factory configuration.
func NewFromConfig(cfg api.FactoryConfig) (api.Provider, error) {
	parsed, err := ParseConfig(cfg)
	if err != nil {
		return nil, err
	}
	if !parsed.Enabled {
		return nil, ErrNotConfigured
	}

	metrics := newHealthMetrics()
	client := newClient(parsed, metrics)
	auth := newAuthenticator(parsed, client)
	hist := newHistoricalService(parsed, client, auth)

	var clk clock.Clock = clock.NewSystem()
	if c, ok := cfg.Deps.Clock.(clock.Clock); ok && c != nil {
		clk = c
	}
	var replayClk *clock.ReplayClock
	if rc, ok := clk.(*clock.ReplayClock); ok {
		replayClk = rc
	} else if !parsed.StartTime.IsZero() {
		replayClk = clock.NewReplay(parsed.StartTime)
		clk = replayClk
	}

	return &Provider{
		cfg:         parsed,
		client:      client,
		auth:        auth,
		hist:        hist,
		coordinator: newStreamCoordinator(hist, parsed),
		registry:    cfg.Deps.SymbolRegistry,
		clk:         clk,
		replayClk:   replayClk,
		subscribed:  make(map[string]symbolregistry.Instrument),
		events:      make(chan events.Event, 1024),
		metrics:     metrics,
		stop:        make(chan struct{}),
	}, nil
}

func (p *Provider) Name() string { return providerName }

func (p *Provider) Capabilities() api.Capabilities {
	return api.Capabilities{
		LiveTicks:      false,
		OptionChain:    true,
		HistoricalData: true,
		Replay:         true,
	}
}

func (p *Provider) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connected {
		return nil
	}

	token, err := p.auth.Authenticate(ctx)
	if err != nil {
		return err
	}
	_ = token
	p.metrics.setAuthenticated(true)
	p.connected = true
	p.metrics.setConnected(true)
	p.stop = make(chan struct{})
	p.wg = sync.WaitGroup{}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.streamLoop(p.stop)
	}()
	slog.Info("groww provider connected")
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
	p.metrics.setConnected(false)
	p.mu.Unlock()
	p.wg.Wait()
	p.client.Close()
	slog.Info("groww provider disconnected")
	return nil
}

func (p *Provider) Subscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, symbol := range symbols {
		inst := p.resolveInstrument(symbol)
		p.subscribed[symbol] = inst
	}
	return nil
}

func (p *Provider) Unsubscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, symbol := range symbols {
		delete(p.subscribed, symbol)
	}
	return nil
}

func (p *Provider) Events() <-chan events.Event { return p.events }

func (p *Provider) Health() health.Report {
	p.mu.RLock()
	connected := p.connected
	last := p.lastEvent
	p.mu.RUnlock()
	report := buildHealthReport(connected, p.metrics, "groww historical provider")
	report.LastEventTime = last
	report.ReconnectCount = atomic.LoadInt64(&p.reconnectCount)
	return report
}

func (p *Provider) resolveInstrument(symbol string) symbolregistry.Instrument {
	if p.registry != nil {
		if inst, ok := p.registry.BySymbol(symbol); ok {
			return inst
		}
	}
	return symbolregistry.Instrument{
		Symbol:         symbol,
		Exchange:       p.cfg.Exchange,
		InstrumentType: market.InstrumentIndex,
		Segment:        p.cfg.Segment,
	}
}

func (p *Provider) streamLoop(stop <-chan struct{}) {
	ctx := context.Background()
	var prev time.Time

	for {
		p.mu.RLock()
		if len(p.subscribed) == 0 {
			p.mu.RUnlock()
			select {
			case <-stop:
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
		symbols := make(map[string]symbolregistry.Instrument, len(p.subscribed))
		for s, inst := range p.subscribed {
			symbols[s] = inst
		}
		p.mu.RUnlock()

		p.coordinator.Reset(symbols)
		if err := p.coordinator.Prime(ctx); err != nil {
			slog.Error("groww stream prime failed", "error", err)
			atomic.AddInt64(&p.reconnectCount, 1)
			select {
			case <-stop:
				return
			case <-time.After(p.cfg.RetryBackoff):
				continue
			}
		}

		for {
			select {
			case <-stop:
				return
			default:
			}

			candle, inst, ok, err := p.coordinator.Next(ctx)
			if err != nil {
				slog.Error("groww stream failed", "error", err)
				atomic.AddInt64(&p.reconnectCount, 1)
				break
			}
			if !ok {
				slog.Info("groww historical stream completed")
				return
			}

			ts := candleTime(candle)
			delay := replayDelay(prev, ts, p.cfg.ReplaySpeed, p.cfg.InstantReplay)
			if delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-stop:
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			if p.replayClk != nil && !prev.IsZero() {
				p.replayClk.Advance(ts.Sub(prev))
			} else if p.replayClk != nil && prev.IsZero() {
				p.replayClk.Set(ts)
			}
			prev = ts

			evt, err := EventFromCandleAt(candle, inst, ts)
			if err != nil {
				continue
			}
			select {
			case p.events <- evt:
				p.metrics.recordCandle()
				p.mu.Lock()
				p.lastEvent = &ts
				p.mu.Unlock()
			case <-stop:
				return
			}
		}
	}
}

var _ api.Provider = (*Provider)(nil)
