package backtest

import (
	"context"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// ReplayProvider replays historical candles as market data events.
type ReplayProvider struct {
	mu         sync.RWMutex
	connected  bool
	subscribed map[string]struct{}
	events     chan events.Event
	candles    []market.Candle
	clk        *clock.ReplayClock
	speed      float64
	position   int
	metrics    *replayMetrics
	stop       chan struct{}
	wg         sync.WaitGroup
}

// NewReplayProvider creates a replay provider with pre-loaded candles.
func NewReplayProvider(clk *clock.ReplayClock, speed float64, candles []market.Candle, metrics *replayMetrics) *ReplayProvider {
	if speed <= 0 {
		speed = 1.0
	}
	if metrics == nil {
		metrics = newReplayMetrics()
	}
	metrics.total.Store(uint64(len(candles)))
	symbols := uniqueSymbols(candles)
	metrics.symbolsLoaded.Store(uint64(len(symbols)))

	if clk == nil && len(candles) > 0 {
		clk = clock.NewReplay(candleTime(candles[0]))
	}

	return &ReplayProvider{
		subscribed: make(map[string]struct{}),
		events:     make(chan events.Event, 256),
		candles:    candles,
		clk:        clk,
		speed:      speed,
		metrics:    metrics,
		stop:       make(chan struct{}),
	}
}

func uniqueSymbols(candles []market.Candle) []string {
	seen := make(map[string]struct{})
	for _, c := range candles {
		seen[c.Symbol] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	return out
}

func (p *ReplayProvider) Name() string { return providerName }

func (p *ReplayProvider) Capabilities() api.Capabilities {
	return api.Capabilities{
		LiveTicks:      false,
		OptionChain:    false,
		HistoricalData: true,
		Replay:         true,
	}
}

func (p *ReplayProvider) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connected {
		return nil
	}
	p.connected = true
	p.stop = make(chan struct{})
	p.metrics.connected.Store(true)
	p.metrics.setStatus(ReplayStatusRunning)
	p.wg = sync.WaitGroup{}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.replayLoop(p.stop)
	}()
	return nil
}

func (p *ReplayProvider) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	if !p.connected {
		p.mu.Unlock()
		return nil
	}
	close(p.stop)
	p.connected = false
	p.mu.Unlock()
	p.wg.Wait()

	p.metrics.connected.Store(false)
	if p.metrics.statusValue() == ReplayStatusRunning {
		p.metrics.setStatus(ReplayStatusStopped)
	}
	return nil
}

func (p *ReplayProvider) Subscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range symbols {
		p.subscribed[s] = struct{}{}
	}
	return nil
}

func (p *ReplayProvider) Unsubscribe(ctx context.Context, symbols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range symbols {
		delete(p.subscribed, s)
	}
	return nil
}

func (p *ReplayProvider) Events() <-chan events.Event { return p.events }

func (p *ReplayProvider) Health() health.Report {
	p.mu.RLock()
	defer p.mu.RUnlock()
	status := health.StatusHealthy
	if !p.connected {
		status = health.StatusDegraded
	}
	replayStatus := p.metrics.statusValue()
	if replayStatus == ReplayStatusCompleted {
		status = health.StatusHealthy
	}
	return health.Report{
		Component: providerName,
		Status:    status,
		Connected: p.connected,
		Message:   "backtest replay provider",
		Details: map[string]string{
			"status":           string(replayStatus),
			"candles_replayed": u64String(p.metrics.processed.Load()),
			"progress":         progressString(p.position, len(p.candles)),
		},
	}
}

func progressString(position, total int) string {
	if total == 0 {
		return "0%"
	}
	pct := (uint64(position) * 100) / uint64(total)
	return u64String(pct) + "%"
}

func (p *ReplayProvider) replayLoop(stop <-chan struct{}) {
	var prev time.Time
	for {
		p.mu.RLock()
		pos := p.position
		total := len(p.candles)
		speed := p.speed
		subscribed := make(map[string]struct{}, len(p.subscribed))
		for s := range p.subscribed {
			subscribed[s] = struct{}{}
		}
		var candle market.Candle
		if pos < total {
			candle = p.candles[pos]
		}
		p.mu.RUnlock()

		if pos >= total {
			p.metrics.setStatus(ReplayStatusCompleted)
			return
		}

		if len(subscribed) > 0 {
			if _, ok := subscribed[candle.Symbol]; !ok {
				p.mu.Lock()
				p.position++
				p.metrics.position.Store(uint64(p.position))
				p.mu.Unlock()
				continue
			}
		}

		ts := candleTime(candle)
		select {
		case <-stop:
			return
		default:
		}

		if !prev.IsZero() && speed > 0 {
			gap := ts.Sub(prev)
			wall := time.Duration(float64(gap) / speed)
			if wall > 0 {
				timer := time.NewTimer(wall)
				select {
				case <-stop:
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			if p.clk != nil {
				p.clk.Advance(ts.Sub(prev))
			}
		} else if p.clk != nil {
			p.clk.Set(ts)
		}
		prev = ts

		evt, err := EventFromCandleAt(candle, ts)
		if err != nil {
			p.mu.Lock()
			p.position++
			p.metrics.position.Store(uint64(p.position))
			p.mu.Unlock()
			continue
		}

		select {
		case p.events <- evt:
			p.metrics.recordReplay(ts)
		case <-stop:
			return
		}

		p.mu.Lock()
		p.position++
		p.metrics.position.Store(uint64(p.position))
		p.mu.Unlock()
	}
}

// Register adds the backtest replay provider factory to the provider registry.
func Register(reg *api.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a replay provider from factory configuration.
func NewFromConfig(cfg api.FactoryConfig) (api.Provider, error) {
	speed := getFloat(cfg.ProviderCfg, "speed", 1.0)
	startStr := getString(cfg.ProviderCfg, "start_time")
	endStr := getString(cfg.ProviderCfg, "end_time")
	dataPath := getString(cfg.ProviderCfg, "data_path")
	timeframe := market.Timeframe(getString(cfg.ProviderCfg, "timeframe"))
	if timeframe == "" {
		timeframe = market.TF1m
	}

	var start, end time.Time
	if startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, err
		}
		start = t
	}
	if endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			return nil, err
		}
		end = t
	}

	symbols := getStringSlice(cfg.ProviderCfg, "symbols")
	opts := LoadOptions{
		Symbols:   symbols,
		StartTime: start,
		EndTime:   end,
		Timeframe: timeframe,
	}

	var candles []market.Candle
	if dataPath != "" {
		loaded, err := Load(dataPath, opts)
		if err != nil {
			return nil, err
		}
		candles = loaded
	}

	var replayClk *clock.ReplayClock
	if c, ok := cfg.Deps.Clock.(*clock.ReplayClock); ok && c != nil {
		replayClk = c
	} else if len(candles) > 0 {
		replayClk = clock.NewReplay(candleTime(candles[0]))
	} else if !start.IsZero() {
		replayClk = clock.NewReplay(start)
	} else {
		replayClk = clock.NewReplay(time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC))
	}

	return NewReplayProvider(replayClk, speed, candles, newReplayMetrics()), nil
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func getFloat(m map[string]any, key string, def float64) float64 {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		speed, err := ParseSpeed(v)
		if err == nil {
			return speed
		}
	}
	return def
}

func getStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Ensure ReplayProvider implements api.Provider.
var _ api.Provider = (*ReplayProvider)(nil)
