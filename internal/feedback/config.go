package feedback

import (
	"fmt"
	"sort"
)

// Config controls the recommendation feedback engine.
type Config struct {
	Enabled           bool
	SubscriberBuffer  int
	RollingWindows    []int
	ConfidenceBuckets []float64
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if len(out.RollingWindows) == 0 {
		out.RollingWindows = []int{25, 50, 100, 250}
	}
	if len(out.ConfidenceBuckets) == 0 {
		out.ConfidenceBuckets = []float64{0.60, 0.70, 0.80, 0.90, 0.95}
	}
	sort.Ints(out.RollingWindows)
	sort.Float64s(out.ConfidenceBuckets)
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("feedback: subscriber_buffer must be >= 1")
	}
	for _, w := range c.RollingWindows {
		if w < 1 {
			return fmt.Errorf("feedback: rolling window sizes must be >= 1")
		}
	}
	for i, b := range c.ConfidenceBuckets {
		if b <= 0 || b > 1 {
			return fmt.Errorf("feedback: confidence bucket thresholds must be in (0, 1]")
		}
		if i > 0 && b <= c.ConfidenceBuckets[i-1] {
			return fmt.Errorf("feedback: confidence bucket thresholds must be strictly increasing")
		}
	}
	return nil
}
