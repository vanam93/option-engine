package researchengine

// Journal stores every completed simulated trade.
type Journal struct {
	Trades []SimulatedTrade
}

// NewJournal creates an empty trade journal.
func NewJournal() *Journal {
	return &Journal{Trades: make([]SimulatedTrade, 0)}
}

// Add appends a completed trade.
func (j *Journal) Add(trade SimulatedTrade) {
	j.Trades = append(j.Trades, trade)
}

// Len returns completed trade count.
func (j *Journal) Len() int {
	return len(j.Trades)
}

// All returns a copy of all trades.
func (j *Journal) All() []SimulatedTrade {
	return append([]SimulatedTrade(nil), j.Trades...)
}

// PnLSeries returns net profit per trade for statistics.
func (j *Journal) PnLSeries() []float64 {
	out := make([]float64, len(j.Trades))
	for i, t := range j.Trades {
		out[i] = t.NetProfit
	}
	return out
}
