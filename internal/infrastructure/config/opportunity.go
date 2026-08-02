package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
)

// IntelligenceConfig groups Stage 5 intelligence engine settings.
type IntelligenceConfig struct {
	Opportunity         OpportunityConfig         `mapstructure:"opportunity"`
	Recommendation      RecommendationConfig      `mapstructure:"recommendation"`
	Validation          ValidationEngineConfig    `mapstructure:"validation"`
	RecommendationState RecommendationStateConfig `mapstructure:"recommendation_state"`
	Alerts              AlertsConfig              `mapstructure:"alerts"`
	Explanation         ExplanationConfig         `mapstructure:"explanation"`
	Quality             QualityConfig             `mapstructure:"quality"`
	Feedback            FeedbackConfig            `mapstructure:"feedback"`
}

// OpportunityConfig controls the opportunity ranking engine.
type OpportunityConfig struct {
	Enabled          bool                     `mapstructure:"enabled"`
	TopN             int                      `mapstructure:"top_n"`
	SubscriberBuffer int                      `mapstructure:"subscriber_buffer"`
	BuyThreshold     float64                  `mapstructure:"buy_threshold"`
	WatchThreshold   float64                  `mapstructure:"watch_threshold"`
	Weights          OpportunityWeightsConfig `mapstructure:"weights"`
}

// OpportunityWeightsConfig holds scoring weight configuration.
type OpportunityWeightsConfig struct {
	Signal       float64 `mapstructure:"signal"`
	Strategy     float64 `mapstructure:"strategy"`
	Performance  float64 `mapstructure:"performance"`
	Optimization float64 `mapstructure:"optimization"`
	WalkForward  float64 `mapstructure:"walkforward"`
	MonteCarlo   float64 `mapstructure:"montecarlo"`
}

// OpportunityEngineConfig is the validated opportunity configuration used by DI wiring.
type OpportunityEngineConfig struct {
	Enabled          bool
	TopN             int
	SubscriberBuffer int
	BuyThreshold     float64
	WatchThreshold   float64
	Weights          OpportunityWeightsConfig
}

// OpportunityEngineSettings maps intelligence opportunity settings.
func (c *Config) OpportunityEngineSettings() OpportunityEngineConfig {
	return OpportunityEngineConfig{
		Enabled:          c.Intelligence.Opportunity.Enabled,
		TopN:             c.Intelligence.Opportunity.TopN,
		SubscriberBuffer: c.Intelligence.Opportunity.SubscriberBuffer,
		BuyThreshold:     c.Intelligence.Opportunity.BuyThreshold,
		WatchThreshold:   c.Intelligence.Opportunity.WatchThreshold,
		Weights:          c.Intelligence.Opportunity.Weights,
	}
}

// BuildOpportunityEngineConfig maps application config into the opportunity engine config.
func BuildOpportunityEngineConfig(cfg OpportunityEngineConfig) (opportunity.Config, error) {
	out := opportunity.Config{
		Enabled:          cfg.Enabled,
		TopN:             cfg.TopN,
		SubscriberBuffer: cfg.SubscriberBuffer,
		BuyThreshold:     cfg.BuyThreshold,
		WatchThreshold:   cfg.WatchThreshold,
		Weights: opportunity.WeightsConfig{
			Signal:       cfg.Weights.Signal,
			Strategy:     cfg.Weights.Strategy,
			Performance:  cfg.Weights.Performance,
			Optimization: cfg.Weights.Optimization,
			WalkForward:  cfg.Weights.WalkForward,
			MonteCarlo:   cfg.Weights.MonteCarlo,
		},
	}
	if err := out.Validate(); err != nil {
		return opportunity.Config{}, fmt.Errorf("opportunity config: %w", err)
	}
	return out, nil
}
