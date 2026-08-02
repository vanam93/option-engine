package ports

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// EventBus is the minimal Stage 2 event bus contract used by analytics engines.
// Analytics code must only consume canonical events published by the gateway.
type EventBus interface {
	Subscribe(buffer int, filter eventbus.Filter) *eventbus.Subscription
	Publish(e events.Event)
}
