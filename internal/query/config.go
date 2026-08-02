package query

import "fmt"

// Config controls the research query API.
type Config struct {
	Enabled     bool
	APIPrefix   string
	DefaultLimit int
	MaxLimit     int
}

func (c Config) withDefaults() Config {
	out := c
	if out.APIPrefix == "" {
		out.APIPrefix = "/api/v1"
	}
	if out.DefaultLimit <= 0 {
		out.DefaultLimit = 50
	}
	if out.MaxLimit <= 0 {
		out.MaxLimit = 500
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	out := c.withDefaults()
	if out.DefaultLimit < 1 {
		return fmt.Errorf("query: default_limit must be >= 1")
	}
	if out.MaxLimit < out.DefaultLimit {
		return fmt.Errorf("query: max_limit must be >= default_limit")
	}
	return nil
}
