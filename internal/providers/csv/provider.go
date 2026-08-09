package csv

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

// Provider streams CSV historical candles through the market event pipeline.
type Provider struct {
	mu         sync.RWMutex
	cfg        Config
	registry   *symbolregistry.Registry
	clk        clock.Clock
	replayClk  *clock.ReplayClock
	coordinator *streamCoordinator
	streamer   *Streamer
	connected  bool
	subscribed map[string]symbolregistry.Instrument
	events     chan events.Event
	metrics    *healthMetrics
	lastEvent  *time.Time
	stop       chan struct{}
	wg         sync.WaitGroup
}

// Register adds the CSV provider factory to the registry.
func Register(reg *api.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a CSV provider from factory configuration.
func NewFromConfig(cfg api.FactoryConfig) (api.Provider, error) {
	parsed, err := ParseConfig(cfg)
	if err != nil {
		return nil, err
	}
	if !parsed.Enabled {
		return nil, ErrNotConfigured
	}

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
		registry:    cfg.Deps.SymbolRegistry,
		clk:         clk,
		replayClk:   replayClk,
		coordinator: newStreamCoordinator(parsed),
		streamer:    NewStreamer(parsed),
		subscribed:  make(map[string]symbolregistry.Instrument),
		events:      make(chan events.Event, 1024),
		metrics:     newHealthMetrics(),
		stop:        make(chan struct{}),
	}, nil
}

func (p *Provider) Name() string { return providerName }

func (p *Provider) Capabilities() api.Capabilities {
	return api.Capabilities{
		LiveTicks:      false,
		OptionChain:    false,
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
	p.connected = true
	p.metrics.setConnected(true)
	p.stop = make(chan struct{})
	p.wg = sync.WaitGroup{}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.streamLoop(p.stop)
	}()
	slog.Info("csv provider connected", "file", p.cfg.DataFilePath())
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
	p.coordinator.Close()
	slog.Info("csv provider disconnected")
	return nil
}

func (p *Provider) Subscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, symbol := range symbols {
		p.subscribed[symbol] = p.resolveInstrument(symbol)
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
	report := buildHealthReport(connected, p.metrics, "csv historical provider")
	report.LastEventTime = last
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
		loop := p.cfg.Loop
		p.mu.RUnlock()

		if err := p.coordinator.Reset(symbols); err != nil {
			slog.Error("csv stream reset failed", "error", err)
			select {
			case <-stop:
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}
		if !p.coordinator.HasStreams() {
			slog.Warn("csv provider has no matching symbol subscriptions", "data_symbol", p.cfg.Symbol)
			select {
			case <-stop:
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		p.metrics.setCurrentFile(p.coordinator.CurrentFile())

		for {
			select {
			case <-stop:
				return
			default:
			}

			candle, inst, iter, ok, err := p.coordinator.Next()
			if err != nil {
				slog.Error("csv stream failed", "error", err)
				break
			}
			if !ok {
				if loop {
					break
				}
				slog.Info("csv historical stream completed")
				return
			}

			p.metrics.recordRowRead()
			if iter != nil {
				p.metrics.setCurrentOffset(iter.Offset())
				p.metrics.syncParseErrors(iter.ParseErrors())
			}

			ts := candleTime(candle)
			delay := p.streamer.Delay(prev, ts)
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

			start := time.Now()
			evt, err := EventFromCandleAt(candle, inst, ts)
			if err != nil {
				p.metrics.recordPublishError()
				continue
			}
			select {
			case p.events <- evt:
				p.metrics.recordCandlePublished(time.Since(start))
				p.mu.Lock()
				p.lastEvent = &ts
				p.mu.Unlock()
			case <-stop:
				return
			default:
				p.metrics.recordPublishError()
			}
		}

		if !loop {
			return
		}
		prev = time.Time{}
	}
}

var _ api.Provider = (*Provider)(nil)
