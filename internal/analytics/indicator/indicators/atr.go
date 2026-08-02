package indicators

import "math"

// ATR computes Wilder's average true range incrementally.
type ATR struct {
	period int

	prevClose float64
	hasPrev   bool
	sumTR     float64
	trCount   int
	value     float64
	count     int
	warmed    bool
}

// NewATR creates an ATR calculator for the given period.
func NewATR(period int) *ATR {
	return &ATR{period: period}
}

// Update ingests OHLC values and returns the incremental result.
func (a *ATR) Update(high, low, close float64) Result {
	a.count++
	tr := a.trueRange(high, low, close)
	a.prevClose = close
	a.hasPrev = true

	if !a.warmed {
		a.sumTR += tr
		a.trCount++
		if a.trCount < a.period {
			return Result{Samples: a.count}
		}
		a.value = a.sumTR / float64(a.period)
		a.warmed = true
		return Result{Value: a.value, WarmedUp: true, Samples: a.count}
	}

	a.value = (a.value*float64(a.period-1) + tr) / float64(a.period)
	return Result{Value: a.value, WarmedUp: true, Samples: a.count}
}

func (a *ATR) trueRange(high, low, close float64) float64 {
	if !a.hasPrev {
		return high - low
	}
	return math.Max(high-low, math.Max(math.Abs(high-a.prevClose), math.Abs(low-a.prevClose)))
}

// Value returns the latest ATR and whether it is warmed up.
func (a *ATR) Value() (float64, bool) {
	return a.value, a.warmed
}
