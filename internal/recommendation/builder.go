package recommendation

import "time"

// Builder transforms opportunity intelligence into recommendations.
type Builder struct {
	cfg       Config
	formatter *Formatter
}

// NewBuilder creates a recommendation builder.
func NewBuilder(cfg Config, formatter *Formatter) *Builder {
	return &Builder{
		cfg:       cfg.withDefaults(),
		formatter: formatter,
	}
}

// Build creates a recommendation from an opportunity update.
func (b *Builder) Build(input InputOpportunity, at time.Time) RecommendationUpdated {
	if at.IsZero() {
		at = input.Timestamp
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	level := ClassifyLevel(input.Confidence, b.cfg)
	components := copyComponents(input.Components)

	return RecommendationUpdated{
		Symbol:               input.Symbol,
		Timeframe:            input.Timeframe,
		Recommendation:       level,
		Confidence:           input.Confidence,
		Rank:                 input.Rank,
		Reasons:              b.formatter.Reasons(input, level),
		SupportingIndicators: b.formatter.SupportingIndicators(components),
		SupportingStrategies: b.formatter.SupportingStrategies(components),
		OptimizationSummary:  b.formatter.OptimizationSummary(components),
		WalkForwardSummary:   b.formatter.WalkForwardSummary(components),
		MonteCarloSummary:    b.formatter.MonteCarloSummary(components),
		GeneratedAt:          at,
	}
}

func copyComponents(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
