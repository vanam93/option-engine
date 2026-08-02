package intelligence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	optimizationScorePattern = regexp.MustCompile(`(?i)optimization score\s+(\d+(?:\.\d+)?)%`)
	walkforwardScorePattern  = regexp.MustCompile(`(?i)walk-forward validation score\s+(\d+(?:\.\d+)?)%`)
	montecarloScorePattern   = regexp.MustCompile(`(?i)monte carlo profit probability\s+(\d+(?:\.\d+)?)%`)
)

// Explainer generates human-readable explanations from recommendation state.
type Explainer struct {
	cfg Config
	fmt *Formatter
}

// NewExplainer creates an explainer.
func NewExplainer(cfg Config, fmt *Formatter) *Explainer {
	return &Explainer{cfg: cfg.withDefaults(), fmt: fmt}
}

// ClassifyLevel maps confidence to a recommendation level.
func (e *Explainer) ClassifyLevel(confidence float64) Level {
	switch {
	case confidence >= e.cfg.StrongBuyThreshold:
		return LevelStrongBuy
	case confidence >= e.cfg.BuyThreshold:
		return LevelBuy
	case confidence >= e.cfg.WatchThreshold:
		return LevelWatch
	default:
		return LevelAvoid
	}
}

// LevelFromStatus derives a level from lifecycle status when recommendation level is absent.
func (e *Explainer) LevelFromStatus(status Status, confidence float64) Level {
	if level := e.ClassifyLevel(confidence); status == StatusActive && level != LevelAvoid {
		return level
	}
	switch status {
	case StatusActive:
		return LevelBuy
	case StatusWatch:
		return LevelWatch
	case StatusExitRecommended, StatusClosed:
		return LevelAvoid
	default:
		return e.ClassifyLevel(confidence)
	}
}

// ResolveLevel returns the effective recommendation level.
func (e *Explainer) ResolveLevel(update StateUpdate) Level {
	if update.Recommendation != "" {
		return update.Recommendation
	}
	return e.LevelFromStatus(update.CurrentStatus, update.Confidence)
}

// BuildEvidence assembles structured research evidence from available update data.
func (e *Explainer) BuildEvidence(update StateUpdate) ResearchEvidence {
	evidence := ResearchEvidence{}

	if len(update.SupportingIndicators) > 0 {
		evidence.Signal = fmt.Sprintf("Signal: %s", strings.Join(update.SupportingIndicators, "; "))
	} else if signal := component(update.Components, "signal"); signal > 0 {
		evidence.Signal = fmt.Sprintf("Signal analytics support is %s at %.0f%%.",
			StrengthLabel(signal), signal*100)
	}

	if len(update.SupportingStrategies) > 0 {
		evidence.Strategy = fmt.Sprintf("Strategy: %s", strings.Join(update.SupportingStrategies, "; "))
	} else if strategy := component(update.Components, "strategy"); strategy > 0 {
		evidence.Strategy = fmt.Sprintf("Strategy alignment is %s at %.0f%%.",
			StrengthLabel(strategy), strategy*100)
	} else if update.Strategy != "" {
		evidence.Strategy = fmt.Sprintf("Trend Following confirmed via %s strategy.", update.Strategy)
	}

	switch update.CurrentStatus {
	case StatusActive, StatusWatch:
		evidence.Risk = "Approved by validation engine."
	case StatusExitRecommended:
		evidence.Risk = "Exit recommended; risk controls suggest reducing exposure."
	case StatusClosed:
		evidence.Risk = "Recommendation closed; validation or lifecycle rules ended the setup."
	}

	if perf := component(update.Components, "performance"); perf > 0 {
		evidence.Performance = fmt.Sprintf("Historical win rate and performance score at %.0f%%.", perf*100)
	}

	if update.OptimizationSummary != "" {
		evidence.Optimization = update.OptimizationSummary
	} else if opt := component(update.Components, "optimization"); opt > 0 {
		evidence.Optimization = fmt.Sprintf("Optimization score %.0f%% (%s parameter fit).",
			opt*100, StrengthLabel(opt))
	} else if score, ok := parsePercentSummary(update.Summary, optimizationScorePattern); ok {
		evidence.Optimization = fmt.Sprintf("Optimization score %.0f%%.", score*100)
	}

	if update.WalkForwardSummary != "" {
		evidence.WalkForward = update.WalkForwardSummary
	} else if wf := component(update.Components, "walkforward"); wf > 0 {
		evidence.WalkForward = fmt.Sprintf("Walk-forward validation score %.0f%% (%s out-of-sample robustness).",
			wf*100, StrengthLabel(wf))
	} else if score, ok := parsePercentSummary(update.Summary, walkforwardScorePattern); ok {
		evidence.WalkForward = fmt.Sprintf("Walk-forward validation score %.0f%%.", score*100)
	}

	if update.MonteCarloSummary != "" {
		evidence.MonteCarlo = update.MonteCarloSummary
	} else if mc := component(update.Components, "montecarlo"); mc > 0 {
		evidence.MonteCarlo = fmt.Sprintf("Monte Carlo profit probability %.0f%%; risk of ruin assessed.",
			mc*100)
	} else if score, ok := parsePercentSummary(update.Summary, montecarloScorePattern); ok {
		evidence.MonteCarlo = fmt.Sprintf("Monte Carlo profit probability %.0f%%.", score*100)
	}

	if dd := component(update.Components, "drawdown"); dd > 0 {
		evidence.Drawdown = fmt.Sprintf("Maximum drawdown within acceptable bounds (%.0f%%).", dd*100)
	}

	return evidence
}

