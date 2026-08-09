package quality

import (
	"fmt"
	"time"
)

// Config controls the recommendation quality engine.
type Config struct {
	Enabled                bool
	SubscriberBuffer       int
	TrackingTimeoutMinutes int
	ExcellentThreshold     float64
	GoodThreshold          float64
	AverageThreshold       float64
	SuccessReturnPct       float64
	FailureReturnPct       float64
}

// WithDefaults returns a copy with production defaults applied for unset fields.
func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.TrackingTimeoutMinutes <= 0 {
		out.TrackingTimeoutMinutes = 120
	}
	if out.ExcellentThreshold <= 0 {
		out.ExcellentThreshold = 0.90
	}
	if out.GoodThreshold <= 0 {
		out.GoodThreshold = 0.75
	}
	if out.AverageThreshold <= 0 {
		out.AverageThreshold = 0.50
	}
	if out.SuccessReturnPct == 0 {
		out.SuccessReturnPct = 0.005
	}
	if out.FailureReturnPct == 0 {
		out.FailureReturnPct = -0.005
	}
	return out
}

// TrackingTimeout returns the configured tracking timeout duration.
func (c Config) TrackingTimeout() time.Duration {
	return time.Duration(c.TrackingTimeoutMinutes) * time.Minute
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("quality: subscriber_buffer must be >= 1")
	}
	if c.TrackingTimeoutMinutes < 1 {
		return fmt.Errorf("quality: tracking_timeout_minutes must be >= 1")
	}
	if c.ExcellentThreshold <= c.GoodThreshold || c.GoodThreshold <= c.AverageThreshold {
		return fmt.Errorf("quality: classification thresholds must satisfy excellent > good > average")
	}
	if c.SuccessReturnPct <= c.FailureReturnPct {
		return fmt.Errorf("quality: success_return_pct must be greater than failure_return_pct")
	}
	return nil
}
