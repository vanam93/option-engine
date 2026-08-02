package replay

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
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
	position       int
	paused         bool
	control        chan struct{}
	stop           chan struct{}
}

// Register adds the replay provider factory to the registry.
func Register(reg *api.Registry) {
	reg.Register(providerName, NewFromConfig)
}

// NewFromConfig constructs a replay provider from factory configuration.
func NewFromConfig(cfg api.FactoryConfig) (api.Provider, error) {
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
		control:    make(chan struct{}, 1),
	}
}

// Pause stops replay progress without disconnecting the provider.
func (p *Provider) Pause() { p.mu.Lock(); p.paused = true; p.mu.Unlock() }

// Resume continues a paused replay.
func (p *Provider) Resume() {
	p.mu.Lock()
	p.paused = false
	p.mu.Unlock()
	select {
	case p.control <- struct{}{}:
	default:
	}
}

// SetSpeed changes the elapsed wall-clock duration for future event gaps.
func (p *Provider) SetSpeed(speed float64) {
	if speed <= 0 {
		return
	}
	p.mu.Lock()
	p.speed = speed
	p.mu.Unlock()
	select {
	case p.control <- struct{}{}:
	default:
	}
}

// Seek moves replay to the first recorded event at or after at. It takes effect
// immediately on the active session and updates the replay clock.
func (p *Provider) Seek(at time.Time) {
	p.mu.Lock()
	pos := 0
	for pos < len(p.recorded) && p.recorded[pos].Timestamp.Before(at) {
		pos++
	}
	p.position = pos
	p.clk.Set(at)
	p.mu.Unlock()
	select {
	case p.control <- struct{}{}:
	default:
	}
}

// LoadEvents replaces the recorded event set.
func (p *Provider) LoadEvents(evts []events.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recorded = evts
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
	p.connected = true
	p.stop = make(chan struct{})
	go p.replayLoop(p.stop)
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

func (p *Provider) replayLoop(stop <-chan struct{}) {
	var prev time.Time
	for {
		p.mu.RLock()
		paused := p.paused
		pos := p.position
		total := len(p.recorded)
		speed := p.speed
		var evt events.Event
		if pos < total {
			evt = p.recorded[pos]
		}
		p.mu.RUnlock()
		if pos >= total {
			return
		}
		if paused {
			select {
			case <-stop:
				return
			case <-p.control:
				continue
			}
		}
		select {
		case <-stop:
			return
		default:
		}

		if !prev.IsZero() {
			gap := evt.Timestamp.Sub(prev)
			if speed > 0 {
				gap = time.Duration(float64(gap) / speed)
			}
			if gap > 0 {
				timer := time.NewTimer(gap)
				select {
				case <-stop:
					timer.Stop()
					return
				case <-p.control:
					timer.Stop()
					continue
				case <-timer.C:
				}
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
		case <-stop:
			return
		}
		p.mu.Lock()
		if p.position == pos {
			p.position++
		}
		p.mu.Unlock()
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
