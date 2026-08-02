package indicators

// SMA computes a simple moving average incrementally using a ring buffer.
type SMA struct {
	period int
	buf    []float64
	idx    int
	count  int
	sum    float64
}

// NewSMA creates an SMA calculator for the given period.
func NewSMA(period int) *SMA {
	return &SMA{
		period: period,
		buf:    make([]float64, period),
	}
}

// Update ingests a close price and returns the incremental result.
func (s *SMA) Update(close float64) Result {
	if s.count < s.period {
		s.buf[s.count] = close
		s.sum += close
		s.count++
		if s.count < s.period {
			return Result{Samples: s.count}
		}
		return Result{
			Value:    s.sum / float64(s.period),
			WarmedUp: true,
			Samples:  s.count,
		}
	}

	oldest := s.buf[s.idx]
	s.buf[s.idx] = close
	s.idx = (s.idx + 1) % s.period
	s.sum = s.sum - oldest + close
	s.count++

	return Result{
		Value:    s.sum / float64(s.period),
		WarmedUp: true,
		Samples:  s.count,
	}
}

// Value returns the latest SMA value and whether it is warmed up.
func (s *SMA) Value() (float64, bool) {
	if s.count < s.period {
		return 0, false
	}
	return s.sum / float64(s.period), true
}
