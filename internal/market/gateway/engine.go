// Package gateway connects providers to the provider-independent market pipeline.
package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	"github.com/vanam-gangireddy/option-engine/internal/market/snapshot"
	"github.com/vanam-gangireddy/option-engine/internal/market/validator"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// Engine owns one pipeline: provider event -> normalization -> validation -> cache -> subscribers.
type Engine struct {
	Source     api.EventSource
	Cache      *cache.Cache
	Bus        *eventbus.Bus
	Validator  *validator.Validator
	Normalizer *normalizer.Normalizer
	now        func() time.Time
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	closed     bool
	wg         sync.WaitGroup
}

func New(source api.EventSource, c *cache.Cache, b *eventbus.Bus, v *validator.Validator, n *normalizer.Normalizer, now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{Source: source, Cache: c, Bus: b, Validator: v, Normalizer: n, now: now}
}

func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	engineCtx, cancel := context.WithCancel(ctx)
	e.ctx = engineCtx
	e.cancel = cancel
	e.started = true
	e.mu.Unlock()

	if e.Source == nil {
		e.mu.Lock()
		e.started = false
		e.mu.Unlock()
		cancel()
		return context.Canceled
	}

	e.wg.Add(1)
	go e.run(engineCtx)
	return nil
}

func (e *Engine) run(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-e.Source.Events():
			if !ok {
				return
			}
			e.handle(evt)
		}
	}
}

func (e *Engine) handle(evt events.Event) {
	if evt.Type != events.MarketDataReceived {
		if e.Bus != nil {
			e.Bus.Publish(evt)
		}
		return
	}

	var payload normalizer.Payload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return
	}
	slog.Debug("gateway received provider payload",
		"provider", evt.Source,
		"symbol", payload.Symbol,
		"timestamp", payload.Timestamp,
	)
	if e.Normalizer == nil {
		return
	}
	if tick, err := e.Normalizer.Tick(payload); err == nil {
		slog.Debug("normalizer produced canonical tick",
			"symbol", tick.Symbol,
			"provider_ts", tick.ProviderTS,
			"received_at", tick.ReceivedAt,
		)
		if e.Validator != nil {
			if err := e.Validator.Validate(tick, e.now()); err != nil {
				slog.Debug("validator rejected tick",
					"symbol", tick.Symbol,
					"error", err.Error(),
				)
				return
			}
			slog.Debug("validator accepted tick",
				"symbol", tick.Symbol,
			)
		}
		if e.Cache != nil {
			e.Cache.PutTick(tick)
			slog.Debug("cache updated",
				"symbol", tick.Symbol,
			)
		}
		canonical, err := events.NewEventWithTime(evt.Type, evt.Source, tick, evt.Timestamp)
		if err == nil && e.Bus != nil {
			e.Bus.Publish(canonical)
			slog.Debug("event bus published event",
				"symbol", tick.Symbol,
				"source", evt.Source,
			)
		}
		if e.Cache != nil {
			snap := snapshot.New(e.Cache, e.now())
			slog.Debug("snapshot updated",
				"symbol", tick.Symbol,
				"snapshot_symbols", len(snap.Ticks),
			)
		}
	}
}

func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	cancel := e.cancel
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()
	if e.Bus != nil {
		e.Bus.Close()
	}
	return nil
}
