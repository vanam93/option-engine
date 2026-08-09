package paper

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultPriceMarket = "market"

// Config controls the paper execution engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	SlippagePercent  float64
	DefaultPrice     string
}

func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.DefaultPrice == "" {
		out.DefaultPrice = defaultPriceMarket
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("paper execution: subscriber_buffer must be >= 1")
	}
	if c.SlippagePercent < 0 {
		return fmt.Errorf("paper execution: slippage_percent must be >= 0")
	}
	mode := strings.ToLower(strings.TrimSpace(c.DefaultPrice))
	if mode != defaultPriceMarket {
		if _, err := strconv.ParseFloat(mode, 64); err != nil {
			return fmt.Errorf("paper execution: default_price must be %q or a numeric value", defaultPriceMarket)
		}
	}
	return nil
}

func (c Config) defaultPriceValue() (float64, bool) {
	mode := strings.ToLower(strings.TrimSpace(c.DefaultPrice))
	if mode == defaultPriceMarket {
		return 0, false
	}
	v, err := strconv.ParseFloat(mode, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (c Config) requiresMarketPrice() bool {
	return strings.ToLower(strings.TrimSpace(c.DefaultPrice)) == defaultPriceMarket
}
