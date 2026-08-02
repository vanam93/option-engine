package delivery

import (
	"strings"
	"time"
)

func mapStateTimelineEvent(event string) TimelineEventType {
	switch strings.TrimSpace(event) {
	case "Recommendation Created":
		return TimelineCreated
	case "Confidence Increased":
		return TimelineConfidenceIncreased
	case "Confidence Decreased":
		return TimelineConfidenceDecreased
	case "Status Changed":
		return TimelineStatusChanged
	case "Exit Recommended":
		return TimelineExitRecommended
	case "Closed":
		return TimelineClosed
	default:
		return TimelineEventType(strings.TrimSpace(event))
	}
}

func appendTimelineEntry(existing []TimelineEntry, entry TimelineEntry) []TimelineEntry {
	if entry.Event == "" {
		return existing
	}
	for _, item := range existing {
		if item.Event == entry.Event &&
			item.Timestamp.Equal(entry.Timestamp) &&
			item.PreviousValue == entry.PreviousValue &&
			item.NewValue == entry.NewValue &&
			item.Reason == entry.Reason {
			return existing
		}
	}
	return append(existing, entry)
}

func mergeStateTimeline(previous []TimelineEntry, latest stateTimelineEntry) []TimelineEntry {
	if latest.Event == "" {
		return previous
	}
	entry := TimelineEntry{
		Timestamp:     latest.Timestamp,
		Event:         mapStateTimelineEvent(latest.Event),
		Reason:        latest.Reason,
		PreviousValue: latest.PreviousValue,
		NewValue:      latest.NewValue,
	}
	return appendTimelineEntry(previous, entry)
}

func appendAlertTimeline(existing []TimelineEntry, alert AlertInput) []TimelineEntry {
	entry := TimelineEntry{
		Timestamp: alert.GeneratedAt,
		Event:     TimelineAlertGenerated,
		Reason:    alert.Reason,
		NewValue:  alert.AlertType,
	}
	return appendTimelineEntry(existing, entry)
}

func appendQualityTimeline(existing []TimelineEntry, at time.Time, outcome string) []TimelineEntry {
	entry := TimelineEntry{
		Timestamp: at,
		Event:     TimelineQualityEvaluated,
		NewValue:  outcome,
	}
	return appendTimelineEntry(existing, entry)
}

func appendEntryTriggeredTimeline(existing []TimelineEntry, at time.Time, price float64) []TimelineEntry {
	entry := TimelineEntry{
		Timestamp: at,
		Event:     TimelineEntryTriggered,
		NewValue:  formatPrice(price),
	}
	return appendTimelineEntry(existing, entry)
}

func appendFeedbackTimeline(existing []TimelineEntry, at time.Time) []TimelineEntry {
	entry := TimelineEntry{
		Timestamp: at,
		Event:     TimelineFeedbackUpdated,
	}
	return appendTimelineEntry(existing, entry)
}

func sortTimeline(entries []TimelineEntry) []TimelineEntry {
	if len(entries) < 2 {
		return entries
	}
	out := append([]TimelineEntry(nil), entries...)
	for i := 1; i < len(out); i++ {
		key := out[i]
		j := i - 1
		for j >= 0 && out[j].Timestamp.After(key.Timestamp) {
			out[j+1] = out[j]
			j--
		}
		out[j+1] = key
	}
	return out
}
