package candle

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

type seriesKey struct {
	symbol    string
	timeframe market.Timeframe
}

type barBuilder struct {
	candle     market.Candle
	vwapNumer  float64
	vwapDenom  float64
	lastVolume int64
	hasLastVol bool
}

func newBarBuilder(symbol string, tf market.Timeframe, open, close time.Time) barBuilder {
	return barBuilder{
		candle: market.Candle{
			ID:        uuid.New(),
			Symbol:    symbol,
			Timeframe: tf,
			OpenTime:  open,
			CloseTime: close,
		},
	}
}

func tickWeight(tick market.Tick, mode VolumeMode, lastVolume int64, hasLastVol bool) (weight int64, nextLast int64, hasNext bool) {
	switch mode {
	case VolumeIncremental:
		if tick.Volume <= 0 {
			return 0, lastVolume, hasLastVol
		}
		return tick.Volume, lastVolume, hasLastVol
	default: // VolumeCumulative
		if tick.Volume <= 0 {
			return 0, lastVolume, hasLastVol
		}
		if !hasLastVol {
			return tick.Volume, tick.Volume, true
		}
		if tick.Volume > lastVolume {
			return tick.Volume - lastVolume, tick.Volume, true
		}
		if tick.Volume < lastVolume {
			return tick.Volume, tick.Volume, true
		}
		return 0, lastVolume, true
	}
}

func (b *barBuilder) addTick(tick market.Tick, mode VolumeMode) {
	price := tick.LTP
	if b.candle.TradeCount == 0 {
		b.candle.Open = price
		b.candle.High = price
		b.candle.Low = price
		b.candle.Close = price
	} else {
		if price > b.candle.High {
			b.candle.High = price
		}
		if price < b.candle.Low {
			b.candle.Low = price
		}
		b.candle.Close = price
	}

	weight, nextLast, hasNext := tickWeight(tick, mode, b.lastVolume, b.hasLastVol)
	if hasNext {
		b.lastVolume = nextLast
		b.hasLastVol = true
	}

	if weight > 0 {
		b.candle.Volume += weight
		b.vwapNumer += price * float64(weight)
		b.vwapDenom += float64(weight)
		if b.vwapDenom > 0 {
			b.candle.VWAP = b.vwapNumer / b.vwapDenom
		}
	}
	b.candle.TradeCount++
}

func (b *barBuilder) build() market.Candle {
	return b.candle
}

func (b *barBuilder) hasTicks() bool {
	return b.candle.TradeCount > 0
}

// UpdateStats records aggregation counters for observability.
type UpdateStats struct {
	Closed    int
	Rejected  int
	Evicted   int
}

// Aggregator maintains in-progress candles per symbol and timeframe.
type Aggregator struct {
	mu           sync.Mutex
	loc          *time.Location
	volumeMode   VolumeMode
	orderPolicy  OrderPolicy
	idleEvict    time.Duration
	bars         map[seriesKey]*barBuilder
	order        []seriesKey
	lastActivity map[seriesKey]time.Time
	lastClosed   map[seriesKey]time.Time
}

// AggregatorOptions configures aggregation behaviour.
type AggregatorOptions struct {
	VolumeMode  VolumeMode
	OrderPolicy OrderPolicy
	IdleEvict   time.Duration
}

func NewAggregator(loc *time.Location, opts AggregatorOptions) *Aggregator {
	if opts.VolumeMode == "" {
		opts.VolumeMode = VolumeCumulative
	}
	if opts.OrderPolicy == "" {
		opts.OrderPolicy = OrderRejectOlder
	}
	return &Aggregator{
		loc:          loc,
		volumeMode:   opts.VolumeMode,
		orderPolicy:  opts.OrderPolicy,
		idleEvict:    opts.IdleEvict,
		bars:         make(map[seriesKey]*barBuilder),
		lastActivity: make(map[seriesKey]time.Time),
		lastClosed:   make(map[seriesKey]time.Time),
	}
}

