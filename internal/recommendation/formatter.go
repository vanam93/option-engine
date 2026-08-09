package recommendation

import (
	"fmt"
	"strings"
)

// Formatter produces human-readable recommendation explanations.
type Formatter struct{}

// NewFormatter creates a recommendation formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Reasons builds human-readable reasons from opportunity intelligence.
func (f *Formatter) Reasons(input InputOpportunity, level Level) []string {
	reasons := make([]string, 0, 6)

	reasons = append(reasons, fmt.Sprintf(
		"Overall confidence is %.0f%% with rank #%d among tracked opportunities.",
		input.Confidence*100, input.Rank,
	))

	if signal := component(input.Components, "signal"); signal > 0 {
		reasons = append(reasons, fmt.Sprintf(
			"Signal analytics support is %s at %.0f%%.",
			strengthLabel(signal), signal*100,
		))
	}
	if strategy := component(input.Components, "strategy"); strategy > 0 {
		reasons = append(reasons, fmt.Sprintf(
			"Strategy alignment is %s at %.0f%%.",
			strengthLabel(strategy), strategy*100,
		))
	}
	if performance := component(input.Components, "performance"); performance > 0 {
		reasons = append(reasons, fmt.Sprintf(
			"Recent simulated performance score is %.0f%%.",
			performance*100,
		))
	}
	if risk := component(input.Components, "risk_factor"); risk > 0 && risk < 1 {
		reasons = append(reasons, "Risk checks did not fully approve the setup; confidence was reduced.")
	}
	if level == LevelStrongBuy {
		reasons = append(reasons, "Multiple intelligence factors align for a high-conviction recommendation.")
	}
	if level == LevelAvoid {
		reasons = append(reasons, "Confidence is below the watch threshold; no actionable setup identified.")
	}

	return reasons
}

// SupportingIndicators lists indicator-related evidence from opportunity components.
func (f *Formatter) SupportingIndicators(components map[string]float64) []string {
	signal := component(components, "signal")
	if signal <= 0 {
		return nil
	}
	return []string{
		fmt.Sprintf("signal_confidence=%.2f", signal),
		fmt.Sprintf("signal_strength=%s", strengthLabel(signal)),
	}
}

// SupportingStrategies lists strategy-related evidence from opportunity components.
func (f *Formatter) SupportingStrategies(components map[string]float64) []string {
	strategy := component(components, "strategy")
	if strategy <= 0 {
		return nil
	}
	return []string{
		fmt.Sprintf("strategy_confidence=%.2f", strategy),
		fmt.Sprintf("strategy_alignment=%s", strengthLabel(strategy)),
	}
}

// OptimizationSummary formats the optimization component.
func (f *Formatter) OptimizationSummary(components map[string]float64) string {
	score := component(components, "optimization")
	if score <= 0 {
		return "No optimization score available for this symbol."
	}
	return fmt.Sprintf("Optimization score %.0f%% (%s parameter fit).",
		score*100, strengthLabel(score))
}

// WalkForwardSummary formats the walk-forward validation component.
func (f *Formatter) WalkForwardSummary(components map[string]float64) string {
	score := component(components, "walkforward")
	if score <= 0 {
		return "Walk-forward validation data not yet available."
	}
	return fmt.Sprintf("Walk-forward validation score %.0f%% (%s out-of-sample robustness).",
		score*100, strengthLabel(score))
}

// MonteCarloSummary formats the Monte Carlo robustness component.
func (f *Formatter) MonteCarloSummary(components map[string]float64) string {
	score := component(components, "montecarlo")
	if score <= 0 {
		return "Monte Carlo robustness data not yet available."
	}
	return fmt.Sprintf("Monte Carlo profit probability %.0f%% (%s simulated robustness).",
		score*100, strengthLabel(score))
}

func component(components map[string]float64, key string) float64 {
	if components == nil {
		return 0
	}
	return components[key]
}

func strengthLabel(score float64) string {
	switch {
	case score >= 0.85:
		return "very strong"
	case score >= 0.70:
		return "strong"
	case score >= 0.50:
		return "moderate"
	case score > 0:
		return "weak"
	default:
		return "unavailable"
	}
}

// ClassifyLevel maps confidence to a recommendation level.
func ClassifyLevel(confidence float64, cfg Config) Level {
	cfg = cfg.WithDefaults()
	switch {
	case confidence >= cfg.StrongBuyThreshold:
		return LevelStrongBuy
	case confidence >= cfg.BuyThreshold:
		return LevelBuy
	case confidence >= cfg.WatchThreshold:
		return LevelWatch
	default:
		return LevelAvoid
	}
}

// LevelLabel returns a display-friendly level name.
func LevelLabel(level Level) string {
	return strings.TrimSpace(string(level))
}