// BuildConfidenceBreakdown assembles per-factor confidence contributions.
func (e *Explainer) BuildConfidenceBreakdown(update StateUpdate) ConfidenceBreakdown {
	breakdown := ConfidenceBreakdown{Overall: update.Confidence}

	if v := componentPtr(update.Components, "signal"); v != nil {
		breakdown.Signal = v
	}
	if v := componentPtr(update.Components, "strategy"); v != nil {
		breakdown.Strategy = v
	}
	if v := componentPtr(update.Components, "performance"); v != nil {
		breakdown.Performance = v
	}
	if v := componentPtr(update.Components, "optimization"); v != nil {
		breakdown.Optimization = v
	}
	if v := componentPtr(update.Components, "walkforward"); v != nil {
		breakdown.WalkForward = v
	}
	if v := componentPtr(update.Components, "montecarlo"); v != nil {
		breakdown.MonteCarlo = v
	}

	if update.CurrentStatus == StatusActive || update.CurrentStatus == StatusWatch {
		validation := 1.0
		if update.CurrentStatus == StatusWatch {
			validation = 0.7
		}
		breakdown.Validation = &validation
	}

	return breakdown
}

// SupportingFactors extracts positive factors from update context.
func (e *Explainer) SupportingFactors(update StateUpdate, evidence ResearchEvidence, level Level) []string {
	factors := make([]string, 0, 8)
	factors = append(factors, update.Reasons...)

	entry := update.LatestTimelineEntry
	switch entry.Event {
	case "Confidence Increased":
		factors = append(factors, "Confidence increased based on validated recommendation update.")
	case "Status Changed":
		if update.CurrentStatus == StatusActive {
			factors = append(factors, "Recommendation upgraded to active status.")
		}
	}

	if evidence.Signal != "" {
		factors = append(factors, evidence.Signal)
	}
	if evidence.Strategy != "" {
		factors = append(factors, evidence.Strategy)
	}
	if evidence.Optimization != "" {
		factors = append(factors, evidence.Optimization)
	}
	if evidence.WalkForward != "" {
		factors = append(factors, evidence.WalkForward)
	}
	if level == LevelStrongBuy {
		factors = append(factors, "Multiple intelligence factors align for a high-conviction recommendation.")
	}

	return dedupeStrings(factors)
}

// RiskFactors extracts risk-related factors.
func (e *Explainer) RiskFactors(update StateUpdate, evidence ResearchEvidence) []string {
	factors := make([]string, 0, 4)

	entry := update.LatestTimelineEntry
	switch entry.Event {
	case "Confidence Decreased":
		factors = append(factors, "Confidence decreased; conviction has weakened.")
	case "Exit Recommended", "Closed":
		factors = append(factors, entry.Reason)
	}

	if update.CurrentStatus == StatusExitRecommended {
		factors = append(factors, "Exit recommended by lifecycle rules.")
	}
	if update.CurrentStatus == StatusClosed {
		factors = append(factors, "Recommendation is closed.")
	}
	if strings.Contains(strings.ToLower(entry.Reason), "reject") {
		factors = append(factors, entry.Reason)
	}
	if evidence.Drawdown != "" {
		factors = append(factors, evidence.Drawdown)
	}

	return dedupeStrings(factors)
}

