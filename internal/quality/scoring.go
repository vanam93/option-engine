package quality

import "math"

// ScoreInput bundles fields used by the quality score formula.
type ScoreInput struct {
	ReturnPct   float64
	MFE         float64
	MAE         float64
	HoldingMins float64
	Confidence  float64
	Level       Level
	Outcome     Outcome
}

// levelFactor maps recommendation level to a 0–1 weight.
func levelFactor(level Level) float64 {
	switch level {
	case LevelStrongBuy:
		return 1.0
	case LevelBuy:
		return 0.85
	case LevelWatch:
		return 0.50
	case LevelAvoid:
		return 0.20
	default:
		return 0.50
	}
}

// outcomeFactor maps outcome to a 0–1 weight.
func outcomeFactor(outcome Outcome) float64 {
	switch outcome {
	case OutcomeSuccess:
		return 1.0
	case OutcomeNeutral:
		return 0.50
	case OutcomeExpired:
		return 0.30
	case OutcomeFailed:
		return 0.0
	default:
		return 0.50
	}
}

// durationFactor scores holding duration relative to an optimal window (15–90 minutes).
func durationFactor(holdingMins float64) float64 {
	if holdingMins <= 0 {
		return 0.5
	}
	switch {
	case holdingMins < 5:
		return 0.6
	case holdingMins <= 90:
		return 1.0
	case holdingMins <= 120:
		return 0.8
	default:
		return 0.5
	}
}

// normalizeReturn maps return percentage to 0–1 using a linear cap at ±5%.
func normalizeReturn(returnPct float64) float64 {
	const cap = 0.05
	if returnPct >= cap {
		return 1.0
	}
	if returnPct <= -cap {
		return 0.0
	}
	return (returnPct + cap) / (2 * cap)
}

// ComputeQualityScore calculates a 0.0–1.0 quality score.
//
// Formula (documented in architecture):
//
//	returnFactor  = clamp((return_pct + 0.05) / 0.10, 0, 1)
//	mfeFactor     = clamp(mfe / 0.05, 0, 1)
//	maePenalty    = clamp(1 - mae / 0.03, 0, 1)
//	durationFactor = piecewise optimal window (peak at 5–90 min)
//	confidenceFactor = confidence (already 0–1)
//	levelFactor   = STRONG_BUY=1.0, BUY=0.85, WATCH=0.50, AVOID=0.20
//	outcomeFactor = SUCCESS=1.0, NEUTRAL=0.5, EXPIRED=0.3, FAILED=0.0
//
//	quality_score = 0.30*returnFactor + 0.20*mfeFactor + 0.15*maePenalty
//	              + 0.10*durationFactor + 0.10*confidenceFactor
//	              + 0.05*levelFactor + 0.10*outcomeFactor
func ComputeQualityScore(input ScoreInput) float64 {
	returnFactor := normalizeReturn(input.ReturnPct)
	mfeFactor := clamp(input.MFE/0.05, 0, 1)
	maePenalty := clamp(1-input.MAE/0.03, 0, 1)
	durFactor := durationFactor(input.HoldingMins)
	confFactor := clamp(input.Confidence, 0, 1)
	lvlFactor := levelFactor(input.Level)
	outFactor := outcomeFactor(input.Outcome)

	score := 0.30*returnFactor +
		0.20*mfeFactor +
		0.15*maePenalty +
		0.10*durFactor +
		0.10*confFactor +
		0.05*lvlFactor +
		0.10*outFactor

	return clamp(score, 0, 1)
}

// Classify maps a quality score and outcome to a classification label.
func Classify(cfg Config, score float64, outcome Outcome) Classification {
	if outcome == OutcomeFailed {
		return ClassificationFailed
	}
	if score >= cfg.ExcellentThreshold {
		return ClassificationExcellent
	}
	if score >= cfg.GoodThreshold {
		return ClassificationGood
	}
	if score >= cfg.AverageThreshold {
		return ClassificationAverage
	}
	if score > 0 {
		return ClassificationPoor
	}
	return ClassificationFailed
}

func clamp(v, min, max float64) float64 {
	return math.Max(min, math.Min(max, v))
}
