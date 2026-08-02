package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/candle"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// BuildCandleEngineConfig maps application config into the candle engine config.
func BuildCandleEngineConfig(cfg CandleEngineConfig) (candle.Config, error) {
	out := candle.Config{
		Enabled:          cfg.Enabled,
		Timezone:         cfg.Timezone,
		SubscriberBuffer: cfg.SubscriberBuffer,
		FlushOnShutdown:  cfg.FlushOnShutdown,
		VolumeMode:       candle.VolumeMode(cfg.VolumeMode),
		OrderPolicy:      candle.OrderPolicy(cfg.OrderPolicy),
		IdleEvictAfter:   cfg.IdleEvictAfter,
	}
	for _, raw := range cfg.Timeframes {
		tf := market.Timeframe(raw)
		if _, err := candle.Duration(tf); err != nil {
			return candle.Config{}, fmt.Errorf("analytics candle timeframe %q: %w", raw, err)
		}
		out.Timeframes = append(out.Timeframes, tf)
	}
	return out, nil
}
