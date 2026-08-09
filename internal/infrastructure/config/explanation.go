package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/intelligence"
)

// ExplanationConfig controls the recommendation intelligence engine.
type ExplanationConfig struct {
	Enabled                    bool `mapstructure:"enabled"`
	SubscriberBuffer           int  `mapstructure:"subscriber_buffer"`
	IncludeTimeline            bool `mapstructure:"include_timeline"`
	IncludeResearch            bool `mapstructure:"include_research"`
	IncludeConfidenceBreakdown bool `mapstructure:"include_confidence_breakdown"`
}

// ExplanationEngineSettings maps intelligence explanation settings.
func (c *Config) ExplanationEngineSettings() ExplanationEngineConfig {
	rec := c.Intelligence.Recommendation
	return ExplanationEngineConfig{
		Enabled:                    c.Intelligence.Explanation.Enabled,
		SubscriberBuffer:           c.Intelligence.Explanation.SubscriberBuffer,
		IncludeTimeline:            c.Intelligence.Explanation.IncludeTimeline,
		IncludeResearch:            c.Intelligence.Explanation.IncludeResearch,
		IncludeConfidenceBreakdown: c.Intelligence.Explanation.IncludeConfidenceBreakdown,
		StrongBuyThreshold:         rec.StrongBuyThreshold,
		BuyThreshold:               rec.BuyThreshold,
		WatchThreshold:             rec.WatchThreshold,
	}
}

// ExplanationEngineConfig is the validated explanation configuration used by DI wiring.
type ExplanationEngineConfig struct {
	Enabled                    bool
	SubscriberBuffer           int
	IncludeTimeline            bool
	IncludeResearch            bool
	IncludeConfidenceBreakdown bool
	StrongBuyThreshold         float64
	BuyThreshold               float64
	WatchThreshold             float64
}

// BuildExplanationEngineConfig maps application config into the intelligence engine config.
func BuildExplanationEngineConfig(cfg ExplanationEngineConfig) (intelligence.Config, error) {
	out := intelligence.Config{
		Enabled:                    cfg.Enabled,
		SubscriberBuffer:           cfg.SubscriberBuffer,
		IncludeTimeline:            cfg.IncludeTimeline,
		IncludeResearch:            cfg.IncludeResearch,
		IncludeConfidenceBreakdown: cfg.IncludeConfidenceBreakdown,
		StrongBuyThreshold:         cfg.StrongBuyThreshold,
		BuyThreshold:               cfg.BuyThreshold,
		WatchThreshold:             cfg.WatchThreshold,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return intelligence.Config{}, fmt.Errorf("explanation config: %w", err)
	}
	return out, nil
}
