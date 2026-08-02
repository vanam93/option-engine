package indicators

import "math"

// Bollinger computes Bollinger Bands incrementally.
type Bollinger struct {
	period int
	stddev float64

	buf   []float64
	idx   int
	count int
	sum   float64
	sumSq float64

	upper     float64
	middle    float64
	lower     float64
	bandWidth float64
	percentB  float64
	warmed    bool
}

// BollingerResult is the output of an incremental Bollinger Bands update.
type BollingerResult struct {
	Upper     float64
	Middle    float64
	Lower     float64
	BandWidth float64
	PercentB  float64
	WarmedUp  bool
	Samples   int
}

// NewBollinger creates a Bollinger Bands calculator for the given period and stddev multiplier.
func NewBollinger(period int, stddev float64) *Bollinger {
	return &Bollinger{
		period: period,
		stddev: stddev,
		buf:    make([]float64, period),
	}
}

// Update ingests a close price and returns the incremental Bollinger result.
func (b *Bollinger) Update(close float64) BollingerResult {
	if b.count < b.period {
		b.buf[b.count] = close
		b.sum += close
		b.sumSq += close * close
		b.count++
		if b.count < b.period {
			return BollingerResult{Samples: b.count}
		}
		return b.compute(close)
	}

	oldest := b.buf[b.idx]
	b.buf[b.idx] = close
	b.idx = (b.idx + 1) % b.period
	b.sum = b.sum - oldest + close
	b.sumSq = b.sumSq - oldest*oldest + close*close
	b.count++
	return b.compute(close)
}

func (b *Bollinger) compute(close float64) BollingerResult {
	n := float64(b.period)
	b.middle = b.sum / n
	variance := (b.sumSq - (b.sum*b.sum)/n) / n
	if variance < 0 {
		variance = 0
	}
	sigma := math.Sqrt(variance)

	b.upper = b.middle + b.stddev*sigma
	b.lower = b.middle - b.stddev*sigma

	span := b.upper - b.lower
	if b.middle != 0 {
		b.bandWidth = span / b.middle
	}
	if span != 0 {
		b.percentB = (close - b.lower) / span
	}

	b.warmed = true
	return BollingerResult{
		Upper:     b.upper,
		Middle:    b.middle,
		Lower:     b.lower,
		BandWidth: b.bandWidth,
		PercentB:  b.percentB,
		WarmedUp:  true,
		Samples:   b.count,
	}
}

// Value returns band values and whether the indicator is warmed up.
func (b *Bollinger) Value() (upper, middle, lower, bandWidth, percentB float64, warmed bool) {
	return b.upper, b.middle, b.lower, b.bandWidth, b.percentB, b.warmed
}
