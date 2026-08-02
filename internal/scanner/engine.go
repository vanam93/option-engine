package scanner

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine consumes analytics events and publishes scanner.updated intelligence events.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	evaluator *Evaluator
	cache     *Cache
	health    healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a market scanner engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock) (*Engine, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if bus == nil {
		return nil, ErrNilBus
	}
	if clk == nil {
		clk = clock.NewSystem()
	}
	return &Engine{
		cfg:       cfg,
		bus:       bus,
		clk:       clk,
		evaluator: NewEvaluator(cfg),
		cache:     NewCache(),
	}, nil
}

// Start subscribes to scanner input events before the consumer goroutine starts.
func (e *Engine) Start(ctx context.Context) error {
	if !e.cfg.Enabled {
		return nil
	}

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
	e.subscription = e.bus.Subscribe(e.cfg.SubscriberBuffer, func(evt events.Event) bool {
		return evt.Type == events.SignalGenerated ||
			evt.Type == events.StrategyDecision ||
			evt.Type == events.PerformanceUpdated
	})
	e.started = true
	e.mu.Unlock()

	e.wg.Add(1)
	go e.run(engineCtx)
	return nil
}

func (e *Engine) run(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			e.drain()
			return
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handle(evt)
		}
	}
}

func (e *Engine) drain() {
	for {
		select {
		case evt, ok := <-e.subscription.C:
			if !ok {
				return
			}
			e.handle(evt)
		default:
			return
		}
	}
}

func (e *Engine) handle(evt events.Event) {
	switch evt.Type {
	case events.SignalGenerated:
		e.handleSignal(evt.Payload)
	case events.StrategyDecision:
		e.handleDecision(evt.Payload)
	case events.PerformanceUpdated:
		e.handlePerformance(evt.Payload)
	}
}

func (e *Engine) handleSignal(payload json.RawMessage) {
	input, ok := parseInputSignal(payload)
	if !ok || !e.cfg.WatchesSymbol(input.Symbol) {
		return
	}
	e.health.recordEvent()
	state := e.cache.applySignal(input)
	results := e.evaluator.EvaluateSignal(state)
	e.publishResults(results)
}

func (e *Engine) handleDecision(payload json.RawMessage) {
	input, ok := parseInputDecision(payload)
	if !ok || !e.cfg.WatchesSymbol(input.Symbol) {
		return
	}
	e.health.recordEvent()
	state := e.cache.applyDecision(input)
	results := e.evaluator.EvaluateDecision(state)
	e.publishResults(results)
}

func (e *Engine) handlePerformance(payload json.RawMessage) {
	input, ok := parseInputPerformance(payload)
	if !ok || !e.cfg.WatchesSymbol(input.Symbol) {
		return
	}
	e.health.recordEvent()
	_ = e.cache.applyPerformance(input)
	performances := e.cache.allPerformance()
	results := e.evaluator.EvaluateRanking(performances, input)
	e.publishResults(results)
}

func parseInputSignal(payload json.RawMessage) (InputSignal, bool) {
	var raw struct {
		Symbol     string    `json:"symbol"`
		Timeframe  string    `json:"timeframe"`
		Signal     string    `json:"signal"`
		Strategy   string    `json:"strategy"`
		Confidence float64   `json:"confidence"`
		Timestamp  time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputSignal{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" {
		return InputSignal{}, false
	}
	return InputSignal{
		Symbol:     raw.Symbol,
		Timeframe:  raw.Timeframe,
		Signal:     raw.Signal,
		Strategy:   raw.Strategy,
		Confidence: raw.Confidence,
		Timestamp:  raw.Timestamp,
	}, true
}

func parseInputDecision(payload json.RawMessage) (InputDecision, bool) {
	var raw struct {
		Symbol     string    `json:"symbol"`
		Timeframe  string    `json:"timeframe"`
		Decision   string    `json:"decision"`
		Strategy   string    `json:"strategy"`
		Confidence float64   `json:"confidence"`
		Reason     string    `json:"reason"`
		Timestamp  time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputDecision{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" {
		return InputDecision{}, false
	}
	return InputDecision{
		Symbol:     raw.Symbol,
		Timeframe:  raw.Timeframe,
		Decision:   raw.Decision,
		Strategy:   raw.Strategy,
		Confidence: raw.Confidence,
		Reason:     raw.Reason,
		Timestamp:  raw.Timestamp,
	}, true
}

func parseInputPerformance(payload json.RawMessage) (InputPerformance, bool) {
	var raw struct {
		Symbol        string    `json:"symbol"`
		Timeframe     string    `json:"timeframe"`
		Strategy      string    `json:"strategy"`
		TotalTrades   int       `json:"total_trades"`
		WinRate       float64   `json:"win_rate"`
		RealizedPnL   float64   `json:"realized_pnl"`
		UnrealizedPnL float64   `json:"unrealized_pnl"`
		Drawdown      float64   `json:"drawdown"`
		Timestamp     time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputPerformance{}, false
	}
	symbol := raw.Symbol
	if symbol == "" {
		symbol = "portfolio"
	}
	timeframe := raw.Timeframe
	if timeframe == "" {
		timeframe = "1m"
	}
	strategy := raw.Strategy
	if strategy == "" {
		strategy = "default"
	}
	return InputPerformance{
		Symbol:        symbol,
		Timeframe:     timeframe,
		Strategy:      strategy,
		TotalTrades:   raw.TotalTrades,
		WinRate:       raw.WinRate,
		RealizedPnL:   raw.RealizedPnL,
		UnrealizedPnL: raw.UnrealizedPnL,
		Drawdown:      raw.Drawdown,
		Timestamp:     raw.Timestamp,
	}, true
}

func (e *Engine) publishResults(results []ScanResult) {
	for _, result := range results {
		if result.Status == StatusNeutral {
			continue
		}
		e.publish(result)
	}
}

func (e *Engine) publish(result ScanResult) {
	out, err := events.NewEventWithClock(e.clk, events.ScannerUpdated, engineName, result.toEvent())
	if err != nil {
		return
	}
	e.bus.Publish(out)
	e.cache.storeResult(result)
	e.health.recordMatch(result.Timestamp)
}

// ScannerSnapshot is an immutable read model of scanner state.
type ScannerSnapshot struct {
	Results []ScanResult  `json:"results"`
	States  []SymbolState `json:"states"`
}

// Snapshot returns the latest scanner results and symbol states.
func (e *Engine) Snapshot() ScannerSnapshot {
	results, states := e.cache.Snapshot()
	return ScannerSnapshot{Results: results, States: states}
}

// Close stops the engine and releases its subscription.
func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	cancel := e.cancel
	sub := e.subscription
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()
	if sub != nil {
		sub.Close()
	}
	return nil
}

// Health reports runtime status for observability probes.
func (e *Engine) Health() health.Report {
	dropped := uint64(0)
	e.mu.Lock()
	connected := e.started && !e.closed
	if e.subscription != nil {
		dropped = e.subscription.Dropped()
	}
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, e.cache.symbolCount())
}
