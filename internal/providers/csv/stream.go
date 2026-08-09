package csv

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

// Streamer controls replay pacing between candles.
type Streamer struct {
	cfg Config
}

// NewStreamer creates a streamer for the given configuration.
func NewStreamer(cfg Config) *Streamer {
	return &Streamer{cfg: cfg}
}

// Delay calculates wall-clock delay before publishing the next candle.
func (s *Streamer) Delay(prev, current time.Time) time.Duration {
	if !s.cfg.PublishDelay {
		return 0
	}
	return replayDelay(prev, current, s.cfg.ReplaySpeed, s.cfg.InstantReplay)
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

// streamCoordinator manages per-symbol CSV iterators.
type streamCoordinator struct {
	cfg      Config
	streamer *Streamer
	iters    map[string]*symbolIterator
}

type symbolIterator struct {
	iterator *Iterator
	inst     symbolregistry.Instrument
	symbol   string
}

func newStreamCoordinator(cfg Config) *streamCoordinator {
	return &streamCoordinator{
		cfg:      cfg,
		streamer: NewStreamer(cfg),
		iters:    make(map[string]*symbolIterator),
	}
}

func (s *streamCoordinator) Reset(symbols map[string]symbolregistry.Instrument) error {
	s.closeAll()
	s.iters = make(map[string]*symbolIterator, len(symbols))
	for symbol, inst := range symbols {
		if !SymbolsMatch(symbol, s.cfg.Symbol) {
			continue
		}
		it, err := OpenDataFile(s.cfg)
		if err != nil {
			return err
		}
		inst.Symbol = symbol
		s.iters[symbol] = &symbolIterator{
			iterator: it,
			inst:     inst,
			symbol:   symbol,
		}
	}
	return nil
}

func (s *streamCoordinator) closeAll() {
	for _, it := range s.iters {
		if it.iterator != nil {
			_ = it.iterator.Close()
		}
	}
}

func (s *streamCoordinator) Close() {
	s.closeAll()
	s.iters = nil
}

func (s *streamCoordinator) Next() (market.Candle, symbolregistry.Instrument, *Iterator, bool, error) {
	var (
		bestCandle market.Candle
		bestInst   symbolregistry.Instrument
		bestIter   *Iterator
		found      bool
		bestTS     time.Time
	)

	for symbol, symIter := range s.iters {
		for {
			row, ok, err := symIter.iterator.Next()
			if err != nil {
				return market.Candle{}, symbolregistry.Instrument{}, symIter.iterator, false, err
			}
			if !ok {
				break
			}
			if !s.cfg.StartTime.IsZero() && row.Timestamp.Before(s.cfg.StartTime) {
				continue
			}
			if !s.cfg.EndTime.IsZero() && row.Timestamp.After(s.cfg.EndTime) {
				continue
			}

			candle := RowToCandle(row, symbol, s.cfg.Timeframe)
			ts := candleTime(candle)
			if !found || ts.Before(bestTS) {
				found = true
				bestTS = ts
				bestCandle = candle
				bestInst = symIter.inst
				bestIter = symIter.iterator
			}
			break
		}
	}

	return bestCandle, bestInst, bestIter, found, nil
}

func (s *streamCoordinator) CurrentFile() string {
	for _, it := range s.iters {
		if it.iterator != nil {
			return it.iterator.Path()
		}
	}
	return s.cfg.DataFilePath()
}

func (s *streamCoordinator) CurrentOffset() int64 {
	var offset int64
	for _, it := range s.iters {
		if it.iterator != nil && it.iterator.Offset() > offset {
			offset = it.iterator.Offset()
		}
	}
	return offset
}

func (s *streamCoordinator) HasStreams() bool {
	return len(s.iters) > 0
}
