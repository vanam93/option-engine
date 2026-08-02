package groww

import (
	"container/heap"
	"context"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

type queuedCandle struct {
	candle    market.Candle
	inst      symbolregistry.Instrument
	symbolKey string
	ts        time.Time
	index     int
}

type candleHeap []*queuedCandle

func (h candleHeap) Len() int           { return len(h) }
func (h candleHeap) Less(i, j int) bool { return h[i].ts.Before(h[j].ts) }
func (h candleHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *candleHeap) Push(x any) {
	item := x.(*queuedCandle)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *candleHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

// streamCoordinator merges per-symbol candle iterators in timestamp order.
type streamCoordinator struct {
	hist     *HistoricalService
	cfg      Config
	interval string
	heap     candleHeap
	iters    map[string]*CandleIterator
}

func newStreamCoordinator(hist *HistoricalService, cfg Config) *streamCoordinator {
	return &streamCoordinator{
		hist:     hist,
		cfg:      cfg,
		interval: cfg.CandleInterval,
		iters:    make(map[string]*CandleIterator),
	}
}

func (s *streamCoordinator) Reset(symbols map[string]symbolregistry.Instrument) {
	s.iters = make(map[string]*CandleIterator, len(symbols))
	s.heap = nil
	for symbol, inst := range symbols {
		inst.Symbol = symbol
		s.iters[symbol] = NewCandleIterator(s.hist, symbol, inst, s.cfg.StartTime, s.cfg.EndTime, s.interval, s.cfg.Timeframe)
	}
}

func (s *streamCoordinator) Prime(ctx context.Context) error {
	h := &s.heap
	heap.Init(h)
	for symbol, it := range s.iters {
		candle, ok, err := it.Next(ctx)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		heap.Push(h, &queuedCandle{
			candle:    candle,
			inst:      it.inst,
			symbolKey: symbol,
			ts:        candleTime(candle),
		})
	}
	return nil
}

func (s *streamCoordinator) Next(ctx context.Context) (market.Candle, symbolregistry.Instrument, bool, error) {
	if len(s.heap) == 0 {
		return market.Candle{}, symbolregistry.Instrument{}, false, nil
	}
	item := heap.Pop(&s.heap).(*queuedCandle)
	if it, ok := s.iters[item.symbolKey]; ok {
		if next, has, err := it.Next(ctx); err != nil {
			return market.Candle{}, symbolregistry.Instrument{}, false, err
		} else if has {
			heap.Push(&s.heap, &queuedCandle{
				candle:    next,
				inst:      it.inst,
				symbolKey: item.symbolKey,
				ts:        candleTime(next),
			})
		}
	}
	return item.candle, item.inst, true, nil
}

func candleTime(c market.Candle) time.Time {
	if !c.CloseTime.IsZero() {
		return c.CloseTime
	}
	return c.OpenTime
}

// replayDelay calculates wall-clock delay between candles for replay speed control.
func replayDelay(prev, current time.Time, speed float64, instant bool) time.Duration {
	if instant || speed <= 0 {
		return 0
	}
	if prev.IsZero() {
		return 0
	}
	gap := current.Sub(prev)
	if gap <= 0 {
		return 0
	}
	return time.Duration(float64(gap) / speed)
}
