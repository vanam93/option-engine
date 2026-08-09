package recommendation

import "fmt"

// Config controls the recommendation engine.
type Config struct {
	Enabled            bool
	SubscriberBuffer   int
	StrongBuyThreshold float64
	BuyThreshold       float64
	WatchThreshold     float64
}

func (c Config) WithDefaults() Config {
	out := c
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	if out.StrongBuyThreshold <= 0 {
		out.StrongBuyThreshold = 0.85
	}
	if out.BuyThreshold <= 0 {
		out.BuyThreshold = 0.70
	}
	if out.WatchThreshold <= 0 {
		out.WatchThreshold = 0.40
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("recommendation: subscriber_buffer must be >= 1")
	}
	if c.StrongBuyThreshold <= c.BuyThreshold {
		return fmt.Errorf("recommendation: strong_buy_threshold must be greater than buy_threshold")
	}
	if c.BuyThreshold <= c.WatchThreshold {
		return fmt.Errorf("recommendation: buy_threshold must be greater than watch_threshold")
	}
	return nil
}
