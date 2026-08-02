package strategy

import "fmt"

const (
	strategyTrendFollowing = "trend_following"
	strategyMeanReversion  = "mean_reversion"
	strategyBreakout       = "breakout"
)

// TrendFollowingConfig enables the EMA + MACD confirmation strategy.
type TrendFollowingConfig struct {
	Enabled bool
}

// MeanReversionConfig enables the RSI + Bollinger confirmation strategy.
type MeanReversionConfig struct {
	Enabled bool
}

// BreakoutConfig is a placeholder for a future breakout strategy.
type BreakoutConfig struct {
	Enabled bool
}

// Config controls the strategy engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	MinConfidence    float64
	TrendFollowing   TrendFollowingConfig
	MeanReversion    MeanReversionConfig
	Breakout         BreakoutConfig
}

func (c Config) withDefaults() Config {
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
		return fmt.Errorf("strategy: subscriber_buffer must be >= 1")
	}
	if !c.hasActiveStrategies() {
		return fmt.Errorf("strategy: at least one strategy must be enabled when engine is enabled")
	}
	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return fmt.Errorf("strategy: min_confidence must be between 0 and 1")
	}
	return nil
}

func (c Config) hasActiveStrategies() bool {
	return c.TrendFollowing.Enabled || c.MeanReversion.Enabled || c.Breakout.Enabled
}

// EnabledStrategies returns the names of enabled strategies.
func (c Config) EnabledStrategies() []string {
	var out []string
	if c.TrendFollowing.Enabled {
		out = append(out, strategyTrendFollowing)
	}
	if c.MeanReversion.Enabled {
		out = append(out, strategyMeanReversion)
	}
	if c.Breakout.Enabled {
		out = append(out, strategyBreakout)
	}
	return out
}
