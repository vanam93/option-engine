package opportunity

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/ports"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/debuglog"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

// Engine consumes scanner intelligence and publishes ranked opportunity events.
type Engine struct {
	cfg     Config
	bus     ports.EventBus
	clk     clock.Clock
	cache   *Cache
	scorer  *Scorer
	ranker  *Ranker
	health  healthSnapshot
	summary Summary

	mu           sync.Mutex
	historyMu    sync.Mutex
	history      []RankedOpportunity
	ctx          context.Context
	cancel       context.CancelFunc
	subscription *eventbus.Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup
}

const maxOpportunityHistory = 10000

// New creates an opportunity ranking engine.
func New(cfg Config, bus ports.EventBus, clk clock.Clock) (*Engine, error) {
	cfg = cfg.WithDefaults()
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
		cfg:    cfg,
		bus:    bus,
		clk:    clk,
		cache:  NewCache(),
		scorer: NewScorer(cfg),
		ranker: NewRanker(cfg),
	}, nil
}

// Start subscribes to intelligence events before the consumer goroutine starts.
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
		switch evt.Type {
		case events.ScannerUpdated,
			events.SignalGenerated,
			events.StrategyDecision,
			events.ApprovedTradeIntent,
			events.PerformanceUpdated,
			events.OptimizationUpdated,
			events.WalkForwardCompleted,
			events.MonteCarloCompleted:
			return true
		default:
			return false
		}
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
		e.handleStrategy(evt.Payload)
	case events.ApprovedTradeIntent:
		e.handleRisk(evt.Payload)
	case events.PerformanceUpdated:
		e.handlePerformance(evt.Payload)
	case events.OptimizationUpdated:
		e.handleOptimization(evt.Payload)
	case events.WalkForwardCompleted:
		e.handleWalkForward(evt.Payload)
	case events.MonteCarloCompleted:
		e.handleMonteCarlo(evt.Payload)
	case events.ScannerUpdated:
		e.handleScanner(evt.Payload)
	}
}

func (e *Engine) handleSignal(payload json.RawMessage) {
	input, ok := parseSignal(payload)
	if !ok {
		return
	}
	e.cache.ApplySignal(input.Symbol, input.Timeframe, input.Confidence, input.Timestamp)
}

func (e *Engine) handleStrategy(payload json.RawMessage) {
	input, ok := parseStrategy(payload)
	if !ok {
		return
	}
	e.cache.ApplyStrategy(input.Symbol, input.Timeframe, input.Confidence, input.Timestamp)
}

func (e *Engine) handleRisk(payload json.RawMessage) {
	input, ok := parseRisk(payload)
	if !ok {
		return
	}
	e.cache.ApplyRisk(input.Symbol, input.Timeframe, input.Approved, input.Timestamp)
}

func (e *Engine) handlePerformance(payload json.RawMessage) {
	input, ok := parsePerformance(payload)
	if !ok {
		return
	}
	e.cache.ApplyPerformance(input.Symbol, input.Timeframe, input.WinRate, input.RealizedPnL, input.UnrealizedPnL, input.Timestamp)
}

func (e *Engine) handleOptimization(payload json.RawMessage) {
	input, ok := parseOptimization(payload)
	if !ok {
		return
	}
	e.cache.ApplyOptimization(input.Symbol, input.Timeframe, input.Score, input.Timestamp)
}

func (e *Engine) handleWalkForward(payload json.RawMessage) {
	input, ok := parseWalkForward(payload)
	if !ok {
		return
	}
	e.cache.ApplyWalkForward(input.ValidationScore, input.Timestamp)
}

func (e *Engine) handleMonteCarlo(payload json.RawMessage) {
	input, ok := parseMonteCarlo(payload)
	if !ok {
		return
	}
	e.cache.ApplyMonteCarlo(input.ProbabilityOfProfit, input.Timestamp)
}

func (e *Engine) handleScanner(payload json.RawMessage) {
	input, ok := parseScanner(payload)
	if !ok {
		return
	}
	e.cache.ApplyScanner(input)
	// #region agent log
	if e.cache.SymbolCount() <= 3 || e.cache.SymbolCount()%500 == 0 {
		debuglog.Write("A", "opportunity/engine.go:handleScanner", "scanner applied to opportunity cache", map[string]any{
			"symbol": input.Symbol, "timeframe": input.Timeframe, "scannerTS": input.Timestamp.UTC().Format(time.RFC3339),
			"cacheKeys": e.cache.SymbolCount(),
		})
	}
	// #endregion
	e.rankAndPublish(input.Timestamp)
}

