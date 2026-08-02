package intelligence

import (
	"fmt"
	"strings"
	"time"
)

// SummaryBuilder produces narrative summaries from recommendation state.
type SummaryBuilder struct {
	fmt *Formatter
}

// NewSummaryBuilder creates a summary builder.
func NewSummaryBuilder(fmt *Formatter) *SummaryBuilder {
	return &SummaryBuilder{fmt: fmt}
}

// ResearchSummary builds a research narrative from evidence and update context.
func (s *SummaryBuilder) ResearchSummary(update StateUpdate, evidence ResearchEvidence) string {
	if !hasResearchEvidence(evidence) {
		return ""
	}

	parts := make([]string, 0, 6)
	if evidence.Signal != "" {
		parts = append(parts, evidence.Signal)
	}
	if evidence.Strategy != "" {
		parts = append(parts, evidence.Strategy)
	}
	if evidence.Optimization != "" {
		parts = append(parts, evidence.Optimization)
	}
	if evidence.WalkForward != "" {
		parts = append(parts, evidence.WalkForward)
	}
	if evidence.MonteCarlo != "" {
		parts = append(parts, evidence.MonteCarlo)
	}
	if evidence.Performance != "" {
		parts = append(parts, evidence.Performance)
	}
	return s.fmt.JoinSentences(parts...)
}

// DecisionSummary summarizes the current recommendation decision.
func (s *SummaryBuilder) DecisionSummary(update StateUpdate, level Level) string {
	return s.fmt.JoinSentences(
		fmt.Sprintf("%s on %s timeframe via %s strategy is classified as %s with %s confidence.",
			update.Symbol, update.Timeframe, update.Strategy,
			s.fmt.LevelLabel(level), s.fmt.FormatConfidence(update.Confidence)),
		fmt.Sprintf("Current lifecycle status is %s.", s.fmt.StatusLabel(update.CurrentStatus)),
	)
}

// TimelineSummary builds a readable narrative from timeline entries.
func (s *SummaryBuilder) TimelineSummary(entries []TimelineEntry) string {
	if len(entries) == 0 {
		return ""
	}

	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := formatTimelineLine(entry)
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "Lifecycle: " + strings.Join(lines, " → ") + "."
}

func formatTimelineLine(entry TimelineEntry) string {
	event := strings.TrimSpace(entry.Event)
	if event == "" {
		return ""
	}
	if entry.PreviousValue != "" && entry.NewValue != "" {
		return fmt.Sprintf("%s (%s to %s)", event, entry.PreviousValue, entry.NewValue)
	}
	return event
}

// HistorySummary returns a short list of key lifecycle events.
func (s *SummaryBuilder) HistorySummary(entries []TimelineEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := formatTimelineLine(entry)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func hasResearchEvidence(evidence ResearchEvidence) bool {
	return evidence.Signal != "" || evidence.Strategy != "" || evidence.Risk != "" ||
		evidence.Performance != "" || evidence.Optimization != "" ||
		evidence.WalkForward != "" || evidence.MonteCarlo != "" ||
		evidence.Drawdown != "" || evidence.Freshness != ""
}

// UpdatedAt extracts the most recent timestamp from timeline entries.
func UpdatedAt(entries []TimelineEntry, fallback time.Time) time.Time {
	if len(entries) == 0 {
		return fallback
	}
	latest := entries[len(entries)-1].Timestamp
	if latest.IsZero() {
		return fallback
	}
	return latest
}
