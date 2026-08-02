package performance

import (
	"sync"
	"time"
)

type applyResult struct {
	Snapshot PerformanceSnapshot
	Updated  PerformanceUpdated
}

// Cache stores trade history, equity curve, and performance counters.
type Cache struct {
	mu sync.Mutex

	trades          []TradeResult
	equityCurve     []EquityPoint
	openSymbols     map[string]bool
	lastRealizedPnL float64

	totalTrades   int
	winningTrades int
	losingTrades  int
	grossProfit   float64
	grossLoss     float64
	tradePnLTotal float64

	realizedPnL   float64
	unrealizedPnL float64

	drawdown DrawdownState
}

// NewCache creates performance analytics state storage.
func NewCache() *Cache {
	return &Cache{
		openSymbols: make(map[string]bool),
	}
}

// Apply processes a portfolio update and returns the latest performance snapshot.
func (c *Cache) Apply(update InputUpdate) applyResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	wasOpen := c.openSymbols[update.Symbol]
	if update.PositionOpen {
		c.openSymbols[update.Symbol] = true
	} else {
		if wasOpen {
			tradePnL := update.RealizedPnL - c.lastRealizedPnL
			c.recordTrade(update.Symbol, tradePnL, update.Timestamp)
		}
		c.openSymbols[update.Symbol] = false
	}

	c.lastRealizedPnL = update.RealizedPnL
	c.realizedPnL = update.RealizedPnL
	c.unrealizedPnL = update.UnrealizedPnL

	net := NetPnL(c.realizedPnL, c.unrealizedPnL)
	c.equityCurve = append(c.equityCurve, EquityPoint{
		Equity:    net,
		Timestamp: update.Timestamp,
	})
	c.drawdown.Update(net)

	snapshot := c.buildSnapshot(update.Timestamp)
	return applyResult{
		Snapshot: snapshot,
		Updated:  snapshotToEvent(snapshot),
	}
}

func (c *Cache) recordTrade(symbol string, pnl float64, at time.Time) {
	c.trades = append(c.trades, TradeResult{
		Symbol:    symbol,
		PnL:       pnl,
		Timestamp: at,
	})
	c.totalTrades++
	c.tradePnLTotal += pnl
	if pnl > 0 {
		c.winningTrades++
		c.grossProfit += pnl
	} else if pnl < 0 {
		c.losingTrades++
		c.grossLoss += -pnl
	}
}

func (c *Cache) buildSnapshot(at time.Time) PerformanceSnapshot {
	equity := make([]float64, len(c.equityCurve))
	for i, point := range c.equityCurve {
		equity[i] = point.Equity
	}

	trades := append([]TradeResult(nil), c.trades...)
	curve := append([]EquityPoint(nil), c.equityCurve...)

	return PerformanceSnapshot{
		TotalTrades:     c.totalTrades,
		WinningTrades:   c.winningTrades,
		LosingTrades:    c.losingTrades,
		WinRate:         WinRate(c.winningTrades, c.totalTrades),
		RealizedPnL:     c.realizedPnL,
		UnrealizedPnL:   c.unrealizedPnL,
		NetPnL:          NetPnL(c.realizedPnL, c.unrealizedPnL),
		ProfitFactor:    ProfitFactor(c.grossProfit, c.grossLoss),
		MaxDrawdown:     c.drawdown.MaxDrawdown,
		CurrentDrawdown: c.drawdown.CurrentDrawdown,
		AverageTradePnL: AverageTradePnL(c.tradePnLTotal, c.totalTrades),
		SharpeRatio:     SharpeRatio(),
		Trades:          trades,
		EquityCurve:     curve,
	}
}

func snapshotToEvent(snapshot PerformanceSnapshot) PerformanceUpdated {
	return PerformanceUpdated{
		TotalTrades:   snapshot.TotalTrades,
		WinRate:       snapshot.WinRate,
		RealizedPnL:   snapshot.RealizedPnL,
		UnrealizedPnL: snapshot.UnrealizedPnL,
		Drawdown:      snapshot.CurrentDrawdown,
		Timestamp:     snapshot.EquityCurve[len(snapshot.EquityCurve)-1].Timestamp,
	}
}

// Snapshot returns an immutable copy of the current performance state.
func (c *Cache) Snapshot() PerformanceSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.equityCurve) == 0 {
		return PerformanceSnapshot{SharpeRatio: SharpeRatio()}
	}
	last := c.equityCurve[len(c.equityCurve)-1].Timestamp
	return c.buildSnapshot(last)
}

func (c *Cache) tradesProcessed() int {
	return c.totalTrades
}

func (c *Cache) snapshotsGenerated() int {
	return len(c.equityCurve)
}

func (c *Cache) totalPnL() float64 {
	return NetPnL(c.realizedPnL, c.unrealizedPnL)
}

func (c *Cache) currentDrawdown() float64 {
	return c.drawdown.CurrentDrawdown
}
