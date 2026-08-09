package risk

import "fmt"

// Config controls the decision and risk engine.
type Config struct {
	Enabled          bool
	SubscriberBuffer int
	MinConfidence    float64
	MaxPositions     int
	MaxTradesPerDay  int
	DefaultQuantity  int
	DayResetTimezone string
}

func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.MinConfidence <= 0 {
		out.MinConfidence = 0.70
	}
	if out.MaxPositions <= 0 {
		out.MaxPositions = 5
	}
	if out.MaxTradesPerDay <= 0 {
		out.MaxTradesPerDay = 20
	}
	if out.DefaultQuantity <= 0 {
		out.DefaultQuantity = 1
	}
	if out.DayResetTimezone == "" {
		out.DayResetTimezone = "Asia/Kolkata"
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("risk: subscriber_buffer must be >= 1")
	}
	if c.MinConfidence < 0 || c.MinConfidence > 1 {
		return fmt.Errorf("risk: min_confidence must be between 0 and 1")
	}
	if c.MaxPositions < 1 {
		return fmt.Errorf("risk: max_positions must be >= 1")
	}
	if c.MaxTradesPerDay < 1 {
		return fmt.Errorf("risk: max_trades_per_day must be >= 1")
	}
	if c.DefaultQuantity < 1 {
		return fmt.Errorf("risk: default_quantity must be >= 1")
	}
	return nil
}
