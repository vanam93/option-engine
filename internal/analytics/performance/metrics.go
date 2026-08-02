package performance

// WinRate returns the fraction of winning trades.
func WinRate(winning, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(winning) / float64(total)
}

// NetPnL combines realized and unrealized profit.
func NetPnL(realized, unrealized float64) float64 {
	return realized + unrealized
}

// ProfitFactor returns gross profit divided by gross loss.
func ProfitFactor(grossProfit, grossLoss float64) float64 {
	if grossLoss <= 0 {
		if grossProfit > 0 {
			return 0
		}
		return 0
	}
	return grossProfit / grossLoss
}

// AverageTradePnL returns the mean PnL across completed trades.
func AverageTradePnL(totalPnL float64, totalTrades int) float64 {
	if totalTrades == 0 {
		return 0
	}
	return totalPnL / float64(totalTrades)
}

// MaxDrawdown returns the largest peak-to-trough equity decline.
func MaxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0
	}
	peak := equity[0]
	maxDD := 0.0
	for _, value := range equity {
		if value > peak {
			peak = value
		}
		if decline := peak - value; decline > maxDD {
			maxDD = decline
		}
	}
	return maxDD
}

// DrawdownState tracks running peak equity and drawdown values.
type DrawdownState struct {
	PeakEquity      float64
	MaxDrawdown     float64
	CurrentDrawdown float64
}

// UpdateDrawdown advances drawdown state with a new equity value.
func (d *DrawdownState) Update(equity float64) {
	if equity > d.PeakEquity {
		d.PeakEquity = equity
		d.CurrentDrawdown = 0
		return
	}
	d.CurrentDrawdown = d.PeakEquity - equity
	if d.CurrentDrawdown > d.MaxDrawdown {
		d.MaxDrawdown = d.CurrentDrawdown
	}
}

// SharpeRatio is a placeholder until return volatility data is available.
func SharpeRatio() float64 {
	return 0
}
