// Package snapshot creates immutable read models from the market cache.
package snapshot

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/domain/option"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
)

type Market struct {
	At     time.Time
	Ticks  map[string]market.Tick
	Chains map[string]option.OptionChainSnapshot
	Depths map[string]cache.Depth
}

func New(c *cache.Cache, at time.Time) Market {
	m := Market{At: at.UTC(), Ticks: map[string]market.Tick{}, Chains: map[string]option.OptionChainSnapshot{}, Depths: map[string]cache.Depth{}}
	for symbol, tick := range c.Ticks() {
		m.Ticks[symbol] = cloneTick(tick)
		if d, ok := c.Depth(symbol); ok {
			m.Depths[symbol] = cloneDepth(d)
		}
		if ch, ok := c.Chain(symbol); ok {
			m.Chains[symbol] = cloneChain(ch)
		}
	}
	return m
}

func cloneTick(t market.Tick) market.Tick {
	return t
}

func cloneDepth(d cache.Depth) cache.Depth {
	return cache.Depth{Bids: append([]cache.DepthLevel(nil), d.Bids...), Asks: append([]cache.DepthLevel(nil), d.Asks...)}
}

func cloneChain(ch option.OptionChainSnapshot) option.OptionChainSnapshot {
	clone := ch
	clone.Contracts = append([]option.OptionContract(nil), ch.Contracts...)
	return clone
}
