package indicator

import "fmt"

// PeriodConfig configures a single indicator lookback window.
type PeriodConfig struct {
	Period int `mapstructure:"period" yaml:"period"`
}

// MACDConfig configures MACD fast, slow, and signal EMA periods.
type MACDConfig struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
}

// BollingerConfig configures Bollinger Bands period and standard deviation multiplier.
type BollingerConfig struct {
	Period int
	StdDev float64
}

// Config controls the indicator engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	EMA              []PeriodConfig
	SMA              []PeriodConfig
	RSI              []PeriodConfig
	ATR              []PeriodConfig
	MACD             *MACDConfig
	Bollinger        *BollingerConfig
}

func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.MACD != nil {
		macd := *out.MACD
		if macd.FastPeriod <= 0 {
			macd.FastPeriod = 12
		}
		if macd.SlowPeriod <= 0 {
			macd.SlowPeriod = 26
		}
		if macd.SignalPeriod <= 0 {
			macd.SignalPeriod = 9
		}
		out.MACD = &macd
	}
	if out.Bollinger != nil {
		bb := *out.Bollinger
		if bb.Period <= 0 {
			bb.Period = 20
		}
		if bb.StdDev <= 0 {
			bb.StdDev = 2.0
		}
		out.Bollinger = &bb
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
	if c.MACD != nil {
		if c.MACD.FastPeriod < 1 {
			return fmt.Errorf("indicator: macd fast_period must be >= 1")
		}
		if c.MACD.SlowPeriod < 1 {
			return fmt.Errorf("indicator: macd slow_period must be >= 1")
		}
		if c.MACD.SignalPeriod < 1 {
			return fmt.Errorf("indicator: macd signal_period must be >= 1")
		}
		if c.MACD.FastPeriod >= c.MACD.SlowPeriod {
			return fmt.Errorf("indicator: macd fast_period must be less than slow_period")
		}
	}
	if c.Bollinger != nil {
		if c.Bollinger.Period < 1 {
			return fmt.Errorf("indicator: bollinger period must be >= 1")
		}
		if c.Bollinger.StdDev <= 0 {
			return fmt.Errorf("indicator: bollinger stddev must be > 0")
		}
	}
	return nil
}

func (c Config) hasIndicators() bool {
	return len(c.EMA) > 0 || len(c.SMA) > 0 || len(c.RSI) > 0 || len(c.ATR) > 0 ||
		c.MACD != nil || c.Bollinger != nil
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
