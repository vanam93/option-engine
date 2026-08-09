package strategylib

import (
	"time"
)

// Signal is the rich output of strategy evaluation for simulators and reports.
type Signal struct {
	Decision    Decision           `json:"decision"`
	Confidence  float64            `json:"confidence"`
	Strength    float64            `json:"strength"`
	Score       float64            `json:"score"`
	Reasons     []string           `json:"reasons,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Indicators  map[string]float64 `json:"indicators,omitempty"`
	Parameters  map[string]any     `json:"parameters,omitempty"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// IsAction reports whether the signal requests a trade action.
func (s Signal) IsAction() bool {
	return s.Decision == DecisionBuy || s.Decision == DecisionSell || s.Decision == DecisionExit
}

// SignalBuilder constructs signals with shared parameter and timestamp context.
type SignalBuilder struct {
	params map[string]any
	at     time.Time
}

// NewSignalBuilder creates a builder for a strategy evaluation bar.
func NewSignalBuilder(params map[string]any, at time.Time) SignalBuilder {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return SignalBuilder{
		params: CloneParams(params),
		at:     at.UTC(),
	}
}

// Ignore returns an IGNORE signal.
func (b SignalBuilder) Ignore() Signal {
	return Signal{
		Decision:    DecisionIgnore,
		Parameters:  CloneParams(b.params),
		GeneratedAt: b.at,
	}
}

// IgnoreWithIndicators returns IGNORE but preserves indicator snapshot for warmup bars.
func (b SignalBuilder) IgnoreWithIndicators(indicators map[string]float64) Signal {
	return Signal{
		Decision:    DecisionIgnore,
		Indicators:  cloneIndicators(indicators),
		Parameters:  CloneParams(b.params),
		GeneratedAt: b.at,
	}
}

func cloneIndicators(ind map[string]float64) map[string]float64 {
	if len(ind) == 0 {
		return nil
	}
	out := make(map[string]float64, len(ind))
	for k, v := range ind {
		out[k] = v
	}
	return out
}

// Action builds an actionable signal with decision, scores, reasons, and tags.
func (b SignalBuilder) Action(
	decision Decision,
	confidence, strength, score float64,
	reasons, tags []string,
	indicators map[string]float64,
) Signal {
	return Signal{
		Decision:    decision,
		Confidence:  clamp01(confidence),
		Strength:    clamp01(strength),
		Score:       clamp01(score),
		Reasons:     append([]string(nil), reasons...),
		Tags:        append([]string(nil), tags...),
		Indicators:  cloneIndicators(indicators),
		Parameters:  CloneParams(b.params),
		GeneratedAt: b.at,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
