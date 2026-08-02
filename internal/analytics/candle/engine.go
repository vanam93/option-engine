package candle

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine aggregates canonical ticks into OHLCV candles and publishes CandleClosed events.
type Engine struct {
	cfg        Config
	bus        ports.EventBus
	clk        clock.Clock
	aggregator *Aggregator

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup

	processed atomic.Uint64
	published atomic.Uint64
	rejected  atomic.Uint64
	evicted   atomic.Uint64
	lastEvent atomic.Value // time.Time
}

// New creates a candle engine. The bus must be the Stage 2 runtime event bus.
func New(cfg Config, bus ports.EventBus, clk clock.Clock) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	loc, err := cfg.Location()
	if err != nil {
		return nil, err
	}
	return &Engine{
		cfg: cfg,
		bus: bus,
		clk: clk,
		aggregator: NewAggregator(loc, AggregatorOptions{
			VolumeMode:  cfg.volumeMode(),
			OrderPolicy: cfg.orderPolicy(),
			IdleEvict:   cfg.IdleEvictAfter,
		}),
	}, nil
}

// Start subscribes to canonical tick events and begins aggregation.
// Subscription is established before the consumer goroutine starts so no
// published tick can be missed once Start returns.
func (e *Engine) Start(ctx context.Context) error {
	if !e.cfg.Enabled {
		return nil
	}

	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	engineCtx, cancel := context.WithCancel(ctx)
	e.ctx = engineCtx
	e.cancel = cancel
	e.subscription = e.bus.Subscribe(e.cfg.SubscriberBuffer, func(evt events.Event) bool {
		return evt.Type == events.MarketDataReceived
	})
	e.started = true
	e.mu.Unlock()

	e.wg.Add(1)
	go e.run(engineCtx)
	return nil
}

func (e *Engine) run(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handle(evt)
		}
	}
}

func (e *Engine) handle(evt events.Event) {
	var tick market.Tick
	if err := json.Unmarshal(evt.Payload, &tick); err != nil {
		return
	}
	e.processed.Add(1)
	e.lastEvent.Store(tick.ProviderTS)

	closed, stats, err := e.aggregator.Update(tick, e.cfg.Timeframes)
	if err != nil {
		return
	}
	e.rejected.Add(uint64(stats.Rejected))
	e.evicted.Add(uint64(stats.Evicted))
	e.publishClosed(closed)
}

func (e *Engine) publishClosed(candles []market.Candle) {
	for _, c := range candles {
		out, err := events.NewEventWithClock(e.clk, events.CandleClosed, engineName, c)
		if err != nil {
			continue
		}
		e.bus.Publish(out)
		e.published.Add(1)
	}
}

// Close stops the engine, optionally flushes in-progress candles, and releases
// its subscription. Flush runs after the consumer goroutine has exited.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	cancel := e.cancel
	sub := e.subscription
	flush := e.cfg.FlushOnShutdown
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()

	if flush && e.aggregator != nil {
		e.publishClosed(e.aggregator.Flush())
	}
	if sub != nil {
		sub.Close()
	}
	return nil
}

// Health reports runtime status for observability probes.
func (e *Engine) Health() health.Report {
	status := health.StatusHealthy
	connected := e.started && !e.closed
	if e.cfg.Enabled && !connected {
		status = health.StatusDegraded
	}
	if !e.cfg.Enabled {
		status = health.StatusHealthy
	}

	var last *time.Time
	if v := e.lastEvent.Load(); v != nil {
		if t, ok := v.(time.Time); ok && !t.IsZero() {
			last = &t
		}
	}

	dropped := uint64(0)
	e.mu.Lock()
	if e.subscription != nil {
		dropped = e.subscription.Dropped()
	}
	activeBuilders := 0
	if e.aggregator != nil {
		activeBuilders = e.aggregator.ActiveBuilders()
	}
	e.mu.Unlock()

	if dropped > 0 {
		status = health.StatusDegraded
	}

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     connected,
		LastEventTime: last,
		Message:       "candle aggregation engine",
		Details: map[string]string{
			"enabled":         boolString(e.cfg.Enabled),
			"processed":       u64String(e.processed.Load()),
			"published":       u64String(e.published.Load()),
			"rejected":        u64String(e.rejected.Load()),
			"evicted":         u64String(e.evicted.Load()),
			"dropped":         u64String(dropped),
			"active_builders": u64String(uint64(activeBuilders)),
			"timeframes":      timeframeString(e.cfg.Timeframes),
			"volume_mode":     string(e.cfg.volumeMode()),
		},
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func u64String(v uint64) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func timeframeString(tfs []market.Timeframe) string {
	if len(tfs) == 0 {
		return ""
	}
	out := string(tfs[0])
	for i := 1; i < len(tfs); i++ {
		out += "," + string(tfs[i])
	}
	return out
}
