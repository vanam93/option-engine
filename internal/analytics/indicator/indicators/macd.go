package indicators

// MACD computes the moving average convergence divergence incrementally.
type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int

	fastEMA   *EMA
	slowEMA   *EMA
	signalEMA *EMA

	macdLine   float64
	signalLine float64
	histogram  float64
	count      int
	warmed     bool
}

// MACDResult is the output of an incremental MACD update.
type MACDResult struct {
	MACD      float64
	Signal    float64
	Histogram float64
	WarmedUp  bool
	Samples   int
}

// NewMACD creates a MACD calculator with fast, slow, and signal EMA periods.
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD {
	return &MACD{
		fastPeriod:   fastPeriod,
		slowPeriod:   slowPeriod,
		signalPeriod: signalPeriod,
		fastEMA:      NewEMA(fastPeriod),
		slowEMA:      NewEMA(slowPeriod),
		signalEMA:    NewEMA(signalPeriod),
	}
}

// Update ingests a close price and returns the incremental MACD result.
func (m *MACD) Update(close float64) MACDResult {
	m.count++

	fastResult := m.fastEMA.Update(close)
	slowResult := m.slowEMA.Update(close)
	if !slowResult.WarmedUp {
		return MACDResult{Samples: m.count}
	}

	m.macdLine = fastResult.Value - slowResult.Value
	signalResult := m.signalEMA.Update(m.macdLine)
	if !signalResult.WarmedUp {
		return MACDResult{Samples: m.count}
	}

	m.signalLine = signalResult.Value
	m.histogram = m.macdLine - m.signalLine
	m.warmed = true
	return MACDResult{
		MACD:      m.macdLine,
		Signal:    m.signalLine,
		Histogram: m.histogram,
		WarmedUp:  true,
		Samples:   m.count,
	}
}

// Value returns MACD line, signal line, histogram, and whether the indicator is warmed up.
func (m *MACD) Value() (macd, signal, histogram float64, warmed bool) {
	return m.macdLine, m.signalLine, m.histogram, m.warmed
}