func (e *Engine) rankAndPublish(at time.Time) {
	if at.IsZero() {
		at = e.clk.Now().UTC()
	}
	states := e.cache.AllSymbols()
	platform := e.cache.Platform()
	ranked := e.ranker.Rank(states, platform, e.scorer)
	top := e.ranker.TopN(ranked)
	summary := Summarize(ranked, top)

	e.mu.Lock()
	e.summary = summary
	e.mu.Unlock()
	e.health.update(summary)

	for _, item := range top {
		e.publish(item, at)
	}
}

func (e *Engine) publish(item RankedOpportunity, at time.Time) {
	e.historyMu.Lock()
	e.history = append(e.history, item)
	if len(e.history) > maxOpportunityHistory {
		e.history = e.history[len(e.history)-maxOpportunityHistory:]
	}
	historyLen := len(e.history)
	e.historyMu.Unlock()
	// #region agent log
	if historyLen <= 3 || historyLen%1000 == 0 {
		debuglog.Write("A", "opportunity/engine.go:publish", "opportunity history appended", map[string]any{
			"symbol": item.Symbol, "timeframe": item.Timeframe, "historyLen": historyLen,
			"itemTS": item.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	// #endregion
	out, err := events.NewEventWithClock(e.clk, events.OpportunityUpdated, engineName, OpportunityUpdated{
		Symbol:         item.Symbol,
		Timeframe:      item.Timeframe,
		Rank:           item.Rank,
		Confidence:     item.Confidence,
		Classification: item.Classification,
		Score:          item.Score,
		Components:     item.Components,
		Timestamp:      at,
	})
	if err != nil {
		return
	}
	e.bus.Publish(out)
}

// OpportunitySnapshot is an immutable read model of ranked opportunities.
type OpportunitySnapshot struct {
	Ranked   []RankedOpportunity `json:"ranked"`
	History  []RankedOpportunity `json:"history,omitempty"`
	Platform PlatformState       `json:"platform"`
	Summary  Summary             `json:"summary"`
}

// Snapshot returns the current ranked opportunity state.
func (e *Engine) Snapshot() OpportunitySnapshot {
	states := e.cache.AllSymbols()
	platform := e.cache.Platform()
	ranked := e.ranker.Rank(states, platform, e.scorer)
	top := e.ranker.TopN(ranked)
	e.historyMu.Lock()
	history := append([]RankedOpportunity(nil), e.history...)
	e.historyMu.Unlock()

	// #region agent log
	stateTS := ""
	if len(states) > 0 {
		stateTS = states[0].UpdatedAt.UTC().Format(time.RFC3339)
	}
	rankedTS := ""
	if len(top) > 0 {
		rankedTS = top[0].Timestamp.UTC().Format(time.RFC3339)
	}
	debuglog.Write("B", "opportunity/engine.go:Snapshot", "api snapshot built from live cache", map[string]any{
		"cacheKeys": len(states), "topN": len(top), "historyLen": len(history),
		"firstStateUpdatedAt": stateTS, "firstRankedTimestamp": rankedTS, "runId": "post-fix",
	})
	// #endregion

	e.mu.Lock()
	summary := e.summary
	e.mu.Unlock()
	if summary.OpportunitiesRanked == 0 && len(ranked) > 0 {
		summary = Summarize(ranked, top)
	}

	return OpportunitySnapshot{
		Ranked:   top,
		History:  history,
		Platform: platform,
		Summary:  summary,
	}
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
	summary := e.summary
	if e.subscription != nil {
		dropped = e.subscription.Dropped()
	}
	e.mu.Unlock()

	return e.health.report(e.cfg, connected, dropped, summary)
}

func parseScanner(payload json.RawMessage) (InputScanner, bool) {
	var raw struct {
		Symbol       string    `json:"symbol"`
		Timeframe    string    `json:"timeframe"`
		ScannerName  string    `json:"scanner_name"`
		Status       string    `json:"status"`
		Score        float64   `json:"score"`
		Confidence   float64   `json:"confidence"`
		MatchedRules []string  `json:"matched_rules"`
		Timestamp    time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return InputScanner{}, false
	}
	if raw.Symbol == "" || raw.Timeframe == "" {
		return InputScanner{}, false
	}
	return InputScanner{
		Symbol:       raw.Symbol,
		Timeframe:    raw.Timeframe,
		ScannerName:  raw.ScannerName,
		Status:       raw.Status,
		Score:        raw.Score,
		Confidence:   raw.Confidence,
		MatchedRules: raw.MatchedRules,
		Timestamp:    raw.Timestamp,
	}, true
}

func parseSignal(payload json.RawMessage) (struct {
	Symbol, Timeframe string
	Confidence        float64
	Timestamp         time.Time
}, bool) {
	var raw struct {
		Symbol     string    `json:"symbol"`
		Timeframe  string    `json:"timeframe"`
		Confidence float64   `json:"confidence"`
		Timestamp  time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil || raw.Symbol == "" {
		return struct {
			Symbol, Timeframe string
			Confidence        float64
			Timestamp         time.Time
		}{}, false
	}
	return struct {
		Symbol, Timeframe string
		Confidence        float64
		Timestamp         time.Time
	}{raw.Symbol, raw.Timeframe, raw.Confidence, raw.Timestamp}, true
}

func parseStrategy(payload json.RawMessage) (struct {
	Symbol, Timeframe string
	Confidence        float64
	Timestamp         time.Time
}, bool) {
	var raw struct {
		Symbol     string    `json:"symbol"`
		Timeframe  string    `json:"timeframe"`
		Confidence float64   `json:"confidence"`
		Timestamp  time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil || raw.Symbol == "" {
		return struct {
			Symbol, Timeframe string
			Confidence        float64
			Timestamp         time.Time
		}{}, false
	}
	return struct {
		Symbol, Timeframe string
		Confidence        float64
		Timestamp         time.Time
	}{raw.Symbol, raw.Timeframe, raw.Confidence, raw.Timestamp}, true
}

func parseRisk(payload json.RawMessage) (struct {
	Symbol, Timeframe string
	Approved          bool
	Timestamp         time.Time
}, bool) {
	var raw struct {
		Symbol    string    `json:"symbol"`
		Timeframe string    `json:"timeframe"`
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil || raw.Symbol == "" {
		return struct {
			Symbol, Timeframe string
			Approved          bool
			Timestamp         time.Time
		}{}, false
	}
	return struct {
		Symbol, Timeframe string
		Approved          bool
		Timestamp         time.Time
	}{raw.Symbol, raw.Timeframe, raw.Status == "APPROVED", raw.Timestamp}, true
}

func parsePerformance(payload json.RawMessage) (struct {
	Symbol, Timeframe          string
	WinRate                    float64
	RealizedPnL, UnrealizedPnL float64
	Timestamp                  time.Time
}, bool) {
	var raw struct {
		Symbol        string    `json:"symbol"`
		Timeframe     string    `json:"timeframe"`
		WinRate       float64   `json:"win_rate"`
		RealizedPnL   float64   `json:"realized_pnl"`
		UnrealizedPnL float64   `json:"unrealized_pnl"`
		Timestamp     time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return struct {
			Symbol, Timeframe          string
			WinRate                    float64
			RealizedPnL, UnrealizedPnL float64
			Timestamp                  time.Time
		}{}, false
	}
	symbol := raw.Symbol
	if symbol == "" {
		symbol = "portfolio"
	}
	timeframe := raw.Timeframe
	if timeframe == "" {
		timeframe = "1m"
	}
	return struct {
		Symbol, Timeframe          string
		WinRate                    float64
		RealizedPnL, UnrealizedPnL float64
		Timestamp                  time.Time
	}{symbol, timeframe, raw.WinRate, raw.RealizedPnL, raw.UnrealizedPnL, raw.Timestamp}, true
}

func parseOptimization(payload json.RawMessage) (struct {
	Symbol, Timeframe string
	Score             float64
	Timestamp         time.Time
}, bool) {
	var raw struct {
		Symbol    string    `json:"symbol"`
		Timeframe string    `json:"timeframe"`
		Score     float64   `json:"score"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return struct {
			Symbol, Timeframe string
			Score             float64
			Timestamp         time.Time
		}{}, false
	}
	symbol := raw.Symbol
	if symbol == "" {
		symbol = "portfolio"
	}
	timeframe := raw.Timeframe
	if timeframe == "" {
		timeframe = "1m"
	}
	return struct {
		Symbol, Timeframe string
		Score             float64
		Timestamp         time.Time
	}{symbol, timeframe, raw.Score, raw.Timestamp}, true
}

func parseWalkForward(payload json.RawMessage) (struct {
	ValidationScore float64
	Timestamp       time.Time
}, bool) {
	var raw struct {
		ValidationScore float64   `json:"validation_score"`
		Timestamp       time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return struct {
			ValidationScore float64
			Timestamp       time.Time
		}{}, false
	}
	return struct {
		ValidationScore float64
		Timestamp       time.Time
	}{raw.ValidationScore, raw.Timestamp}, true
}

func parseMonteCarlo(payload json.RawMessage) (struct {
	ProbabilityOfProfit float64
	Timestamp           time.Time
}, bool) {
	var raw struct {
		ProbabilityOfProfit float64   `json:"probability_of_profit"`
		Timestamp           time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return struct {
			ProbabilityOfProfit float64
			Timestamp           time.Time
		}{}, false
	}
	return struct {
		ProbabilityOfProfit float64
		Timestamp           time.Time
	}{raw.ProbabilityOfProfit, raw.Timestamp}, true
}
