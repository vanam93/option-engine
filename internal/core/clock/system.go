package clock

import "time"

// SystemClock uses the real system clock.
type SystemClock struct{}

// NewSystem returns a production clock backed by time.Now().
func NewSystem() Clock {
	return SystemClock{}
}

func (SystemClock) Now() time.Time                       { return time.Now().UTC() }
func (SystemClock) Since(t time.Time) time.Duration      { return time.Since(t) }
func (SystemClock) Until(t time.Time) time.Duration      { return time.Until(t) }
