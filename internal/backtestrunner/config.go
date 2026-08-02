package backtestrunner

import "fmt"

// Config controls the historical backtest runner orchestrator.
type Config struct {
	Enabled            bool
	AutoStart          bool
	ConcurrentSessions int
	SubscriberBuffer   int
}

func (c Config) withDefaults() Config {
	out := c
	if out.ConcurrentSessions <= 0 {
		out.ConcurrentSessions = 1
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
	if c.ConcurrentSessions < 1 {
		return fmt.Errorf("backtestrunner: concurrent_sessions must be >= 1")
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("backtestrunner: subscriber_buffer must be >= 1")
	}
	return nil
}
