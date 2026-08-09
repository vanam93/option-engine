package indicators

// SuperTrend computes ATR-based trend bands and direction incrementally.
type SuperTrend struct {
	atrPeriod    int
	multiplier   float64
	atr          *ATR
	prevClose    float64
	hasPrevClose bool
	finalUpper   float64
	finalLower   float64
	value        float64
	direction    int // 1 bullish, -1 bearish
	count        int
	warmed       bool
}

// SuperTrendResult is the output of an incremental SuperTrend update.
type SuperTrendResult struct {
	Value     float64
	Direction int
	Upper     float64
	Lower     float64
	WarmedUp  bool
	Samples   int
}

// NewSuperTrend creates a SuperTrend calculator with ATR period and multiplier.
func NewSuperTrend(atrPeriod int, multiplier float64) *SuperTrend {
	return &SuperTrend{
		atrPeriod:  atrPeriod,
		multiplier: multiplier,
		atr:        NewATR(atrPeriod),
	}
}

// Update ingests OHLC values and returns the incremental SuperTrend result.
func (s *SuperTrend) Update(high, low, close float64) SuperTrendResult {
	s.count++
	atrResult := s.atr.Update(high, low, close)
	if !atrResult.WarmedUp {
		s.prevClose = close
		s.hasPrevClose = true
		return SuperTrendResult{Samples: s.count}
	}

	hl2 := (high + low) / 2.0
	basicUpper := hl2 + s.multiplier*atrResult.Value
	basicLower := hl2 - s.multiplier*atrResult.Value

	finalUpper := basicUpper
	finalLower := basicLower
	if s.hasPrevClose {
		if basicUpper < s.finalUpper || s.prevClose > s.finalUpper {
			finalUpper = basicUpper
		} else {
			finalUpper = s.finalUpper
		}
		if basicLower > s.finalLower || s.prevClose < s.finalLower {
			finalLower = basicLower
		} else {
			finalLower = s.finalLower
		}
	}

	direction := s.direction
	value := s.value
	if s.warmed {
		if s.value == s.finalUpper {
			if close > finalUpper {
				direction = 1
				value = finalLower
			} else {
				direction = -1
				value = finalUpper
			}
		} else {
			if close < finalLower {
				direction = -1
				value = finalUpper
			} else {
				direction = 1
				value = finalLower
			}
		}
	} else {
		if close <= finalUpper {
			direction = -1
			value = finalUpper
		} else {
			direction = 1
			value = finalLower
		}
	}

	s.finalUpper = finalUpper
	s.finalLower = finalLower
	s.value = value
	s.direction = direction
	s.prevClose = close
	s.hasPrevClose = true
	s.warmed = true

	return SuperTrendResult{
		Value:     value,
		Direction: direction,
		Upper:     finalUpper,
		Lower:     finalLower,
		WarmedUp:  true,
		Samples:   s.count,
	}
}

// Value returns SuperTrend line, direction, and warmed state.
func (s *SuperTrend) Value() (value float64, direction int, warmed bool) {
	return s.value, s.direction, s.warmed
}
