package opportunity

import (
	"sync"
	"time"
)

type symbolKey struct {
	symbol    string
	timeframe string
}

// SymbolState holds accumulated intelligence for a symbol.
type SymbolState struct {
	Symbol    string
	Timeframe string

	SignalConfidence   float64
	StrategyConfidence float64
	RiskApproved       bool
	PerformanceScore   float64
	OptimizationScore  float64

	ScannerScore      float64
	ScannerConfidence float64
	ScannerStatus     string

	UpdatedAt time.Time
}

// PlatformState holds cross-symbol research intelligence.
type PlatformState struct {
	WalkForwardScore float64
	MonteCarloScore  float64
	UpdatedAt        time.Time
}

// Cache maintains per-symbol and platform intelligence state.
type Cache struct {
	mu       sync.Mutex
	symbols  map[symbolKey]*SymbolState
	platform PlatformState
}

// NewCache creates an empty opportunity cache.
func NewCache() *Cache {
	return &Cache{symbols: make(map[symbolKey]*SymbolState)}
}

func (c *Cache) stateFor(symbol, timeframe string) *SymbolState {
	key := symbolKey{symbol: symbol, timeframe: timeframe}
	state, ok := c.symbols[key]
	if !ok {
		state = &SymbolState{Symbol: symbol, Timeframe: timeframe}
		c.symbols[key] = state
	}
	return state
}

func (c *Cache) ApplySignal(symbol, timeframe string, confidence float64, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.stateFor(symbol, timeframe)
	state.SignalConfidence = clamp01(confidence)
	state.UpdatedAt = at
}

func (c *Cache) ApplyStrategy(symbol, timeframe string, confidence float64, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.stateFor(symbol, timeframe)
	state.StrategyConfidence = clamp01(confidence)
	state.UpdatedAt = at
}

func (c *Cache) ApplyRisk(symbol, timeframe string, approved bool, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.stateFor(symbol, timeframe)
	state.RiskApproved = approved
	state.UpdatedAt = at
}

func (c *Cache) ApplyPerformance(symbol, timeframe string, winRate float64, realized, unrealized float64, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.stateFor(symbol, timeframe)
	state.PerformanceScore = clamp01(winRate*0.6 + normalizePnL(realized+unrealized)*0.4)
	state.UpdatedAt = at
}

func (c *Cache) ApplyOptimization(symbol, timeframe string, score float64, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.stateFor(symbol, timeframe)
	state.OptimizationScore = clamp01(score)
	state.UpdatedAt = at
}

func (c *Cache) ApplyScanner(input InputScanner) *SymbolState {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.stateFor(input.Symbol, input.Timeframe)
	state.ScannerScore = clamp01(input.Score)
	state.ScannerConfidence = clamp01(input.Confidence)
	state.ScannerStatus = input.Status
	state.UpdatedAt = input.Timestamp
	return cloneState(state)
}

func (c *Cache) ApplyWalkForward(validationScore float64, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.platform.WalkForwardScore = clamp01(validationScore)
	c.platform.UpdatedAt = at
}

func (c *Cache) ApplyMonteCarlo(probabilityOfProfit float64, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.platform.MonteCarloScore = clamp01(probabilityOfProfit)
	c.platform.UpdatedAt = at
}

func (c *Cache) AllSymbols() []SymbolState {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]SymbolState, 0, len(c.symbols))
	for _, state := range c.symbols {
		out = append(out, *cloneState(state))
	}
	return out
}

func (c *Cache) Platform() PlatformState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.platform
}

func (c *Cache) SymbolCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.symbols)
}

func cloneState(state *SymbolState) *SymbolState {
	if state == nil {
		return nil
	}
	copy := *state
	return &copy
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func normalizePnL(pnl float64) float64 {
	if pnl <= 0 {
		return 0
	}
	if pnl >= 1000 {
		return 1
	}
	return pnl / 1000
}
