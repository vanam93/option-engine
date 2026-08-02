package indicator

import "fmt"
// PeriodConfig configures a single indicator lookback window.
type PeriodConfig struct {
	Period int `mapstructure:"period" yaml:"period"`
}

// Config controls the indicator engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	EMA              []PeriodConfig
	SMA              []PeriodConfig
}

func (c Config) withDefaults() Config {
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
		return fmt.Errorf("indicator: subscriber_buffer must be >= 1")
	}
	if len(c.EMA) == 0 && len(c.SMA) == 0 {
		return fmt.Errorf("indicator: at least one EMA or SMA period is required when enabled")
	}
	for _, p := range c.EMA {
		if p.Period < 1 {
			return fmt.Errorf("indicator: ema period must be >= 1")
		}
	}
	for _, p := range c.SMA {
		if p.Period < 1 {
			return fmt.Errorf("indicator: sma period must be >= 1")
		}
	}
	return nil
}

// EMAPeriods returns configured EMA periods.
func (c Config) EMAPeriods() []int {
	out := make([]int, 0, len(c.EMA))
	for _, p := range c.EMA {
		out = append(out, p.Period)
	}
	return out
}

// SMAPeriods returns configured SMA periods.
func (c Config) SMAPeriods() []int {
	out := make([]int, 0, len(c.SMA))
	for _, p := range c.SMA {
		out = append(out, p.Period)
	}
	return out
}
