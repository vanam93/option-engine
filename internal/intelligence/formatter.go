package intelligence

import (
	"fmt"
	"strings"
)

// Formatter produces human-readable intelligence text.
type Formatter struct{}

// NewFormatter creates an intelligence formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// LevelLabel returns a display-friendly recommendation level.
func (f *Formatter) LevelLabel(level Level) string {
	switch level {
	case LevelStrongBuy:
		return "Strong Buy"
	case LevelBuy:
		return "Buy"
	case LevelWatch:
		return "Watch"
	case LevelAvoid:
		return "Avoid"
	default:
		return string(level)
	}
}

// StatusLabel returns a display-friendly lifecycle status.
func (f *Formatter) StatusLabel(status Status) string {
	switch status {
	case StatusActive:
		return "Active"
	case StatusWatch:
		return "Watch"
	case StatusExitRecommended:
		return "Exit Recommended"
	case StatusClosed:
		return "Closed"
	case StatusCreated:
		return "Created"
	default:
		return string(status)
	}
}

// FormatConfidence formats confidence as a percentage string.
func (f *Formatter) FormatConfidence(confidence float64) string {
	return fmt.Sprintf("%.0f%%", confidence*100)
}

// FormatTransition formats an upgrade or downgrade transition.
func (f *Formatter) FormatTransition(from, to Level) string {
	return fmt.Sprintf("%s → %s", f.LevelLabel(from), f.LevelLabel(to))
}

// FormatStatusTransition formats a status change.
func (f *Formatter) FormatStatusTransition(from, to Status) string {
	return fmt.Sprintf("%s → %s", f.StatusLabel(from), f.StatusLabel(to))
}

// JoinSentences joins non-empty sentences into a paragraph.
func (f *Formatter) JoinSentences(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " ")
}

// StrengthLabel returns a qualitative label for a score.
func StrengthLabel(score float64) string {
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
