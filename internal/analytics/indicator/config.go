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
	RSI              []PeriodConfig
	ATR              []PeriodConfig
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
	if !c.hasIndicators() {
		return fmt.Errorf("indicator: at least one indicator period is required when enabled")
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
	for _, p := range c.RSI {
		if p.Period < 1 {
			return fmt.Errorf("indicator: rsi period must be >= 1")
		}
	}
	for _, p := range c.ATR {
		if p.Period < 1 {
			return fmt.Errorf("indicator: atr period must be >= 1")
		}
	}
	return nil
}

func (c Config) hasIndicators() bool {
	return len(c.EMA) > 0 || len(c.SMA) > 0 || len(c.RSI) > 0 || len(c.ATR) > 0
}

// EMAPeriods returns configured EMA periods.
func (c Config) EMAPeriods() []int {
	return periodList(c.EMA)
}

// SMAPeriods returns configured SMA periods.
func (c Config) SMAPeriods() []int {
	return periodList(c.SMA)
}

// RSIPeriods returns configured RSI periods.
func (c Config) RSIPeriods() []int {
	return periodList(c.RSI)
}

// ATRPeriods returns configured ATR periods.
func (c Config) ATRPeriods() []int {
	return periodList(c.ATR)
}

func periodList(cfg []PeriodConfig) []int {
	out := make([]int, 0, len(cfg))
	for _, p := range cfg {
		out = append(out, p.Period)
	}
	return out
}
