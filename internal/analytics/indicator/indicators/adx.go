package indicators

import "math"

// ADX computes Wilder's ADX with directional indicators incrementally.
type ADX struct {
	period int

	prevHigh  float64
	prevLow   float64
	prevClose float64
	hasPrev   bool

	sumTR       float64
	sumPlusDM   float64
	sumMinusDM  float64
	smoothTR    float64
	smoothPlus  float64
	smoothMinus float64
	smoothDX    float64
	dxSamples   int

	plusDI  float64
	minusDI float64
	value   float64
	count   int
	warmed  bool
}

// ADXResult is the output of an incremental ADX update.
type ADXResult struct {
	ADX      float64
	PlusDI   float64
	MinusDI  float64
	WarmedUp bool
	Samples  int
}

// NewADX creates an ADX calculator for the given period.
func NewADX(period int) *ADX {
	return &ADX{period: period}
}

// Update ingests OHLC values and returns the incremental ADX result.
func (a *ADX) Update(high, low, close float64) ADXResult {
	a.count++
	if !a.hasPrev {
		a.prevHigh = high
		a.prevLow = low
		a.prevClose = close
		a.hasPrev = true
		return ADXResult{Samples: a.count}
	}

	upMove := high - a.prevHigh
	downMove := a.prevLow - low
	plusDM := 0.0
	minusDM := 0.0
	if upMove > downMove && upMove > 0 {
		plusDM = upMove
	}
	if downMove > upMove && downMove > 0 {
		minusDM = downMove
	}

	tr := math.Max(high-low, math.Max(math.Abs(high-a.prevClose), math.Abs(low-a.prevClose)))
	a.prevHigh = high
	a.prevLow = low
	a.prevClose = close

	if !a.warmed {
		a.sumTR += tr
		a.sumPlusDM += plusDM
		a.sumMinusDM += minusDM
		if a.count-1 < a.period {
			return ADXResult{Samples: a.count}
		}

		if a.smoothTR == 0 {
			a.smoothTR = a.sumTR
			a.smoothPlus = a.sumPlusDM
			a.smoothMinus = a.sumMinusDM
		} else {
			a.smoothTR = a.smoothTR - a.smoothTR/float64(a.period) + tr
			a.smoothPlus = a.smoothPlus - a.smoothPlus/float64(a.period) + plusDM
			a.smoothMinus = a.smoothMinus - a.smoothMinus/float64(a.period) + minusDM
		}
	} else {
		a.smoothTR = a.smoothTR - a.smoothTR/float64(a.period) + tr
		a.smoothPlus = a.smoothPlus - a.smoothPlus/float64(a.period) + plusDM
		a.smoothMinus = a.smoothMinus - a.smoothMinus/float64(a.period) + minusDM
	}

	if a.smoothTR == 0 {
		return ADXResult{Samples: a.count}
	}

	a.plusDI = 100 * a.smoothPlus / a.smoothTR
	a.minusDI = 100 * a.smoothMinus / a.smoothTR
	diSum := a.plusDI + a.minusDI
	if diSum == 0 {
		return ADXResult{Samples: a.count}
	}
	dx := 100 * math.Abs(a.plusDI-a.minusDI) / diSum

	if !a.warmed {
		a.smoothDX += dx
		a.dxSamples++
		if a.dxSamples < a.period {
			return ADXResult{
				PlusDI:  a.plusDI,
				MinusDI: a.minusDI,
				Samples: a.count,
			}
		}
		a.value = a.smoothDX / float64(a.period)
		a.warmed = true
	} else {
		a.value = (a.value*float64(a.period-1) + dx) / float64(a.period)
	}

	return ADXResult{
		ADX:      a.value,
		PlusDI:   a.plusDI,
		MinusDI:  a.minusDI,
		WarmedUp: a.warmed,
		Samples:  a.count,
	}
}

// Value returns ADX, +DI, -DI, and warmed state.
func (a *ADX) Value() (adx, plusDI, minusDI float64, warmed bool) {
	return a.value, a.plusDI, a.minusDI, a.warmed
}
