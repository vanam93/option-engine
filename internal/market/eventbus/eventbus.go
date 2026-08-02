// Package eventbus provides bounded, filtered, non-blocking fan-out.
package eventbus

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"sync"
	"sync/atomic"
)

type Filter func(events.Event) bool
type Subscription struct {
	C       <-chan events.Event
	cancel  func()
	dropped *atomic.Uint64
}

func (s *Subscription) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
func (s *Subscription) Dropped() uint64 { return s.dropped.Load() }

type subscriber struct {
	ch      chan events.Event
	filter  Filter
	dropped atomic.Uint64
}
type Bus struct {
	mu     sync.RWMutex
	subs   map[uint64]*subscriber
	next   uint64
	closed bool
}

func New() *Bus { return &Bus{subs: make(map[uint64]*subscriber)} }
func (b *Bus) Subscribe(buffer int, f Filter) *Subscription {
	if buffer < 1 {
		buffer = 1
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := &subscriber{ch: make(chan events.Event, buffer), filter: f}
	if b.closed {
		close(s.ch)
		return &Subscription{C: s.ch, dropped: &s.dropped}
	}
	id := b.next
	b.next++
	b.subs[id] = s
	return &Subscription{C: s.ch, dropped: &s.dropped, cancel: func() { b.remove(id) }}
}
func (b *Bus) remove(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if s, ok := b.subs[id]; ok {
		delete(b.subs, id)
		close(s.ch)
	}
}
func (b *Bus) Publish(e events.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, s := range b.subs {
		if s.filter != nil && !s.filter(e) {
			continue
		}
		select {
		case s.ch <- e:
		default:
			s.dropped.Add(1)
		}
	}
}
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, s := range b.subs {
		delete(b.subs, id)
		close(s.ch)
	}
}
func (b *Bus) Len() int { b.mu.RLock(); defer b.mu.RUnlock(); return len(b.subs) }
