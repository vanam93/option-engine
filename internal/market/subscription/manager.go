// Package subscription keeps desired subscriptions independent of provider connections.
package subscription

import (
	"context"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
	"sort"
	"sync"
)

type Manager struct {
	mu       sync.RWMutex
	provider api.Provider
	desired  map[string]struct{}
	batch    int
}

func New(p api.Provider, batch int) *Manager {
	if batch < 1 {
		batch = 1
	}
	return &Manager{provider: p, desired: map[string]struct{}{}, batch: batch}
}
func (m *Manager) Subscribe(ctx context.Context, symbols []string) error {
	m.mu.Lock()
	for _, s := range symbols {
		if s != "" {
			m.desired[s] = struct{}{}
		}
	}
	p := m.provider
	m.mu.Unlock()
	return m.call(ctx, p.Subscribe, symbols)
}
func (m *Manager) Unsubscribe(ctx context.Context, symbols []string) error {
	m.mu.Lock()
	for _, s := range symbols {
		delete(m.desired, s)
	}
	p := m.provider
	m.mu.Unlock()
	return m.call(ctx, p.Unsubscribe, symbols)
}
func (m *Manager) call(ctx context.Context, fn func(context.Context, []string) error, ss []string) error {
	for i := 0; i < len(ss); i += m.batch {
		e := i + m.batch
		if e > len(ss) {
			e = len(ss)
		}
		if err := fn(ctx, ss[i:e]); err != nil {
			return err
		}
	}
	return nil
}
func (m *Manager) Recover(ctx context.Context, p api.Provider) error {
	m.mu.Lock()
	m.provider = p
	ss := make([]string, 0, len(m.desired))
	for s := range m.desired {
		ss = append(ss, s)
	}
	m.mu.Unlock()
	sort.Strings(ss)
	return m.call(ctx, p.Subscribe, ss)
}
func (m *Manager) Symbols() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.desired))
	for s := range m.desired {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
