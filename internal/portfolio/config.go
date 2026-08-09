package portfolio

import "fmt"

// Config controls the portfolio and PnL engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
}

func (c Config) WithDefaults() Config {
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
		return fmt.Errorf("portfolio: subscriber_buffer must be >= 1")
	}
	return nil
}
