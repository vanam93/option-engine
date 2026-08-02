package signal

import "fmt"

// EMACrossConfig configures the fast/slow EMA crossover rule.
type EMACrossConfig struct {
	Enabled    bool
	FastPeriod int
	SlowPeriod int
}

// MACDCrossConfig configures the MACD line / signal crossover rule.
type MACDCrossConfig struct {
	Enabled bool
}

// RSIConfig configures RSI threshold signals.
type RSIConfig struct {
	Enabled    bool
	Oversold   float64
	Overbought float64
}

// BollingerConfig configures Bollinger band threshold signals.
type BollingerConfig struct {
	Enabled bool
}

// Config controls the signal engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	EMACross         EMACrossConfig
	MACDCross        MACDCrossConfig
	RSI              RSIConfig
	Bollinger        BollingerConfig
}

func (c Config) withDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.EMACross.FastPeriod <= 0 {
		out.EMACross.FastPeriod = 9
	}
	if out.EMACross.SlowPeriod <= 0 {
		out.EMACross.SlowPeriod = 21
	}
	if out.RSI.Oversold <= 0 {
		out.RSI.Oversold = 30
	}
	if out.RSI.Overbought <= 0 {
		out.RSI.Overbought = 70
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("signal: subscriber_buffer must be >= 1")
	}
	if !c.hasActiveRules() {
		return fmt.Errorf("signal: at least one rule must be enabled when engine is enabled")
	}
	if c.EMACross.Enabled {
		if c.EMACross.FastPeriod < 1 || c.EMACross.SlowPeriod < 1 {
			return fmt.Errorf("signal: ema_cross periods must be >= 1")
		}
		if c.EMACross.FastPeriod >= c.EMACross.SlowPeriod {
			return fmt.Errorf("signal: ema_cross fast_period must be less than slow_period")
		}
	}
	if c.RSI.Enabled && c.RSI.Oversold >= c.RSI.Overbought {
		return fmt.Errorf("signal: rsi oversold must be less than overbought")
	}
	return nil
}

func (c Config) hasActiveRules() bool {
	return c.EMACross.Enabled || c.MACDCross.Enabled || c.RSI.Enabled || c.Bollinger.Enabled
}

// ActiveRules returns the names of enabled rules.
func (c Config) ActiveRules() []string {
	var out []string
	if c.EMACross.Enabled {
		out = append(out, strategyEMACross)
	}
	if c.MACDCross.Enabled {
		out = append(out, strategyMACDCross)
	}
	if c.RSI.Enabled {
		out = append(out, strategyRSI)
	}
	if c.Bollinger.Enabled {
		out = append(out, strategyBollinger)
	}
	return out
}
