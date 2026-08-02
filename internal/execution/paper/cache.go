package paper

import (
	"sync"
)

const (
	actionLongEntry  = "LONG_ENTRY"
	actionShortEntry = "SHORT_ENTRY"
	actionLongExit   = "LONG_EXIT"
	actionShortExit  = "SHORT_EXIT"
)

type seriesKey struct {
	symbol    string
	timeframe string
}

type positionSide string

const (
	positionFlat  positionSide = "flat"
	positionLong  positionSide = "long"
	positionShort positionSide = "short"
)

type position struct {
	Symbol     string
	Timeframe  string
	Side       positionSide
	Quantity   int
	EntryPrice float64
	OrderID    string
	Strategy   string
}

// Cache stores executed orders, order sequencing, and open positions.
type Cache struct {
	mu           sync.Mutex
	orderCounter uint64
	executed     []ExecutionReport
	positions    map[seriesKey]*position
}

// NewCache creates paper execution state storage.
func NewCache() *Cache {
	return &Cache{
		positions: make(map[seriesKey]*position),
	}
}

func (c *Cache) nextOrderID() string {
	c.orderCounter++
	return formatOrderID(c.orderCounter)
}

func (c *Cache) record(report ExecutionReport) {
	c.executed = append(c.executed, report)
}

func (c *Cache) apply(report ExecutionReport) {
	if report.Status != Filled {
		return
	}
	key := seriesKey{symbol: report.Symbol, timeframe: report.Timeframe}
	switch report.Action {
	case actionLongEntry:
		c.positions[key] = &position{
			Symbol:     report.Symbol,
			Timeframe:  report.Timeframe,
			Side:       positionLong,
			Quantity:   report.Quantity,
			EntryPrice: report.ExecutionPrice,
			OrderID:    report.OrderID,
			Strategy:   report.Strategy,
		}
	case actionShortEntry:
		c.positions[key] = &position{
			Symbol:     report.Symbol,
			Timeframe:  report.Timeframe,
			Side:       positionShort,
			Quantity:   report.Quantity,
			EntryPrice: report.ExecutionPrice,
			OrderID:    report.OrderID,
			Strategy:   report.Strategy,
		}
	case actionLongExit, actionShortExit:
		delete(c.positions, key)
	}
}

func (c *Cache) activePositions() int {
	count := 0
	for _, pos := range c.positions {
		if pos.Side == positionLong || pos.Side == positionShort {
			count++
		}
	}
	return count
}

// ActivePositions returns the number of open long or short positions.
func (c *Cache) ActivePositions() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.activePositions()
}

func (c *Cache) position(symbol, timeframe string) (position, bool) {
	pos := c.positions[seriesKey{symbol: symbol, timeframe: timeframe}]
	if pos == nil {
		return position{}, false
	}
	return *pos, true
}

func formatOrderID(n uint64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
