package candle

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

const engineName = "candle_engine"

// Config controls candle aggregation behaviour.
type Config struct {
	Enabled          bool
	Timeframes       []market.Timeframe
	Timezone         string
	SubscriberBuffer int
	FlushOnShutdown  bool
	VolumeMode       VolumeMode
	OrderPolicy      OrderPolicy
	IdleEvictAfter   time.Duration
}

// WithDefaults returns cfg with production defaults applied for unset enum fields.
func (c Config) WithDefaults() Config {
	out := c
	if out.VolumeMode == "" {
		out.VolumeMode = VolumeCumulative
	}
	if out.OrderPolicy == "" {
		out.OrderPolicy = OrderRejectOlder
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if len(c.Timeframes) == 0 {
		return fmt.Errorf("candle: at least one timeframe is required when enabled")
	}
	for _, tf := range c.Timeframes {
		if _, err := Duration(tf); err != nil {
			return err
		}
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("candle: subscriber_buffer must be >= 1")
	}
	switch c.VolumeMode {
	case "", VolumeCumulative, VolumeIncremental:
	default:
		return fmt.Errorf("candle: invalid volume_mode %q", c.VolumeMode)
	}
	switch c.OrderPolicy {
	case "", OrderRejectOlder, OrderRejectStale:
	default:
		return fmt.Errorf("candle: invalid order_policy %q", c.OrderPolicy)
	}
	if c.IdleEvictAfter < 0 {
		return fmt.Errorf("candle: idle_evict_after must be >= 0")
	}
	return nil
}

// Location resolves the timezone used for candle bucket alignment.
func (c Config) Location() (*time.Location, error) {
	tz := c.Timezone
	if tz == "" {
		tz = "Asia/Kolkata"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("candle: load timezone %q: %w", tz, err)
	}
	return loc, nil
}

func (c Config) volumeMode() VolumeMode {
	if c.VolumeMode == "" {
		return VolumeCumulative
	}
	return c.VolumeMode
}

func (c Config) orderPolicy() OrderPolicy {
	if c.OrderPolicy == "" {
		return OrderRejectOlder
	}
	return c.OrderPolicy
}
