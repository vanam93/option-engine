// Package snapshot creates immutable read models from the market cache.
package snapshot

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/domain/option"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
	"time"
)

type Market struct {
	At     time.Time
	Ticks  map[string]market.Tick
	Chains map[string]option.OptionChainSnapshot
	Depths map[string]cache.Depth
}

func New(c *cache.Cache, at time.Time) Market {
	m := Market{At: at.UTC(), Ticks: c.Ticks(), Chains: map[string]option.OptionChainSnapshot{}, Depths: map[string]cache.Depth{}}
	for s := range m.Ticks {
		if d, ok := c.Depth(s); ok {
			m.Depths[s] = d
		}
	}
	for _, t := range m.Ticks {
		if ch, ok := c.Chain(t.Symbol); ok {
			m.Chains[t.Symbol] = ch
		}
	}
	return m
}
