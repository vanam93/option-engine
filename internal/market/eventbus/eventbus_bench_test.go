package eventbus

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"testing"
)

func BenchmarkPublish(b *testing.B) {
	bus := New()
	defer bus.Close()
	bus.Subscribe(4096, nil)
	e := events.Event{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(e)
	}
}
