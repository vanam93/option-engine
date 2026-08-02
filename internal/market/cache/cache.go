// Package cache maintains the latest immutable copies of market state.
package cache

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/domain/option"
	"sync"
)

type DepthLevel struct {
	Price    float64
	Quantity int64
}
type Depth struct{ Bids, Asks []DepthLevel }
type Cache struct {
	mu     sync.RWMutex
	ticks  map[string]market.Tick
	chains map[string]option.OptionChainSnapshot
	depths map[string]Depth
}

func New() *Cache {
	return &Cache{ticks: make(map[string]market.Tick), chains: make(map[string]option.OptionChainSnapshot), depths: make(map[string]Depth)}
}
func (c *Cache) PutTick(t market.Tick) { c.mu.Lock(); c.ticks[t.Symbol] = t; c.mu.Unlock() }
func (c *Cache) Tick(symbol string) (market.Tick, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.ticks[symbol]
	return t, ok
}
func (c *Cache) PutChain(v option.OptionChainSnapshot) {
	v.Contracts = append([]option.OptionContract(nil), v.Contracts...)
	c.mu.Lock()
	c.chains[v.Underlying] = v
	c.mu.Unlock()
}
func (c *Cache) Chain(symbol string) (option.OptionChainSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.chains[symbol]
	v.Contracts = append([]option.OptionContract(nil), v.Contracts...)
	return v, ok
}
func cloneDepth(d Depth) Depth {
	return Depth{Bids: append([]DepthLevel(nil), d.Bids...), Asks: append([]DepthLevel(nil), d.Asks...)}
}
func (c *Cache) PutDepth(symbol string, d Depth) {
	c.mu.Lock()
	c.depths[symbol] = cloneDepth(d)
	c.mu.Unlock()
}
func (c *Cache) Depth(symbol string) (Depth, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	d, ok := c.depths[symbol]
	return cloneDepth(d), ok
}
func (c *Cache) Ticks() map[string]market.Tick {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]market.Tick, len(c.ticks))
	for k, v := range c.ticks {
		out[k] = v
	}
	return out
}
