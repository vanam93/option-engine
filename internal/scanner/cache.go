package scanner

import (
	"sync"
	"time"
)

type symbolKey struct {
	symbol    string
	timeframe string
}

// SymbolState holds the latest intelligence inputs for a symbol.
type SymbolState struct {
	Symbol       string
	Timeframe    string
	LastSignal   InputSignal
	LastDecision InputDecision
	Performance  InputPerformance
	HasSignal    bool
	HasDecision  bool
	HasPerf      bool
	UpdatedAt    time.Time
}

// Cache maintains per-symbol scanner state.
type Cache struct {
	mu      sync.Mutex
	byKey   map[symbolKey]*SymbolState
	symbols map[string]struct{}
	latest  map[symbolKey]ScanResult
}

// NewCache creates an empty scanner cache.
func NewCache() *Cache {
	return &Cache{
		byKey:   make(map[symbolKey]*SymbolState),
		symbols: make(map[string]struct{}),
		latest:  make(map[symbolKey]ScanResult),
	}
}

func (c *Cache) applySignal(input InputSignal) *SymbolState {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := symbolKey{symbol: input.Symbol, timeframe: input.Timeframe}
	state, ok := c.byKey[key]
	if !ok {
		state = &SymbolState{Symbol: input.Symbol, Timeframe: input.Timeframe}
		c.byKey[key] = state
	}
	state.LastSignal = input
	state.HasSignal = true
	state.UpdatedAt = input.Timestamp
	c.symbols[input.Symbol] = struct{}{}
	return state
}

func (c *Cache) applyDecision(input InputDecision) *SymbolState {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := symbolKey{symbol: input.Symbol, timeframe: input.Timeframe}
	state, ok := c.byKey[key]
	if !ok {
		state = &SymbolState{Symbol: input.Symbol, Timeframe: input.Timeframe}
		c.byKey[key] = state
	}
	state.LastDecision = input
	state.HasDecision = true
	state.UpdatedAt = input.Timestamp
	c.symbols[input.Symbol] = struct{}{}
	return state
}

func (c *Cache) applyPerformance(input InputPerformance) *SymbolState {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := symbolKey{symbol: input.Symbol, timeframe: input.Timeframe}
	state, ok := c.byKey[key]
	if !ok {
		state = &SymbolState{Symbol: input.Symbol, Timeframe: input.Timeframe}
		c.byKey[key] = state
	}
	state.Performance = input
	state.HasPerf = true
	state.UpdatedAt = input.Timestamp
	c.symbols[input.Symbol] = struct{}{}
	return state
}

func (c *Cache) allPerformance() []InputPerformance {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]InputPerformance, 0, len(c.byKey))
	for _, state := range c.byKey {
		if state.HasPerf {
			out = append(out, state.Performance)
		}
	}
	return out
}

func (c *Cache) symbolCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.symbols)
}

func (c *Cache) storeResult(result ScanResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := symbolKey{symbol: result.Symbol, timeframe: result.Timeframe}
	c.latest[key] = result
}

// Snapshot returns the latest scanner results and symbol states.
func (c *Cache) Snapshot() ([]ScanResult, []SymbolState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	results := make([]ScanResult, 0, len(c.latest))
	for _, result := range c.latest {
		results = append(results, result)
	}

	states := make([]SymbolState, 0, len(c.byKey))
	for _, state := range c.byKey {
		states = append(states, *state)
	}
	return results, states
}
