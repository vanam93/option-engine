package strategy

import (
	"sync"
	"time"
)

const (
	signalEMACross   = "ema_cross"
	signalMACDCross  = "macd_cross"
	signalRSI        = "rsi"
	signalBollinger  = "bollinger"
)

const (
	inputBuy       = "BUY"
	inputSell      = "SELL"
	inputExitLong  = "EXIT_LONG"
	inputExitShort = "EXIT_SHORT"
)

type seriesKey struct {
	symbol    string
	timeframe string
}

type positionState string

const (
	positionFlat  positionState = "flat"
	positionLong  positionState = "long"
	positionShort positionState = "short"
)

type storedSignal struct {
	signal     string
	confidence float64
	at         time.Time
}

type seriesState struct {
	position         positionState
	lastDecision     Decision
	lastDecisionAt   time.Time
	signals          map[string]storedSignal
}

// Cache stores minimal per-series state for strategy evaluation.
type Cache struct {
	mu     sync.Mutex
	series map[seriesKey]*seriesState
}

// NewCache creates strategy evaluation state storage.
func NewCache() *Cache {
	return &Cache{
		series: make(map[seriesKey]*seriesState),
	}
}

func (c *Cache) update(sig InputSignal) seriesKey {
	key := seriesKey{symbol: sig.Symbol, timeframe: sig.Timeframe}
	state := c.series[key]
	if state == nil {
		state = &seriesState{
			position: positionFlat,
			signals:  make(map[string]storedSignal),
		}
		c.series[key] = state
	}
	state.signals[sig.Strategy] = storedSignal{
		signal:     sig.Signal,
		confidence: sig.Confidence,
		at:         sig.Timestamp,
	}
	return key
}

func (c *Cache) state(key seriesKey) *seriesState {
	return c.series[key]
}

func (c *Cache) recordDecision(key seriesKey, decision Decision, at time.Time) {
	state := c.series[key]
	if state == nil {
		return
	}
	state.lastDecision = decision
	state.lastDecisionAt = at
	switch decision {
	case LongEntry:
		state.position = positionLong
	case ShortEntry:
		state.position = positionShort
	case LongExit, ShortExit:
		state.position = positionFlat
	}
}
