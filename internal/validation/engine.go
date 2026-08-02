package validation

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

// Engine consumes recommendation.updated events and publishes validated.recommendation events.
type Engine struct {
	cfg       Config
	bus       ports.EventBus
	clk       clock.Clock
	cache     *Cache
	validator *Validator
	health    healthSnapshot

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

// New creates a recommendation validation engine subscribed to recommendation.updated events only.
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
		cache:     NewCache(),
		validator: NewValidator(cfg),
	}, nil
}

// Start subscribes to recommendation.updated before the consumer goroutine starts.
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
		return evt.Type == events.RecommendationUpdated
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
	input, ok := parseRecommendation(evt.Payload)
	if !ok {
		return
	}

	at := e.clk.Now().UTC()
	previous, _ := e.cache.Get(input.Symbol, input.Timeframe)
	if e.validator.IsDuplicate(input, previous) {
		e.health.record(ValidatedRecommendation{}, 0, true, false)
		return
	}

	outcome := e.validator.Validate(input, at)
	result := outcome.Result

	expired := false
	for _, reason := range result.RejectionReasons {
		if reason == "recommendation is stale" {
			expired = true
			break
		}
	}

	e.cache.Put(result)
	e.publish(result)
	e.health.record(result, outcome.Score, false, expired)
}

func parseRecommendation(payload json.RawMessage) (InputRecommendation, bool) {
	var raw struct {
		Symbol              string    `json:"symbol"`
		Timeframe           string    `json:"timeframe"`
		Recommendation      string    `json:"recommendation"`
		Confidence          float64   `json:"confidence"`
		OptimizationScore   *float64  `json:"optimization_score"`
		WalkforwardScore    *float64  `json:"walkforward_score"`
		MonteCarloScore     *float64  `json:"monte_carlo_score"`
		WinRate             *float64  `json:"win_rate"`
		Drawdown            *float64  `json:"drawdown"`
		OptimizationSummary string    `json:"optimization_summary"`
		WalkForwardSummary  string    `json:"walk_forward_summary"`
		MonteCarloSummary   string    `json:"monte_carlo_summary"`
		GeneratedAt         time.Time `json:"generated_at"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputRecommendation{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" {
		return InputRecommendation{}, false
	}

	input := InputRecommendation{
		Symbol:              raw.Symbol,
		Timeframe:           raw.Timeframe,
		Recommendation:      recommendation.Level(strings.ToUpper(strings.TrimSpace(raw.Recommendation))),
		Confidence:          raw.Confidence,
		OptimizationSummary: raw.OptimizationSummary,
		WalkForwardSummary:  raw.WalkForwardSummary,
		MonteCarloSummary:   raw.MonteCarloSummary,
		GeneratedAt:         raw.GeneratedAt,
	}

	if raw.OptimizationScore != nil {
		input.OptimizationScore = *raw.OptimizationScore
		input.hasOptimizationScore = true
	}
	if raw.WalkforwardScore != nil {
		input.WalkforwardScore = *raw.WalkforwardScore
		input.hasWalkforwardScore = true
	}
	if raw.MonteCarloScore != nil {
		input.MonteCarloScore = *raw.MonteCarloScore
		input.hasMonteCarloScore = true
	}
	if raw.WinRate != nil {
		input.WinRate = *raw.WinRate
		input.hasWinRate = true
	}
	if raw.Drawdown != nil {
		input.Drawdown = *raw.Drawdown
		input.hasDrawdown = true
	}

	return input, true
}

func (e *Engine) publish(result ValidatedRecommendation) {
	out, err := events.NewEventWithClock(e.clk, events.ValidatedRecommendation, engineName, result)
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// Latest returns the most recent validation for a symbol and timeframe.
func (e *Engine) Latest(symbol, timeframe string) (ValidatedRecommendation, bool) {
	return e.cache.Get(symbol, timeframe)
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

	return e.health.report(e.cfg, connected, dropped)
}
