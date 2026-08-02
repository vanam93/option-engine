// Package normalizer converts provider-specific payloads into domain ticks.
package normalizer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// Payload is the provider-neutral wire representation accepted by Normalizer.
// Adapters translate broker messages into this structure; business code only sees Tick.
type Payload struct {
	Symbol, Exchange                        string
	InstrumentType                          market.InstrumentType
	LTP, Open, High, Low, Close, Bid, Ask   float64
	Volume, BidQty, AskQty, OI, SequenceNum int64
	Timestamp                               time.Time
}

type Normalizer struct{ now func() time.Time }

func New(now func() time.Time) *Normalizer {
	if now == nil {
		now = time.Now
	}
	return &Normalizer{now: now}
}
func (n *Normalizer) Tick(p Payload) (market.Tick, error) {
	if strings.TrimSpace(p.Symbol) == "" {
		return market.Tick{}, fmt.Errorf("normalize: missing symbol")
	}
	if p.Timestamp.IsZero() {
		return market.Tick{}, fmt.Errorf("normalize: missing timestamp")
	}
	return market.Tick{ID: uuid.New(), Symbol: strings.ToUpper(strings.TrimSpace(p.Symbol)), Exchange: strings.ToUpper(strings.TrimSpace(p.Exchange)), InstrumentType: p.InstrumentType, LTP: p.LTP, Open: p.Open, High: p.High, Low: p.Low, Close: p.Close, Bid: p.Bid, Ask: p.Ask, Volume: p.Volume, BidQty: p.BidQty, AskQty: p.AskQty, OI: p.OI, SequenceNum: p.SequenceNum, ProviderTS: p.Timestamp.UTC(), ReceivedAt: n.now().UTC()}, nil
}
