package validation

import "fmt"

// Config controls the recommendation validation engine.
type Config struct {
	Enabled              bool
	SubscriberBuffer     int
	MinConfidence        float64
	MinOptimizationScore float64
	MinWalkforwardScore  float64
	MinMonteCarloScore   float64
	MinWinRate           float64
	MaxDrawdown          float64
	FreshnessSeconds     int
	SuppressDuplicates   bool
	ReplayMode           bool
}

func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.MinConfidence <= 0 {
		out.MinConfidence = 0.70
	}
	if out.MinOptimizationScore <= 0 {
		out.MinOptimizationScore = 0.60
	}
	if out.MinWalkforwardScore <= 0 {
		out.MinWalkforwardScore = 0.60
	}
	if out.MinMonteCarloScore <= 0 {
		out.MinMonteCarloScore = 0.60
	}
	if out.MinWinRate <= 0 {
		out.MinWinRate = 0.50
	}
	if out.MaxDrawdown <= 0 {
		out.MaxDrawdown = 0.20
	}
	if out.FreshnessSeconds <= 0 {
		out.FreshnessSeconds = 300
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("validation: subscriber_buffer must be >= 1")
	}
	if c.MinConfidence <= 0 || c.MinConfidence > 1 {
		return fmt.Errorf("validation: min_confidence must be in (0, 1]")
	}
	if c.MaxDrawdown <= 0 || c.MaxDrawdown > 1 {
		return fmt.Errorf("validation: max_drawdown must be in (0, 1]")
	}
	if c.FreshnessSeconds < 1 {
		return fmt.Errorf("validation: freshness_seconds must be >= 1")
	}
	return nil
}
