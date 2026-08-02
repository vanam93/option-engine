// Package gateway connects providers to the provider-independent market pipeline.
package gateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/market/validator"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// Engine owns one pipeline: provider event -> validation -> cache -> subscribers.
type Engine struct {
	Provider  api.Provider
	Cache     *cache.Cache
	Bus       *eventbus.Bus
	Validator *validator.Validator
	now       func() time.Time
	wg        sync.WaitGroup
}

func New(p api.Provider, c *cache.Cache, b *eventbus.Bus, v *validator.Validator, now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{Provider: p, Cache: c, Bus: b, Validator: v, now: now}
}
func (e *Engine) Start(ctx context.Context) error {
	if err := e.Provider.Connect(ctx); err != nil {
		return err
	}
	e.wg.Add(1)
	go e.run(ctx)
	return nil
}
func (e *Engine) run(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-e.Provider.Events():
			if !ok {
				return
			}
			e.handle(evt)
		}
	}
}
func (e *Engine) handle(evt events.Event) {
	if evt.Type != events.MarketDataReceived {
		e.Bus.Publish(evt)
		return
	}
	var tick market.Tick
	if json.Unmarshal(evt.Payload, &tick) != nil {
		return
	}
	if e.Validator != nil && e.Validator.Validate(tick, e.now()) != nil {
		return
	}
	e.Cache.PutTick(tick)
	e.Bus.Publish(evt)
}
func (e *Engine) Close(ctx context.Context) error {
	err := e.Provider.Disconnect(ctx)
	e.wg.Wait()
	e.Bus.Close()
	return err
}
