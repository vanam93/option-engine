package eventbus

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"testing"
)

func TestFilterAndBackpressure(t *testing.T) {
	b := New()
	defer b.Close()
	s := b.Subscribe(1, func(e events.Event) bool { return e.Type == events.MarketDataReceived })
	b.Publish(events.Event{Type: events.TickPersisted})
	b.Publish(events.Event{Type: events.MarketDataReceived})
	b.Publish(events.Event{Type: events.MarketDataReceived})
	if s.Dropped() != 1 {
		t.Fatalf("dropped=%d", s.Dropped())
	}
	if got := <-s.C; got.Type != events.MarketDataReceived {
		t.Fatal("filter failed")
	}
}
