package console

import (
	"fmt"
	"time"
)

// Config controls the recommendation console renderer.
type Config struct {
	Enabled          bool
	RefreshInterval  time.Duration
	SubscriberBuffer int
}

func (c Config) WithDefaults() Config {
	out := c
	if out.RefreshInterval <= 0 {
		out.RefreshInterval = time.Second
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 512
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.RefreshInterval < time.Millisecond {
		return fmt.Errorf("console: refresh_interval must be >= 1ms")
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("console: subscriber_buffer must be >= 1")
	}
	return nil
}
