package risk

import (
	"sync"
	"time"
)

const (
	actionLongEntry  = "LONG_ENTRY"
	actionShortEntry = "SHORT_ENTRY"
	actionLongExit   = "LONG_EXIT"
	actionShortExit  = "SHORT_EXIT"
	actionHold       = "HOLD"
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

type seriesState struct {
	position positionState
}

// Cache stores per-series position state and daily trade counters.
type Cache struct {
	mu               sync.Mutex
	series           map[seriesKey]*seriesState
	tradesToday      int
	resetDay         time.Time
	dayResetLocation *time.Location
}

// NewCache creates risk evaluation state storage.
func NewCache(timezone string) (*Cache, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	return &Cache{
		series:           make(map[seriesKey]*seriesState),
		dayResetLocation: loc,
	}, nil
}

func (c *Cache) resetDayIfNeeded(at time.Time) {
	if c.dayResetLocation == nil {
		return
	}
	day := at.In(c.dayResetLocation)
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, c.dayResetLocation)
	if c.resetDay.IsZero() || dayStart.After(c.resetDay) {
		c.resetDay = dayStart
		c.tradesToday = 0
	}
}

func (c *Cache) seriesState(key seriesKey) *seriesState {
	state := c.series[key]
	if state == nil {
		state = &seriesState{position: positionFlat}
		c.series[key] = state
	}
	return state
}

func (c *Cache) activePositions() int {
	count := 0
	for _, state := range c.series {
		if state.position == positionLong || state.position == positionShort {
			count++
		}
	}
	return count
}

func (c *Cache) apply(intent ApprovedTradeIntent) {
	if intent.Status != Approved {
		return
	}
	c.tradesToday++
	key := seriesKey{symbol: intent.Symbol, timeframe: intent.Timeframe}
	state := c.seriesState(key)
	switch intent.Action {
	case actionLongEntry:
		state.position = positionLong
	case actionShortEntry:
		state.position = positionShort
	case actionLongExit, actionShortExit:
		state.position = positionFlat
	}
}
