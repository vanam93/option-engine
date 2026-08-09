package validation

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	optimizationScorePattern = regexp.MustCompile(`(?i)optimization score\s+(\d+(?:\.\d+)?)%`)
	walkforwardScorePattern  = regexp.MustCompile(`(?i)walk-forward validation score\s+(\d+(?:\.\d+)?)%`)
	montecarloScorePattern   = regexp.MustCompile(`(?i)monte carlo profit probability\s+(\d+(?:\.\d+)?)%`)
)

// Validator applies research-quality thresholds to recommendations.
type Validator struct {
	cfg Config
}

// NewValidator creates a recommendation validator.
func NewValidator(cfg Config) *Validator {
	return &Validator{cfg: cfg.WithDefaults()}
}

// ValidationOutcome captures the result of threshold checks.
type ValidationOutcome struct {
	Result ValidatedRecommendation
	Score  float64
}

// Validate checks a recommendation against configured thresholds.
func (v *Validator) Validate(input InputRecommendation, at time.Time) ValidationOutcome {
	cfg := v.cfg
	reasons := make([]string, 0, 8)
	scores := make([]float64, 0, 6)
	passed := 0
	checked := 0

	if input.Confidence < cfg.MinConfidence {
		reasons = append(reasons, "confidence below minimum threshold")
	}
	scores = append(scores, input.Confidence)
	checked++
	if input.Confidence >= cfg.MinConfidence {
		passed++
	}

	optimizationScore := input.optimizationScore()
	if input.hasOptimizationScore {
		checked++
		scores = append(scores, optimizationScore)
		if optimizationScore < cfg.MinOptimizationScore {
			reasons = append(reasons, "optimization score below minimum threshold")
		} else {
			passed++
		}
	}

	walkforwardScore := input.walkforwardScore()
	if input.hasWalkforwardScore {
		checked++
		scores = append(scores, walkforwardScore)
		if walkforwardScore < cfg.MinWalkforwardScore {
			reasons = append(reasons, "walk-forward score below minimum threshold")
		} else {
			passed++
		}
	}

	monteCarloScore := input.monteCarloScore()
	if input.hasMonteCarloScore {
		checked++
		scores = append(scores, monteCarloScore)
		if monteCarloScore < cfg.MinMonteCarloScore {
			reasons = append(reasons, "monte carlo robustness below minimum threshold")
		} else {
			passed++
		}
	}

	if input.hasWinRate {
		checked++
		scores = append(scores, input.WinRate)
		if input.WinRate < cfg.MinWinRate {
			reasons = append(reasons, "win rate below minimum threshold")
		} else {
			passed++
		}
	}

	if input.hasDrawdown {
		checked++
		scores = append(scores, 1-input.Drawdown)
		if input.Drawdown > cfg.MaxDrawdown {
			reasons = append(reasons, "drawdown exceeds maximum threshold")
		} else {
			passed++
		}
	}

	if !input.GeneratedAt.IsZero() {
		age := at.Sub(input.GeneratedAt)
		if age < 0 {
			age = 0
		}
		checked++
		freshnessScore := 1.0
		if !v.cfg.ReplayMode && age > time.Duration(cfg.FreshnessSeconds)*time.Second {
			reasons = append(reasons, "recommendation is stale")
			freshnessScore = 0
		} else {
			passed++
		}
		scores = append(scores, freshnessScore)
	} else {
		reasons = append(reasons, "recommendation timestamp missing")
	}

	status := StatusValid
	if len(reasons) > 0 {
		status = StatusRejected
	}

	validationScore := 0.0
	if checked > 0 {
		validationScore = float64(passed) / float64(checked)
	}
	if len(scores) > 0 {
		mean := 0.0
		for _, score := range scores {
			mean += score
		}
		validationScore = (validationScore + mean/float64(len(scores))) / 2
	}

	return ValidationOutcome{
		Result: ValidatedRecommendation{
			Symbol:           input.Symbol,
			Timeframe:        input.Timeframe,
			Recommendation:   input.Recommendation,
			Confidence:       input.Confidence,
			ValidationStatus: status,
			RejectionReasons: reasons,
			ValidatedAt:      at,
		},
		Score: validationScore,
	}
}

// IsDuplicate reports whether the incoming recommendation matches the latest cached validation.
func (v *Validator) IsDuplicate(input InputRecommendation, previous ValidatedRecommendation) bool {
	if !v.cfg.SuppressDuplicates {
		return false
	}
	if previous.Symbol == "" || previous.Timeframe == "" {
		return false
	}
	if previous.Recommendation != input.Recommendation {
		return false
	}
	return math.Abs(previous.Confidence-input.Confidence) < 0.0001
}

func (input *InputRecommendation) optimizationScore() float64 {
	if input.hasOptimizationScore {
		return input.OptimizationScore
	}
	if score, ok := parsePercentSummary(input.OptimizationSummary, optimizationScorePattern); ok {
		input.OptimizationScore = score
		input.hasOptimizationScore = true
		return score
	}
	return 0
}

func (input *InputRecommendation) walkforwardScore() float64 {
	if input.hasWalkforwardScore {
		return input.WalkforwardScore
	}
	if score, ok := parsePercentSummary(input.WalkForwardSummary, walkforwardScorePattern); ok {
		input.WalkforwardScore = score
		input.hasWalkforwardScore = true
		return score
	}
	return 0
}

func (input *InputRecommendation) monteCarloScore() float64 {
	if input.hasMonteCarloScore {
		return input.MonteCarloScore
	}
	if score, ok := parsePercentSummary(input.MonteCarloSummary, montecarloScorePattern); ok {
		input.MonteCarloScore = score
		input.hasMonteCarloScore = true
		return score
	}
	return 0
}

func parsePercentSummary(summary string, pattern *regexp.Regexp) (float64, bool) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return 0, false
	}
	matches := pattern.FindStringSubmatch(summary)
	if len(matches) < 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, false
	}
	return value / 100, true
}
