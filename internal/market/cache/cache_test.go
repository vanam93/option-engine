package cache

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"testing"
)

func TestDepthIsCopied(t *testing.T) {
	c := New()
	d := Depth{Bids: []DepthLevel{{Price: 1}}}
	c.PutDepth("N", d)
	d.Bids[0].Price = 2
	got, _ := c.Depth("N")
	if got.Bids[0].Price != 1 {
		t.Fatal("mutable input leaked")
	}
	c.PutTick(market.Tick{Symbol: "N"})
	if _, ok := c.Tick("N"); !ok {
		t.Fatal("tick missing")
	}
}
