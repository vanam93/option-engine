package recommendationstate

import (
	"fmt"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

const (
	timelineCreated           = "Recommendation Created"
	timelineConfidenceIncreased = "Confidence Increased"
	timelineConfidenceDecreased = "Confidence Decreased"
	timelineStatusChanged     = "Status Changed"
	timelineExitRecommended   = "Exit Recommended"
	timelineClosed            = "Closed"
)

// applyUpdate mutates state from a validated recommendation and returns the latest timeline entry.
func applyUpdate(rec *Recommendation, timeline *[]TimelineEntry, input InputValidated, at time.Time) TimelineEntry {
	if input.ValidationStatus == "REJECTED" {
		return applyRejection(rec, timeline, input, at)
	}
	return applyValidUpdate(rec, timeline, input, at)
}

func applyValidUpdate(rec *Recommendation, timeline *[]TimelineEntry, input InputValidated, at time.Time) TimelineEntry {
	target := mapRecommendationToStatus(input.Recommendation)
	var latest TimelineEntry

	if rec.RecommendationID == "" {
		latest = appendTimeline(timeline, TimelineEntry{
			Timestamp:     at,
			Event:         timelineCreated,
			Reason:        fmt.Sprintf("validated %s recommendation", strings.ToLower(string(input.Recommendation))),
			PreviousValue: "",
			NewValue:      string(target),
		})
		rec.CurrentStatus = StatusCreated
		rec.Confidence = input.Confidence
		rec.UpdatedAt = at

		if target != StatusCreated {
			latest = appendTimeline(timeline, TimelineEntry{
				Timestamp:     at,
				Event:         timelineStatusChanged,
				Reason:        "initial activation from validated recommendation",
				PreviousValue: string(StatusCreated),
				NewValue:      string(target),
			})
			rec.CurrentStatus = target
		}
		return latest
	}

	if input.Confidence > rec.Confidence {
		latest = appendTimeline(timeline, TimelineEntry{
			Timestamp:     at,
			Event:         timelineConfidenceIncreased,
			Reason:        "validated recommendation confidence increased",
			PreviousValue: fmt.Sprintf("%.4f", rec.Confidence),
			NewValue:      fmt.Sprintf("%.4f", input.Confidence),
		})
	} else if input.Confidence < rec.Confidence {
		latest = appendTimeline(timeline, TimelineEntry{
			Timestamp:     at,
			Event:         timelineConfidenceDecreased,
			Reason:        "validated recommendation confidence decreased",
			PreviousValue: fmt.Sprintf("%.4f", rec.Confidence),
			NewValue:      fmt.Sprintf("%.4f", input.Confidence),
		})
	}
	rec.Confidence = input.Confidence
	rec.UpdatedAt = at

	if target == StatusExitRecommended && rec.CurrentStatus != StatusExitRecommended && rec.CurrentStatus != StatusClosed {
		latest = appendTimeline(timeline, TimelineEntry{
			Timestamp:     at,
			Event:         timelineExitRecommended,
			Reason:        "validated recommendation suggests exit",
			PreviousValue: string(rec.CurrentStatus),
			NewValue:      string(StatusExitRecommended),
		})
		rec.CurrentStatus = StatusExitRecommended
		return latest
	}

	if target != rec.CurrentStatus && rec.CurrentStatus != StatusClosed {
		latest = appendTimeline(timeline, TimelineEntry{
			Timestamp:     at,
			Event:         timelineStatusChanged,
			Reason:        "validated recommendation status changed",
			PreviousValue: string(rec.CurrentStatus),
			NewValue:      string(target),
		})
		rec.CurrentStatus = target
	}

	if latest.Event == "" {
		latest = lastTimelineEntry(*timeline)
	}
	return latest
}

func applyRejection(rec *Recommendation, timeline *[]TimelineEntry, input InputValidated, at time.Time) TimelineEntry {
	if rec.RecommendationID == "" {
		return TimelineEntry{}
	}
	if rec.CurrentStatus == StatusClosed {
		return lastTimelineEntry(*timeline)
	}

	reason := "validation rejected"
	if len(input.RejectionReasons) > 0 {
		reason = strings.Join(input.RejectionReasons, "; ")
	}
	latest := appendTimeline(timeline, TimelineEntry{
		Timestamp:     at,
		Event:         timelineClosed,
		Reason:        reason,
		PreviousValue: string(rec.CurrentStatus),
		NewValue:      string(StatusClosed),
	})
	rec.CurrentStatus = StatusClosed
	rec.UpdatedAt = at
	rec.ClosedAt = &at
	return latest
}

func mapRecommendationToStatus(level recommendation.Level) Status {
	switch level {
	case recommendation.LevelStrongBuy, recommendation.LevelBuy:
		return StatusActive
	case recommendation.LevelWatch:
		return StatusWatch
	case recommendation.LevelAvoid:
		return StatusExitRecommended
	default:
		return StatusWatch
	}
}

func buildSummary(rec Recommendation, entry TimelineEntry) string {
	return fmt.Sprintf("%s %s %s %s: %s (confidence %.2f) — %s",
		rec.Symbol,
		rec.Timeframe,
		rec.Strategy,
		rec.CurrentStatus,
		entry.Event,
		rec.Confidence,
		entry.Reason,
	)
}

func appendTimeline(timeline *[]TimelineEntry, entry TimelineEntry) TimelineEntry {
	*timeline = append(*timeline, entry)
	return entry
}

func lastTimelineEntry(entries []TimelineEntry) TimelineEntry {
	if len(entries) == 0 {
		return TimelineEntry{}
	}
	return entries[len(entries)-1]
}
