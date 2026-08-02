package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/subscription"
	providerapi "github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// Manager owns the lifecycle of the active market data provider.
type Manager struct {
	mu           sync.RWMutex
	provider     Provider
	reg          *Registry
	cfg          ManagerConfig
	subscription *subscription.Manager
	session      *providerSession
}

type providerSession struct {
	mu       sync.RWMutex
	provider Provider
}

func newProviderSession(p Provider) *providerSession {
	return &providerSession{provider: p}
}

func (s *providerSession) Events() <-chan events.Event {
	s.mu.RLock()
	p := s.provider
	s.mu.RUnlock()
	if p == nil {
		return nil
	}
	return p.Events()
}

func (s *providerSession) Health() health.Report {
	s.mu.RLock()
	p := s.provider
	s.mu.RUnlock()
	if p == nil {
		return health.Report{Component: "provider_session", Status: health.StatusUnhealthy, Message: "provider session not initialized"}
	}
	return p.Health()
}

func (s *providerSession) SetProvider(p Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provider = p
}

// ManagerConfig drives provider selection and runtime behaviour.
type ManagerConfig struct {
	ActiveProvider  string
	ProviderCfg     map[string]any
	Reconnect       ReconnectConfig
	Subscription    SubscriptionConfig
	Heartbeat       HeartbeatConfig
	ShutdownTimeout time.Duration
}

// NewManager creates a manager without an active provider.
func NewManager(reg *Registry, cfg ManagerConfig) *Manager {
	return &Manager{reg: reg, cfg: cfg}
}

// Init creates and stores the configured provider.
func (m *Manager) Init() error {
	return m.InitWithDeps(FactoryConfig{
		ProviderCfg: m.cfg.ProviderCfg,
	})
}

// InitWithProvider wires a pre-built provider instance (for example backtest replay).
func (m *Manager) InitWithProvider(p Provider) error {
	if p == nil {
		return fmt.Errorf("provider manager: nil provider")
	}
	m.mu.Lock()
	m.provider = p
	if m.session == nil {
		m.session = newProviderSession(p)
	} else {
		m.session.SetProvider(p)
	}
	m.mu.Unlock()
	return nil
}

// InitWithDeps creates the configured provider with injected dependencies.
func (m *Manager) InitWithDeps(base FactoryConfig) error {
	base.Reconnect = m.cfg.Reconnect
	base.Subscription = m.cfg.Subscription
	base.Heartbeat = m.cfg.Heartbeat
	if base.ProviderCfg == nil {
		base.ProviderCfg = m.cfg.ProviderCfg
	}

	p, err := m.reg.Create(m.cfg.ActiveProvider, base)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.provider = p
	if m.session == nil {
		m.session = newProviderSession(p)
	} else {
		m.session.SetProvider(p)
	}
	m.mu.Unlock()
	return nil
}

// Connect connects the active provider.
func (m *Manager) Connect(ctx context.Context) error {
	p, err := m.active()
	if err != nil {
		return err
	}
	if err := p.Connect(ctx); err != nil {
		return err
	}
	if m.subscription != nil {
		_ = m.subscription.Recover(ctx, p)
	}
	return nil
}

// Disconnect disconnects the active provider.
func (m *Manager) Disconnect(ctx context.Context) error {
	p, err := m.active()
	if err != nil {
		return err
	}
	return p.Disconnect(ctx)
}

// Subscribe delegates to the active provider in batches.
func (m *Manager) Subscribe(ctx context.Context, symbols []string) error {
	p, err := m.active()
	if err != nil {
		return err
	}
	batch := m.cfg.Subscription.BatchSize
	if batch <= 0 {
		batch = 200
	}
	for i := 0; i < len(symbols); i += batch {
		end := i + batch
		if end > len(symbols) {
			end = len(symbols)
		}
		if err := p.Subscribe(ctx, symbols[i:end]); err != nil {
			return err
		}
	}
	return nil
}

// SetSubscriptionManager wires the stateful subscription tracker into the provider lifecycle.
func (m *Manager) SetSubscriptionManager(sub *subscription.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscription = sub
}

// Unsubscribe delegates to the active provider.
func (m *Manager) Unsubscribe(ctx context.Context, symbols []string) error {
	p, err := m.active()
	if err != nil {
		return err
	}
	return p.Unsubscribe(ctx, symbols)
}

// Events is intentionally unavailable for runtime use; the gateway is the sole consumer of provider events.
func (m *Manager) Events() (<-chan events.Event, error) {
	return nil, fmt.Errorf("provider manager no longer exposes raw provider events")
}

// Provider returns the active provider instance.
func (m *Manager) Provider() (Provider, error) {
	return m.active()
}

// Session returns the runtime event source backed by the active provider.
func (m *Manager) Session() providerapi.EventSource {
	m.mu.RLock()
	if m.session != nil {
		s := m.session
		m.mu.RUnlock()
		return s
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		m.session = newProviderSession(nil)
	}
	return m.session
}

// Health aggregates health from the active provider.
func (m *Manager) Health() health.Report {
	p, err := m.active()
	if err != nil {
		return health.Report{
			Component: "provider_manager",
			Status:    health.StatusUnhealthy,
			Message:   err.Error(),
		}
	}
	report := p.Health()
	report.Component = "provider_manager"
	return report
}

// Switch replaces the active provider at runtime (config-driven deployments).
func (m *Manager) Switch(ctx context.Context, name string) error {
	if err := m.Disconnect(ctx); err != nil {
		// best effort
	}

	p, err := m.reg.Create(name, FactoryConfig{Name: name, ProviderCfg: m.cfg.ProviderCfg, Reconnect: m.cfg.Reconnect, Subscription: m.cfg.Subscription, Heartbeat: m.cfg.Heartbeat})
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.provider = p
	m.cfg.ActiveProvider = name
	if m.session == nil {
		m.session = newProviderSession(p)
	} else {
		m.session.SetProvider(p)
	}
	m.mu.Unlock()

	if err := p.Connect(ctx); err != nil {
		return err
	}
	if m.subscription != nil {
		_ = m.subscription.Recover(ctx, p)
	}
	return nil
}

func (m *Manager) active() (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.provider == nil {
		return nil, fmt.Errorf("provider manager not initialized")
	}
	return m.provider, nil
}

// ReconnectLoop runs automatic reconnects according to configuration.
func (m *Manager) ReconnectLoop(ctx context.Context) {
	interval := ParseDuration(m.cfg.Reconnect.Interval, "5s")
	maxRetries := MaxRetries(m.cfg.Reconnect.MaxRetries)
	var attempts int64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		report := m.Health()
		if report.Connected {
			attempts = 0
			if !wait(ctx, interval) {
				return
			}
			continue
		}

		if maxRetries >= 0 && attempts >= int64(maxRetries) {
			return
		}

		attempts++
		_ = m.Connect(ctx)
		if !wait(ctx, interval) {
			return
		}
	}
}

func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
