package replay

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/option-engine/option-engine/internal/core/clock"
	"github.com/option-engine/option-engine/internal/core/health"
	"github.com/option-engine/option-engine/internal/domain/events"
	"github.com/option-engine/option-engine/internal/providers"
)

const providerName = "replay"

// Provider replays pre-recorded events for backtesting and debugging.
type Provider struct {
	mu             sync.RWMutex
	connected      bool
	subscribed     map[string]struct{}
	events         chan events.Event
	recorded       []events.Event
	reconnectCount int64
	lastEvent      *time.Time
	clk            *clock.ReplayClock
	speed          float64
	stop           chan struct{}
}

// Register adds the replay provider factory to the registry.
func Register(reg *providers.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a replay provider from factory configuration.
func NewFromConfig(cfg providers.FactoryConfig) (providers.Provider, error) {
	startStr := getString(cfg.ProviderCfg, "start_time")
	speed := getFloat(cfg.ProviderCfg, "speed", 1.0)

	var start time.Time
	if startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, err
		}
		start = t
	} else {
		start = time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	}

	replayClk := clock.NewReplay(start)
	return New(replayClk, speed, nil), nil
}

// New creates a replay provider with optional pre-loaded events.
func New(clk *clock.ReplayClock, speed float64, recorded []events.Event) *Provider {
	if speed <= 0 {
		speed = 1.0
	}
	return &Provider{
		subscribed: make(map[string]struct{}),
		events:     make(chan events.Event, 1024),
		recorded:   recorded,
		clk:        clk,
		speed:      speed,
		stop:       make(chan struct{}),
	}
}

// LoadEvents replaces the recorded event set.
func (p *Provider) LoadEvents(evts []events.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recorded = evts
}

func (p *Provider) Name() string { return providerName }

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
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
	p.connected = true
	p.stop = make(chan struct{})
	go p.replayLoop()
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
		status = health.StatusDegraded
	}
	return health.Report{
		Component:      providerName,
		Status:         status,
		Connected:      p.connected,
		ReconnectCount: atomic.LoadInt64(&p.reconnectCount),
		LastEventTime:  p.lastEvent,
		Message:        "replay provider",
		Details: map[string]string{
			"recorded_events": itoa(len(p.recorded)),
			"speed":           ftoa(p.speed),
		},
	}
}

func (p *Provider) replayLoop() {
	p.mu.RLock()
	eventsCopy := append([]events.Event(nil), p.recorded...)
	p.mu.RUnlock()

	var prev time.Time
	for i, evt := range eventsCopy {
		select {
		case <-p.stop:
			return
		default:
		}

		if i > 0 && !prev.IsZero() {
			gap := evt.Timestamp.Sub(prev)
			if p.speed > 0 {
				gap = time.Duration(float64(gap) / p.speed)
			}
			if gap > 0 {
				time.Sleep(gap)
			}
			p.clk.Advance(evt.Timestamp.Sub(prev))
		}
		prev = evt.Timestamp

		select {
		case p.events <- evt:
			now := p.clk.Now()
			p.mu.Lock()
			p.lastEvent = &now
			p.mu.Unlock()
		case <-p.stop:
			return
		}
	}
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
	default:
		return def
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	// small helper without strconv import for minimal deps in health details
	buf := make([]byte, 0, 12)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func ftoa(f float64) string {
	// simple formatting for health details
	if f == 1.0 {
		return "1"
	}
	return itoa(int(f*10)) + "x" // rough; good enough for details map
}
