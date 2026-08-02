package portfolio

import (
	"sync"
)

type seriesKey struct {
	symbol    string
	timeframe string
}

type positionState struct {
	Position
	ReferencePrice float64
}

type applyResult struct {
	Updated       PortfolioUpdated
	TradeRecorded bool
}

// Cache stores positions, trade history, and PnL summaries.
type Cache struct {
	mu              sync.Mutex
	positions       map[seriesKey]*positionState
	trades          []TradeRecord
	realizedTotal   float64
	unrealizedTotal float64
	exposure        float64
	referencePrice  map[string]float64
}

// NewCache creates portfolio state storage.
func NewCache() *Cache {
	return &Cache{
		positions:      make(map[seriesKey]*positionState),
		referencePrice: make(map[string]float64),
	}
}

// Apply processes a filled execution report and returns the portfolio update.
func (c *Cache) Apply(report InputReport) (applyResult, bool) {
	if report.Status != statusFilled {
		return applyResult{}, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.trades = append(c.trades, tradeFromReport(report))
	c.referencePrice[report.Symbol] = report.ExecutionPrice

	key := seriesKey{symbol: report.Symbol, timeframe: report.Timeframe}
	var tradeRealized float64

	switch report.Action {
	case actionLongEntry:
		c.positions[key] = &positionState{
			Position: Position{
				Symbol:       report.Symbol,
				Timeframe:    report.Timeframe,
				Side:         SideLong,
				Quantity:     report.Quantity,
				AveragePrice: report.ExecutionPrice,
				EntryTime:    report.Timestamp,
			},
			ReferencePrice: report.ExecutionPrice,
		}
	case actionShortEntry:
		c.positions[key] = &positionState{
			Position: Position{
				Symbol:       report.Symbol,
				Timeframe:    report.Timeframe,
				Side:         SideShort,
				Quantity:     report.Quantity,
				AveragePrice: report.ExecutionPrice,
				EntryTime:    report.Timestamp,
			},
			ReferencePrice: report.ExecutionPrice,
		}
	case actionLongExit, actionShortExit:
		pos := c.positions[key]
		if pos == nil {
			break
		}
		tradeRealized = RealizedPnL(pos.Side, pos.AveragePrice, report.ExecutionPrice, pos.Quantity)
		c.realizedTotal += tradeRealized
		delete(c.positions, key)
	default:
		return applyResult{}, false
	}

	c.recomputeAggregates()

	var positionCopy *Position
	if pos := c.positions[key]; pos != nil {
		copyPos := pos.Position
		positionCopy = &copyPos
	}

	return applyResult{
		Updated: PortfolioUpdated{
			Symbol:        report.Symbol,
			Position:      positionCopy,
			RealizedPnL:   c.realizedTotal,
			UnrealizedPnL: c.symbolUnrealized(report.Symbol),
			Timestamp:     report.Timestamp,
		},
		TradeRecorded: true,
	}, true
}

func (c *Cache) recomputeAggregates() {
	c.unrealizedTotal = 0
	c.exposure = 0
	for _, pos := range c.positions {
		ref := pos.ReferencePrice
		if latest, ok := c.referencePrice[pos.Symbol]; ok && latest > 0 {
			ref = latest
		}
		c.unrealizedTotal += UnrealizedPnL(pos.Side, pos.AveragePrice, ref, pos.Quantity)
		c.exposure += Exposure(pos.Side, ref, pos.Quantity)
	}
}

func (c *Cache) symbolUnrealized(symbol string) float64 {
	var total float64
	for _, pos := range c.positions {
		if pos.Symbol != symbol {
			continue
		}
		ref := pos.ReferencePrice
		if latest, ok := c.referencePrice[symbol]; ok && latest > 0 {
			ref = latest
		}
		total += UnrealizedPnL(pos.Side, pos.AveragePrice, ref, pos.Quantity)
	}
	return total
}

// Snapshot returns an immutable copy of the current portfolio state.
func (c *Cache) Snapshot() PortfolioState {
	c.mu.Lock()
	defer c.mu.Unlock()

	positions := make([]Position, 0, len(c.positions))
	for _, pos := range c.positions {
		positions = append(positions, pos.Position)
	}
	trades := append([]TradeRecord(nil), c.trades...)
	return PortfolioState{
		Positions:     positions,
		Trades:        trades,
		RealizedPnL:   c.realizedTotal,
		UnrealizedPnL: c.unrealizedTotal,
		Exposure:      c.exposure,
	}
}

func (c *Cache) positionsCount() int {
	return len(c.positions)
}

func (c *Cache) tradesProcessed() int {
	return len(c.trades)
}

func (c *Cache) realizedPnL() float64 {
	return c.realizedTotal
}

func (c *Cache) unrealizedPnL() float64 {
	return c.unrealizedTotal
}

func tradeFromReport(report InputReport) TradeRecord {
	return TradeRecord{
		OrderID:        report.OrderID,
		Symbol:         report.Symbol,
		Timeframe:      report.Timeframe,
		Action:         report.Action,
		Quantity:       report.Quantity,
		ExecutionPrice: report.ExecutionPrice,
		Strategy:       report.Strategy,
		Timestamp:      report.Timestamp,
	}
}
