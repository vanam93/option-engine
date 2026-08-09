package researchengine

// SimulationMetrics captures simulator signal and position accounting.
type SimulationMetrics struct {
	BuySignals       int
	SellSignals      int
	ExitSignals      int
	OpensLong        int
	OpensShort       int
	Closes           int
	IgnoredBuyLong   int
	IgnoredSellShort int
	IgnoredExitFlat  int
	EndOfDataCloses  int
}

// Reset clears all counters.
func (m *SimulationMetrics) Reset() {
	if m == nil {
		return
	}
	*m = SimulationMetrics{}
}
