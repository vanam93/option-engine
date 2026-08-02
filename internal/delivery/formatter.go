package delivery

import (
	"fmt"
	"strconv"
)

// Formatter produces human-readable delivery labels.
type Formatter struct{}

// NewFormatter creates a delivery formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// StatusLabel returns a display label for a lifecycle status.
func (f *Formatter) StatusLabel(status Status) string {
	switch status {
	case StatusCreated:
		return "Created"
	case StatusActive:
		return "Active"
	case StatusWatch:
		return "Watch"
	case StatusExitRecommended:
		return "Exit Recommended"
	case StatusClosed:
		return "Closed"
	default:
		return string(status)
	}
}

// LevelLabel returns a display label for a recommendation level.
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

// ConfidenceBucket returns the bucket label for a confidence value.
func ConfidenceBucket(confidence float64) string {
	switch {
	case confidence >= 0.95:
		return "0.95-1.00"
	case confidence >= 0.90:
		return "0.90-0.95"
	case confidence >= 0.80:
		return "0.80-0.90"
	case confidence >= 0.70:
		return "0.70-0.80"
	case confidence >= 0.60:
		return "0.60-0.70"
	default:
		return "below-0.60"
	}
}

func formatPrice(price float64) string {
	return strconv.FormatFloat(price, 'f', 2, 64)
}

// DocumentSummary returns a short summary for a delivery document.
func (f *Formatter) DocumentSummary(doc DeliveryDocument) string {
	return fmt.Sprintf("%s %s %s %s: %s (confidence %.2f)",
		doc.Symbol,
		doc.Timeframe,
		doc.Strategy,
		f.StatusLabel(doc.CurrentStatus),
		f.LevelLabel(doc.CurrentRecommendationLevel),
		doc.CurrentConfidence,
	)
}
