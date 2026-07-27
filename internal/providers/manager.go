package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/option-engine/option-engine/internal/core/health"
	"github.com/option-engine/option-engine/internal/domain/events"
)

// Manager owns the lifecycle of the active market data provider.
type Manager struct {
	mu       sync.RWMutex
	provider Provider
	reg      *Registry
	cfg      ManagerConfig
}

// ManagerConfig drives provider selection and runtime behaviour.
type ManagerConfig struct {
	ActiveProvider string
	ProviderCfg    map[string]any
	Reconnect      ReconnectConfig
	Subscription   SubscriptionConfig
	Heartbeat      HeartbeatConfig
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
	m.mu.Unlock()
	return nil
}

// Connect connects the active provider.
func (m *Manager) Connect(ctx context.Context) error {
	p, err := m.active()
	if err != nil {
		return err
	}
	return p.Connect(ctx)
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

// Unsubscribe delegates to the active provider.
func (m *Manager) Unsubscribe(ctx context.Context, symbols []string) error {
	p, err := m.active()
	if err != nil {
		return err
	}
	return p.Unsubscribe(ctx, symbols)
}

// Events returns the event stream from the active provider.
func (m *Manager) Events() (<-chan events.Event, error) {
	p, err := m.active()
	if err != nil {
		return nil, err
	}
	return p.Events(), nil
}

// Provider returns the active provider instance.
func (m *Manager) Provider() (Provider, error) {
	return m.active()
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

	p, err := CreateFromConfig(
		m.reg,
		name,
		m.cfg.Reconnect,
		m.cfg.Subscription,
		m.cfg.Heartbeat,
		m.cfg.ProviderCfg,
	)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.provider = p
	m.cfg.ActiveProvider = name
	m.mu.Unlock()

	return p.Connect(ctx)
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
			time.Sleep(interval)
			continue
		}

		if maxRetries >= 0 && attempts >= int64(maxRetries) {
			return
		}

		attempts++
		_ = m.Connect(ctx)
		time.Sleep(interval)
	}
}
