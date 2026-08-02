package quality

import (
	"sync"
	"time"
)

// activeTracker holds mutable tracking state for one recommendation.
type activeTracker struct {
	recommendationID  string
	symbol            string
	timeframe         string
	strategy          string
	level             Level
	confidence        float64
	status            Status
	entryTime         time.Time
	entryPrice        float64
	currentPrice      float64
	highestPrice      float64
	lowestPrice       float64
	hasPrice          bool
	latestCandleClose time.Time
	startedAt         time.Time
	lastUpdatedAt     time.Time
}

func newActiveTracker(input IntelligenceInput, at time.Time) *activeTracker {
	entryAt := input.GeneratedAt
	if entryAt.IsZero() {
		entryAt = at
	}
	return &activeTracker{
		recommendationID: input.RecommendationID,
		symbol:           input.Symbol,
		timeframe:        input.Timeframe,
		strategy:         input.Strategy,
		level:            input.RecommendationLevel,
		confidence:       input.Confidence,
		status:           input.CurrentStatus,
		entryTime:        entryAt,
		startedAt:        at,
		lastUpdatedAt:    at,
	}
}

func (t *activeTracker) applyIntelligence(input IntelligenceInput, at time.Time) {
	if input.RecommendationLevel != "" {
		t.level = input.RecommendationLevel
	}
	if input.Confidence > 0 {
		t.confidence = input.Confidence
	}
	if input.CurrentStatus != "" {
		t.status = input.CurrentStatus
	}
	if input.Strategy != "" {
		t.strategy = input.Strategy
	}
	if !input.GeneratedAt.IsZero() && t.entryTime.IsZero() {
		t.entryTime = input.GeneratedAt
	}
	t.lastUpdatedAt = at
}

func (t *activeTracker) applyState(input StateInput, at time.Time) {
	if input.CurrentStatus != "" {
		t.status = input.CurrentStatus
	}
	if input.Confidence > 0 {
		t.confidence = input.Confidence
	}
	t.lastUpdatedAt = at
}

func (t *activeTracker) applyCandle(update CandleUpdate) bool {
	if update.Symbol != t.symbol {
		return false
	}
	if t.timeframe != "" && update.Timeframe != "" && update.Timeframe != t.timeframe {
		return false
	}

	price := update.Candle.Close
	if price <= 0 {
		return false
	}

	if !t.hasPrice {
		t.entryPrice = price
		t.currentPrice = price
		t.highestPrice = price
		t.lowestPrice = price
		t.hasPrice = true
		if t.entryTime.IsZero() {
			t.entryTime = update.At
		}
	} else {
		t.currentPrice = price
		if price > t.highestPrice {
			t.highestPrice = price
		}
		if price < t.lowestPrice {
			t.lowestPrice = price
		}
	}

	if !update.Candle.CloseTime.IsZero() {
		t.latestCandleClose = update.Candle.CloseTime
	}
	t.lastUpdatedAt = update.At
	return true
}

func (t *activeTracker) isTimedOut(timeout time.Duration, now time.Time) bool {
	if timeout <= 0 {
		return false
	}
	start := t.entryTime
	if start.IsZero() {
		start = t.startedAt
	}
	return now.Sub(start) >= timeout
}

func (t *activeTracker) holdingDuration(at time.Time) time.Duration {
	start := t.entryTime
	if start.IsZero() {
		start = t.startedAt
	}
	if at.Before(start) {
		return 0
	}
	return at.Sub(start)
}

func (t *activeTracker) snapshot() activeTracker {
	return *t
}

// TrackerRegistry manages active recommendation trackers.
type TrackerRegistry struct {
	mu       sync.RWMutex
	trackers map[string]*activeTracker
}

// NewTrackerRegistry creates an empty tracker registry.
func NewTrackerRegistry() *TrackerRegistry {
	return &TrackerRegistry{trackers: make(map[string]*activeTracker)}
}

// Start begins tracking a recommendation if not already active.
func (r *TrackerRegistry) Start(input IntelligenceInput, at time.Time) *activeTracker {
	if input.RecommendationID == "" || input.CurrentStatus == StatusClosed {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.trackers[input.RecommendationID]; ok {
		existing.applyIntelligence(input, at)
		return existing
	}

	tracker := newActiveTracker(input, at)
	r.trackers[input.RecommendationID] = tracker
	return tracker
}

// Get returns an active tracker by recommendation ID.
func (r *TrackerRegistry) Get(id string) (*activeTracker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tracker, ok := r.trackers[id]
	return tracker, ok
}

// ApplyState updates tracker status from a state event.
func (r *TrackerRegistry) ApplyState(input StateInput, at time.Time) *activeTracker {
	r.mu.Lock()
	defer r.mu.Unlock()

	tracker, ok := r.trackers[input.RecommendationID]
	if !ok {
		return nil
	}
	tracker.applyState(input, at)
	return tracker
}

// ApplyCandle updates all matching active trackers and returns updated IDs.
func (r *TrackerRegistry) ApplyCandle(update CandleUpdate) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	updated := make([]string, 0)
	for id, tracker := range r.trackers {
		if tracker.applyCandle(update) {
			updated = append(updated, id)
		}
	}
	return updated
}

// Remove deletes an active tracker and returns it.
func (r *TrackerRegistry) Remove(id string) (*activeTracker, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tracker, ok := r.trackers[id]
	if !ok {
		return nil, false
	}
	delete(r.trackers, id)
	return tracker, true
}

// ActiveIDs returns all active tracker recommendation IDs.
func (r *TrackerRegistry) ActiveIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.trackers))
	for id := range r.trackers {
		ids = append(ids, id)
	}
	return ids
}

// ActiveCount returns the number of active trackers.
func (r *TrackerRegistry) ActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.trackers)
}

// CheckTimeouts returns trackers that exceeded the tracking timeout.
func (r *TrackerRegistry) CheckTimeouts(timeout time.Duration, now time.Time) []*activeTracker {
	r.mu.Lock()
	defer r.mu.Unlock()

	expired := make([]*activeTracker, 0)
	for id, tracker := range r.trackers {
		if tracker.isTimedOut(timeout, now) {
			expired = append(expired, tracker)
			delete(r.trackers, id)
		}
	}
	return expired
}
