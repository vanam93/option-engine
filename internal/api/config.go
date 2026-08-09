package api

import (
	"fmt"
	"time"
)

// Config controls the Intelligence API layer.
type Config struct {
	Enabled      bool
	Prefix       string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DefaultLimit int
	MaxLimit     int
}

func (c Config) WithDefaults() Config {
	out := c
	if out.Prefix == "" {
		out.Prefix = "/api/v1"
	}
	if out.ReadTimeout <= 0 {
		out.ReadTimeout = 30 * time.Second
	}
	if out.WriteTimeout <= 0 {
		out.WriteTimeout = 30 * time.Second
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
	out := c.WithDefaults()
	if out.DefaultLimit < 1 {
		return fmt.Errorf("api: default_limit must be >= 1")
	}
	if out.MaxLimit < out.DefaultLimit {
		return fmt.Errorf("api: max_limit must be >= default_limit")
	}
	return nil
}
