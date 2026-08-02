package intelligence

import (
	"sync"
	"time"
)

type storedSnapshot struct {
	document   IntelligenceDocument
	timeline   []TimelineEntry
	level      Level
	status     Status
	confidence float64
}

// Cache stores the latest intelligence document per recommendation ID.
type Cache struct {
	mu   sync.RWMutex
	byID map[string]*storedSnapshot
}

// NewCache creates an empty intelligence cache.
func NewCache() *Cache {
	return &Cache{byID: make(map[string]*storedSnapshot)}
}

// Apply stores the latest intelligence document and timeline.
func (c *Cache) Apply(doc IntelligenceDocument, timeline []TimelineEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byID[doc.RecommendationID] = &storedSnapshot{
		document:   doc,
		timeline:   append([]TimelineEntry(nil), timeline...),
		level:      doc.RecommendationLevel,
		status:     doc.CurrentStatus,
		confidence: doc.Confidence,
	}
}

// Peek returns a copy of the stored snapshot without modifying the cache.
func (c *Cache) Peek(id string) *storedSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stored, ok := c.byID[id]
	if !ok || stored == nil {
		return nil
	}
	copyStored := *stored
	copyStored.timeline = append([]TimelineEntry(nil), stored.timeline...)
	return &copyStored
}

// Get returns the latest intelligence document for a recommendation ID.
func (c *Cache) Get(id string) (IntelligenceDocument, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stored, ok := c.byID[id]
	if !ok || stored == nil {
		return IntelligenceDocument{}, false
	}
	return stored.document, true
}

// Stats returns cache counters for health reporting.
func (c *Cache) Stats() (documents int, averageConfidence float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	documents = len(c.byID)
	if documents == 0 {
		return 0, 0
	}
	var total float64
	for _, stored := range c.byID {
		if stored == nil {
			continue
		}
		total += stored.confidence
	}
	return documents, total / float64(documents)
}

// appendTimelineEntry adds a timeline entry if it is new.
func appendTimelineEntry(existing []TimelineEntry, entry TimelineEntry) []TimelineEntry {
	if entry.Event == "" {
		return existing
	}
	for _, item := range existing {
		if item.Event == entry.Event && item.Timestamp.Equal(entry.Timestamp) &&
			item.PreviousValue == entry.PreviousValue && item.NewValue == entry.NewValue {
			return existing
		}
	}
	return append(existing, entry)
}

// mergeTimeline merges prior timeline with the latest entry from a state update.
func mergeTimeline(previous []TimelineEntry, latest TimelineEntry) []TimelineEntry {
	out := append([]TimelineEntry(nil), previous...)
	return appendTimelineEntry(out, latest)
}

// freshnessLabel returns a human-readable freshness description.
func freshnessLabel(updatedAt time.Time, now time.Time) string {
	if updatedAt.IsZero() {
		return ""
	}
	age := now.Sub(updatedAt)
	if age < 0 {
		age = 0
	}
	switch {
	case age < time.Minute:
		return "Recommendation is fresh (updated moments ago)."
	case age < 5*time.Minute:
		return "Recommendation is fresh (updated within the last few minutes)."
	case age < 30*time.Minute:
		return "Recommendation was updated recently."
	default:
		return "Recommendation may be aging; verify current market conditions."
	}
}
