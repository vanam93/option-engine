package alerts

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	timelineCreated             = "Recommendation Created"
	timelineConfidenceIncreased = "Confidence Increased"
	timelineConfidenceDecreased = "Confidence Decreased"
	timelineStatusChanged       = "Status Changed"
	timelineExitRecommended     = "Exit Recommended"
	timelineClosed              = "Closed"
)

// evaluateAlerts derives meaningful lifecycle alerts from a recommendation state update.
func evaluateAlerts(update StateUpdate, cfg Config, firstSeen bool) []candidateAlert {
	var alerts []candidateAlert
	entry := update.LatestTimelineEntry

	if firstSeen {
		alerts = append(alerts, candidateAlert{
			AlertType: AlertRecommendationCreated,
			Message:   fmt.Sprintf("New recommendation for %s %s", update.Symbol, update.Timeframe),
			Reason:    entry.Reason,
		})
	}

	switch entry.Event {
	case timelineConfidenceIncreased:
		if confidenceDelta(entry) >= cfg.ConfidenceChangeThreshold {
			alerts = append(alerts, candidateAlert{
				AlertType: AlertConfidenceIncreased,
				Message:   fmt.Sprintf("Confidence increased for %s %s", update.Symbol, update.Timeframe),
				Reason:    entry.Reason,
			})
		}
	case timelineConfidenceDecreased:
		if confidenceDelta(entry) >= cfg.ConfidenceChangeThreshold {
			alerts = append(alerts, candidateAlert{
				AlertType: AlertConfidenceDecreased,
				Message:   fmt.Sprintf("Confidence decreased for %s %s", update.Symbol, update.Timeframe),
				Reason:    entry.Reason,
			})
		}
	case timelineExitRecommended:
		alerts = append(alerts, candidateAlert{
			AlertType: AlertExitRecommended,
			Message:   fmt.Sprintf("Exit recommended for %s %s", update.Symbol, update.Timeframe),
			Reason:    entry.Reason,
		})
	case timelineClosed:
		alerts = append(alerts, candidateAlert{
			AlertType: AlertRecommendationClosed,
			Message:   fmt.Sprintf("Recommendation closed for %s %s", update.Symbol, update.Timeframe),
			Reason:    entry.Reason,
		})
	case timelineStatusChanged:
		if strings.EqualFold(entry.NewValue, string(StatusActive)) {
			alerts = append(alerts, candidateAlert{
				AlertType: AlertEntryZoneReached,
				Message:   fmt.Sprintf("Entry zone reached for %s %s", update.Symbol, update.Timeframe),
				Reason:    entry.Reason,
			})
		} else if isMeaningfulStatusTransition(entry.PreviousValue, entry.NewValue) {
			alerts = append(alerts, candidateAlert{
				AlertType: AlertStatusChanged,
				Message:   fmt.Sprintf("Status changed to %s for %s %s", entry.NewValue, update.Symbol, update.Timeframe),
				Reason:    entry.Reason,
			})
		}
	}

	return alerts
}

func confidenceDelta(entry TimelineEntry) float64 {
	prev, err1 := strconv.ParseFloat(entry.PreviousValue, 64)
	next, err2 := strconv.ParseFloat(entry.NewValue, 64)
	if err1 != nil || err2 != nil {
		return 0
	}
	return math.Abs(next - prev)
}

func isMeaningfulStatusTransition(previous, next string) bool {
	previous = strings.ToUpper(strings.TrimSpace(previous))
	next = strings.ToUpper(strings.TrimSpace(next))
	if previous == "" || next == "" || previous == next {
		return false
	}
	if next == string(StatusActive) {
		return false
	}
	return true
}
