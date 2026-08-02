package recommendationstate

import "fmt"

// Config controls the recommendation state manager.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	MaxActive        int
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.MaxActive <= 0 {
		out.MaxActive = 10000
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("recommendationstate: subscriber_buffer must be >= 1")
	}
	if c.MaxActive < 1 {
		return fmt.Errorf("recommendationstate: max_active must be >= 1")
	}
	return nil
}