// DetectChange identifies upgrade and downgrade reasons from prior state.
func (e *Explainer) DetectChange(update StateUpdate, previous *storedSnapshot) (upgrade, downgrade string) {
	if previous == nil {
		return "", ""
	}

	currentLevel := e.ResolveLevel(update)
	prevLevel := previous.level
	if prevLevel == "" {
		prevLevel = e.LevelFromStatus(previous.status, previous.confidence)
	}

	if levelRank(currentLevel) > levelRank(prevLevel) {
		upgrade = e.buildUpgradeReason(update, previous, prevLevel, currentLevel)
	} else if levelRank(currentLevel) < levelRank(prevLevel) {
		downgrade = e.buildDowngradeReason(update, previous, prevLevel, currentLevel)
	}

	if upgrade == "" && downgrade == "" && previous.status != update.CurrentStatus {
		if statusRank(update.CurrentStatus) > statusRank(previous.status) {
			upgrade = fmt.Sprintf("Status changed from %s to %s.",
				e.fmt.StatusLabel(previous.status), e.fmt.StatusLabel(update.CurrentStatus))
		} else if statusRank(update.CurrentStatus) < statusRank(previous.status) {
			downgrade = fmt.Sprintf("Status changed from %s to %s.",
				e.fmt.StatusLabel(previous.status), e.fmt.StatusLabel(update.CurrentStatus))
		}
	}

	return upgrade, downgrade
}

func (e *Explainer) buildUpgradeReason(update StateUpdate, previous *storedSnapshot, from, to Level) string {
	reasons := make([]string, 0, 4)
	reasons = append(reasons, fmt.Sprintf("Recommendation upgraded from %s to %s.",
		e.fmt.LevelLabel(from), e.fmt.LevelLabel(to)))

	if update.Confidence > previous.confidence {
		reasons = append(reasons, "Confidence increased.")
	}
	entry := update.LatestTimelineEntry
	if strings.Contains(strings.ToLower(entry.Reason), "optimization") {
		reasons = append(reasons, "Optimization improved.")
	}
	if update.CurrentStatus == StatusActive && previous.status != StatusActive {
		reasons = append(reasons, "Validation succeeded.")
	}
	return strings.Join(reasons, " ")
}

func (e *Explainer) buildDowngradeReason(update StateUpdate, previous *storedSnapshot, from, to Level) string {
	reasons := make([]string, 0, 4)
	reasons = append(reasons, fmt.Sprintf("Recommendation downgraded from %s to %s.",
		e.fmt.LevelLabel(from), e.fmt.LevelLabel(to)))

	if update.Confidence < previous.confidence {
		reasons = append(reasons, "Confidence decreased.")
	}
	entry := update.LatestTimelineEntry
	if strings.Contains(strings.ToLower(entry.Reason), "performance") {
		reasons = append(reasons, "Performance deteriorated.")
	}
	if strings.Contains(strings.ToLower(entry.Reason), "walk-forward") || strings.Contains(strings.ToLower(entry.Reason), "walkforward") {
		reasons = append(reasons, "Walk-forward weakened.")
	}
	return strings.Join(reasons, " ")
}

// BuildExplanation composes the primary human-readable explanation.
func (e *Explainer) BuildExplanation(update StateUpdate, level Level, evidence ResearchEvidence, upgrade, downgrade string) string {
	parts := make([]string, 0, 6)

	switch {
	case upgrade != "":
		parts = append(parts, upgrade)
	case downgrade != "":
		parts = append(parts, downgrade)
	default:
		parts = append(parts, fmt.Sprintf("%s is a %s recommendation with %s confidence.",
			update.Symbol, e.fmt.LevelLabel(level), e.fmt.FormatConfidence(update.Confidence)))
	}

	entry := update.LatestTimelineEntry
	if entry.Event != "" {
		parts = append(parts, fmt.Sprintf("Latest event: %s — %s.", entry.Event, entry.Reason))
	}

	if evidence.Risk != "" {
		parts = append(parts, evidence.Risk)
	}

	return e.fmt.JoinSentences(parts...)
}

func levelRank(level Level) int {
	switch level {
	case LevelStrongBuy:
		return 4
	case LevelBuy:
		return 3
	case LevelWatch:
		return 2
	case LevelAvoid:
		return 1
	default:
		return 0
	}
}

func statusRank(status Status) int {
	switch status {
	case StatusActive:
		return 4
	case StatusWatch:
		return 3
	case StatusExitRecommended:
		return 2
	case StatusClosed:
		return 1
	case StatusCreated:
		return 0
	default:
		return 0
	}
}

func component(components map[string]float64, key string) float64 {
	if components == nil {
		return 0
	}
	return components[key]
}

func componentPtr(components map[string]float64, key string) *float64 {
	v := component(components, key)
	if v <= 0 {
		return nil
	}
	out := v
	return &out
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

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
