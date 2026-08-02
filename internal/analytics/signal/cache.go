package signal

import (
	"sync"

	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
)

type seriesKey struct {
	symbol    string
	timeframe string
}

type emaPeriodKey struct {
	seriesKey
	period int
}

type emaCrossState struct {
	prevFast float64
	prevSlow float64
	hasPrev  bool
}

type macdCrossState struct {
	prevMACD   float64
	prevSignal float64
	hasPrev    bool
}

// Cache stores minimal state required for crossover detection.
type Cache struct {
	mu        sync.Mutex
	emaValues map[emaPeriodKey]float64
	emaCross  map[seriesKey]*emaCrossState
	macdCross map[seriesKey]*macdCrossState
}

// NewCache creates signal evaluation state storage.
func NewCache() *Cache {
	return &Cache{
		emaValues: make(map[emaPeriodKey]float64),
		emaCross:  make(map[seriesKey]*emaCrossState),
		macdCross: make(map[seriesKey]*macdCrossState),
	}
}

func (c *Cache) setEMAValue(value domainindicator.IndicatorValue, period int, v float64) {
	key := emaPeriodKey{
		seriesKey: seriesKey{symbol: value.Symbol, timeframe: value.Timeframe},
		period:    period,
	}
	c.emaValues[key] = v
}

func (c *Cache) emaValue(symbol, timeframe string, period int) (float64, bool) {
	v, ok := c.emaValues[emaPeriodKey{
		seriesKey: seriesKey{symbol: symbol, timeframe: timeframe},
		period:    period,
	}]
	return v, ok
}

func (c *Cache) emaCrossState(key seriesKey) *emaCrossState {
	state := c.emaCross[key]
	if state == nil {
		state = &emaCrossState{}
		c.emaCross[key] = state
	}
	return state
}

func (c *Cache) macdCrossState(key seriesKey) *macdCrossState {
	state := c.macdCross[key]
	if state == nil {
		state = &macdCrossState{}
		c.macdCross[key] = state
	}
	return state
}
