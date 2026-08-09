package laboratory

import "fmt"

// Config controls the strategy laboratory orchestrator.
type Config struct {
	Enabled           bool
	AutoVersion       bool
	ConcurrentStudies int
}

func (c Config) WithDefaults() Config {
	out := c
	if out.ConcurrentStudies <= 0 {
		out.ConcurrentStudies = 1
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.ConcurrentStudies < 1 {
		return fmt.Errorf("laboratory: concurrent_studies must be >= 1")
	}
	return nil
}
