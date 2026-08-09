package scanner

import "fmt"

// ScannersConfig toggles individual scanner rules.
type ScannersConfig struct {
	EMA     bool `mapstructure:"ema"`
	RSI     bool `mapstructure:"rsi"`
	MACD    bool `mapstructure:"macd"`
	Trend   bool `mapstructure:"trend"`
	Ranking bool `mapstructure:"ranking"`
}

// Config controls the market scanner engine.
type Config struct {
	Enabled          bool
	Symbols          []string
	Scanners         ScannersConfig
	SubscriberBuffer int
	MinConfidence    float64
}

func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.MinConfidence <= 0 {
		out.MinConfidence = 0.5
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("scanner: subscriber_buffer must be >= 1")
	}
	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return fmt.Errorf("scanner: min_confidence must be between 0 and 1")
	}
	return nil
}

// EnabledScannerCount returns the number of active scanner rules.
func (c Config) EnabledScannerCount() int {
	count := 0
	if c.Scanners.EMA {
		count++
	}
	if c.Scanners.RSI {
		count++
	}
	if c.Scanners.MACD {
		count++
	}
	if c.Scanners.Trend {
		count++
	}
	if c.Scanners.Ranking {
		count++
	}
	return count
}

// WatchesSymbol reports whether the scanner should process the given symbol.
func (c Config) WatchesSymbol(symbol string) bool {
	if len(c.Symbols) == 0 {
		return true
	}
	for _, s := range c.Symbols {
		if s == symbol {
			return true
		}
	}
	return false
}
