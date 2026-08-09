package aicontext

import "fmt"

// Config controls the AI context builder engine.
type Config struct {
	Enabled          bool
	ExecutivePrompt  bool
	TechnicalPrompt  bool
	JSONPrompt       bool
	SubscriberBuffer int
}

func (c Config) WithDefaults() Config {
	out := c
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
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("aicontext: subscriber_buffer must be >= 1")
	}
	return nil
}