// Update ingests a canonical tick and returns completed candles plus stats.
func (a *Aggregator) Update(tick market.Tick, timeframes []market.Timeframe) ([]market.Candle, UpdateStats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ts := tick.ProviderTS
	if ts.IsZero() {
		ts = tick.ReceivedAt
	}
	ts = ts.In(a.loc)

	var closed []market.Candle
	var stats UpdateStats

	if a.idleEvict > 0 {
		evicted := a.evictIdleLocked(tick.Symbol, ts)
		closed = append(closed, evicted...)
		stats.Evicted = len(evicted)
	}

	for _, tf := range timeframes {
		open, err := BucketStart(ts, tf, a.loc)
		if err != nil {
			return nil, stats, err
		}
		closeAt, err := BucketClose(open, tf)
		if err != nil {
			return nil, stats, err
		}

		key := seriesKey{symbol: tick.Symbol, timeframe: tf}
		builder, exists := a.bars[key]

		if exists {
			switch {
			case open.Before(builder.candle.OpenTime):
				stats.Rejected++
				continue
			case a.orderPolicy == OrderRejectStale:
				if last, ok := a.lastClosed[key]; ok && !open.After(last) {
					stats.Rejected++
					continue
				}
			case open.Equal(builder.candle.OpenTime):
				// same bucket: apply tick below
			case open.After(builder.candle.OpenTime):
				if builder.hasTicks() {
					closed = append(closed, builder.build())
					a.lastClosed[key] = builder.candle.OpenTime
				}
				created := newBarBuilder(tick.Symbol, tf, open, closeAt)
				a.bars[key] = &created
				builder = a.bars[key]
			default:
				stats.Rejected++
				continue
			}
		} else {
			if a.orderPolicy == OrderRejectStale {
				if last, ok := a.lastClosed[key]; ok && !open.After(last) {
					stats.Rejected++
					continue
				}
			}
			created := newBarBuilder(tick.Symbol, tf, open, closeAt)
			a.bars[key] = &created
			a.order = append(a.order, key)
			builder = a.bars[key]
		}

		builder.addTick(tick, a.volumeMode)
		a.lastActivity[key] = ts
	}
	return closed, stats, nil
}

func (a *Aggregator) evictIdleLocked(symbol string, now time.Time) []market.Candle {
	var closed []market.Candle
	keep := a.order[:0]
	for _, key := range a.order {
		if key.symbol != symbol {
			keep = append(keep, key)
			continue
		}
		last, ok := a.lastActivity[key]
		if !ok || now.Sub(last) <= a.idleEvict {
			keep = append(keep, key)
			continue
		}
		if builder, ok := a.bars[key]; ok && builder.hasTicks() {
			closed = append(closed, builder.build())
			a.lastClosed[key] = builder.candle.OpenTime
		}
		delete(a.bars, key)
		delete(a.lastActivity, key)
	}
	a.order = keep
	return closed
}

// Flush returns all in-progress candles in stable order and clears builder state.
func (a *Aggregator) Flush() []market.Candle {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]market.Candle, 0, len(a.bars))
	for _, key := range a.order {
		if builder, ok := a.bars[key]; ok && builder.hasTicks() {
			out = append(out, builder.build())
		}
	}
	a.bars = make(map[seriesKey]*barBuilder)
	a.order = a.order[:0]
	a.lastActivity = make(map[seriesKey]time.Time)
	a.lastClosed = make(map[seriesKey]time.Time)
	return out
}

// Snapshot returns a copy of all in-progress candles.
func (a *Aggregator) Snapshot() []market.Candle {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]market.Candle, 0, len(a.bars))
	for _, key := range a.order {
		if builder, ok := a.bars[key]; ok && builder.hasTicks() {
			out = append(out, builder.build())
		}
	}
	return out
}

// ActiveBuilders returns the number of in-progress series.
func (a *Aggregator) ActiveBuilders() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.bars)
}
