package performance

import "fmt"

// Config controls the performance analytics engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("performance: subscriber_buffer must be >= 1")
	}
	return nil
}
