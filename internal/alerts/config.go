package alerts

import (
	"fmt"
	"time"
)

// Config controls the alert engine.
type Config struct {
	Enabled                   bool
	SubscriberBuffer          int
	ConfidenceChangeThreshold float64
	CooldownSeconds           int
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.ConfidenceChangeThreshold <= 0 {
		out.ConfidenceChangeThreshold = 0.05
	}
	if out.CooldownSeconds <= 0 {
		out.CooldownSeconds = 300
	}
	return out
}

// Cooldown returns the configured deduplication cooldown duration.
func (c Config) Cooldown() time.Duration {
	return time.Duration(c.CooldownSeconds) * time.Second
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("alerts: subscriber_buffer must be >= 1")
	}
	if c.ConfidenceChangeThreshold <= 0 || c.ConfidenceChangeThreshold > 1 {
		return fmt.Errorf("alerts: confidence_change_threshold must be in (0, 1]")
	}
	if c.CooldownSeconds < 1 {
		return fmt.Errorf("alerts: cooldown_seconds must be >= 1")
	}
	return nil
}
