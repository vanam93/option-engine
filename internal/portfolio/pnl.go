package portfolio

// RealizedPnL computes closed-trade profit using (exit_price - entry_price) * quantity.
// For short positions quantity is treated as negative so the formula remains consistent.
func RealizedPnL(side Side, entryPrice, exitPrice float64, quantity int) float64 {
	qty := signedQuantity(side, quantity)
	return (exitPrice - entryPrice) * qty
}

// UnrealizedPnL computes mark-to-market profit using (current_price - average_price) * quantity.
func UnrealizedPnL(side Side, averagePrice, currentPrice float64, quantity int) float64 {
	qty := signedQuantity(side, quantity)
	return (currentPrice - averagePrice) * qty
}

// Exposure returns absolute notional exposure for an open position.
func Exposure(side Side, price float64, quantity int) float64 {
	if quantity <= 0 || price <= 0 {
		return 0
	}
	return price * float64(quantity)
}

func signedQuantity(side Side, quantity int) float64 {
	if quantity <= 0 {
		return 0
	}
	if side == SideShort {
		return -float64(quantity)
	}
	return float64(quantity)
}
