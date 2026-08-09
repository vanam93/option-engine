package indicators

// DonchianChannel computes the highest high and lowest low over a lookback window.
type DonchianChannel struct {
	period int
	highs  []float64
	lows   []float64
	idx    int
	count  int
	upper  float64
	lower  float64
	warmed bool
}

// DonchianResult is the output of an incremental Donchian channel update.
type DonchianResult struct {
	Upper    float64
	Lower    float64
	WarmedUp bool
	Samples  int
}

// NewDonchianChannel creates a Donchian channel calculator for the given period.
func NewDonchianChannel(period int) *DonchianChannel {
	return &DonchianChannel{
		period: period,
		highs:  make([]float64, period),
		lows:   make([]float64, period),
	}
}

// Update ingests high and low and returns channel boundaries.
func (d *DonchianChannel) Update(high, low float64) DonchianResult {
	if d.count < d.period {
		d.highs[d.count] = high
		d.lows[d.count] = low
		d.count++
		if d.count < d.period {
			return DonchianResult{Samples: d.count}
		}
		return d.compute()
	}

	d.highs[d.idx] = high
	d.lows[d.idx] = low
	d.idx = (d.idx + 1) % d.period
	d.count++
	return d.compute()
}

func (d *DonchianChannel) compute() DonchianResult {
	d.upper = d.highs[0]
	d.lower = d.lows[0]
	for i := 1; i < d.period; i++ {
		if d.highs[i] > d.upper {
			d.upper = d.highs[i]
		}
		if d.lows[i] < d.lower {
			d.lower = d.lows[i]
		}
	}
	d.warmed = true
	return DonchianResult{
		Upper:    d.upper,
		Lower:    d.lower,
		WarmedUp: true,
		Samples:  d.count,
	}
}

// Value returns upper, lower, and warmed state.
func (d *DonchianChannel) Value() (upper, lower float64, warmed bool) {
	return d.upper, d.lower, d.warmed
}

// ChannelFromSlice computes Donchian boundaries from a candle slice (non-incremental).
func ChannelFromSlice(highs, lows []float64, period int) (upper, lower float64, ok bool) {
	if period <= 0 || len(highs) < period || len(lows) < period {
		return 0, 0, false
	}
	start := len(highs) - period
	upper = highs[start]
	lower = lows[start]
	for i := start + 1; i < len(highs); i++ {
		if highs[i] > upper {
			upper = highs[i]
		}
		if lows[i] < lower {
			lower = lows[i]
		}
	}
	return upper, lower, true
}
