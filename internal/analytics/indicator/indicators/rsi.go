package indicators

// RSI computes Wilder's relative strength index incrementally.
type RSI struct {
	period int

	prevClose   float64
	hasPrev     bool
	sumGain     float64
	sumLoss     float64
	changeCount int
	avgGain     float64
	avgLoss     float64
	value       float64
	count       int
	warmed      bool
}

// NewRSI creates an RSI calculator for the given period.
func NewRSI(period int) *RSI {
	return &RSI{period: period}
}

// Update ingests a close price and returns the incremental result.
func (r *RSI) Update(close float64) Result {
	r.count++
	if !r.hasPrev {
		r.prevClose = close
		r.hasPrev = true
		return Result{Samples: r.count}
	}

	change := close - r.prevClose
	r.prevClose = close

	gain := 0.0
	loss := 0.0
	if change > 0 {
		gain = change
	} else if change < 0 {
		loss = -change
	}

	if !r.warmed {
		r.sumGain += gain
		r.sumLoss += loss
		r.changeCount++
		if r.changeCount < r.period {
			return Result{Samples: r.count}
		}
		r.avgGain = r.sumGain / float64(r.period)
		r.avgLoss = r.sumLoss / float64(r.period)
		r.value = computeRSI(r.avgGain, r.avgLoss)
		r.warmed = true
		return Result{Value: r.value, WarmedUp: true, Samples: r.count}
	}

	r.avgGain = (r.avgGain*float64(r.period-1) + gain) / float64(r.period)
	r.avgLoss = (r.avgLoss*float64(r.period-1) + loss) / float64(r.period)
	r.value = computeRSI(r.avgGain, r.avgLoss)
	return Result{Value: r.value, WarmedUp: true, Samples: r.count}
}

func computeRSI(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50
		}
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// Value returns the latest RSI and whether it is warmed up.
func (r *RSI) Value() (float64, bool) {
	return r.value, r.warmed
}
