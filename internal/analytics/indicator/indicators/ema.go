package indicators

// EMA computes an exponential moving average incrementally.
// The first warmed value uses an SMA seed over the lookback window.
type EMA struct {
	period int
	alpha  float64

	sum    float64
	value  float64
	count  int
	warmed bool
}

// NewEMA creates an EMA calculator for the given period.
func NewEMA(period int) *EMA {
	return &EMA{
		period: period,
		alpha:  2.0 / float64(period+1),
	}
}

// Update ingests a close price and returns the incremental result.
func (e *EMA) Update(close float64) Result {
	e.count++
	if !e.warmed {
		e.sum += close
		if e.count < e.period {
			return Result{Samples: e.count}
		}
		e.value = e.sum / float64(e.period)
		e.warmed = true
		return Result{Value: e.value, WarmedUp: true, Samples: e.count}
	}

	e.value = e.alpha*close + (1-e.alpha)*e.value
	return Result{Value: e.value, WarmedUp: true, Samples: e.count}
}

// Value returns the latest EMA value and whether it is warmed up.
func (e *EMA) Value() (float64, bool) {
	return e.value, e.warmed
}
