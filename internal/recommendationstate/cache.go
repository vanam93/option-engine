package recommendationstate

import (
	"fmt"
	"sync"
	"time"
)

type compositeKey struct {
	symbol    string
	timeframe string
	strategy  string
}

// Cache stores active and closed recommendations with timelines and indexes.
type Cache struct {
	mu sync.RWMutex

	maxActive int

	active map[compositeKey]string
	closed map[compositeKey]string

	byID       map[string]*storedRecommendation
	bySymbol   map[string]map[string]struct{}
	byStrategy map[string]map[string]struct{}

	seqBySymbolDate map[string]uint64
}

type storedRecommendation struct {
	recommendation Recommendation
	timeline       []TimelineEntry
}

// NewCache creates an empty recommendation state cache.
func NewCache(maxActive int) *Cache {
	if maxActive < 1 {
		maxActive = 10000
	}
	return &Cache{
		maxActive:       maxActive,
		active:          make(map[compositeKey]string),
		closed:          make(map[compositeKey]string),
		byID:            make(map[string]*storedRecommendation),
		bySymbol:        make(map[string]map[string]struct{}),
		byStrategy:      make(map[string]map[string]struct{}),
		seqBySymbolDate: make(map[string]uint64),
	}
}

// ApplyValidated merges a validated recommendation into state.
// Returns the updated recommendation, latest timeline entry, and whether this was a duplicate merge.
func (c *Cache) ApplyValidated(input InputValidated, at time.Time) (Recommendation, TimelineEntry, bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := compositeKey{symbol: input.Symbol, timeframe: input.Timeframe, strategy: input.Strategy}
	id, exists := c.active[key]
	if !exists {
		if closedID, ok := c.closed[key]; ok {
			id = closedID
			exists = true
		}
	}

	if !exists {
		if input.ValidationStatus != "VALID" {
			return Recommendation{}, TimelineEntry{}, false, false
		}
		if len(c.active) >= c.maxActive {
			return Recommendation{}, TimelineEntry{}, false, false
		}
		rec := Recommendation{
			Symbol:    input.Symbol,
			Timeframe: input.Timeframe,
			Strategy:  input.Strategy,
			CreatedAt: at,
			UpdatedAt: at,
		}
		timeline := make([]TimelineEntry, 0, 4)
		latest := applyUpdate(&rec, &timeline, input, at)
		if latest.Event == "" {
			return Recommendation{}, TimelineEntry{}, false, false
		}
		rec.RecommendationID = c.nextID(input.Symbol, at)

		stored := &storedRecommendation{recommendation: rec, timeline: timeline}
		c.byID[rec.RecommendationID] = stored
		c.active[key] = rec.RecommendationID
		c.index(rec.RecommendationID, rec.Symbol, rec.Strategy)
		return rec, latest, false, true
	}

	stored := c.byID[id]
	if stored == nil {
		return Recommendation{}, TimelineEntry{}, false, false
	}

	rec := stored.recommendation
	wasClosed := rec.CurrentStatus == StatusClosed
	latest := applyUpdate(&rec, &stored.timeline, input, at)
	if latest.Event == "" {
		return Recommendation{}, TimelineEntry{}, true, false
	}

	stored.recommendation = rec
	if rec.CurrentStatus == StatusClosed {
		delete(c.active, key)
		c.closed[key] = rec.RecommendationID
	} else if wasClosed {
		delete(c.closed, key)
		c.active[key] = rec.RecommendationID
	}

	return rec, latest, true, true
}

func (c *Cache) nextID(symbol string, at time.Time) string {
	dateKey := fmt.Sprintf("%s:%s", at.UTC().Format("20060102"), symbol)
	c.seqBySymbolDate[dateKey]++
	seq := c.seqBySymbolDate[dateKey]
	return fmt.Sprintf("REC-%s-%s-%06d", at.UTC().Format("20060102"), symbol, seq)
}

func (c *Cache) index(id, symbol, strategy string) {
	if _, ok := c.bySymbol[symbol]; !ok {
		c.bySymbol[symbol] = make(map[string]struct{})
	}
	c.bySymbol[symbol][id] = struct{}{}

	if _, ok := c.byStrategy[strategy]; !ok {
		c.byStrategy[strategy] = make(map[string]struct{})
	}
	c.byStrategy[strategy][id] = struct{}{}
}

// GetByID returns a recommendation and its timeline by ID.
func (c *Cache) GetByID(id string) (Recommendation, []TimelineEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stored, ok := c.byID[id]
	if !ok || stored == nil {
		return Recommendation{}, nil, false
	}
	timeline := append([]TimelineEntry(nil), stored.timeline...)
	return stored.recommendation, timeline, true
}

// Stats returns cache counters for health reporting.
func (c *Cache) Stats() (active, closed, timelineEntries int, averageConfidence float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active = len(c.active)
	closed = len(c.closed)

	var totalConfidence float64
	var count int
	for _, stored := range c.byID {
		if stored == nil {
			continue
		}
		count++
		totalConfidence += stored.recommendation.Confidence
		for range stored.timeline {
			timelineEntries++
		}
	}
	if count > 0 {
		averageConfidence = totalConfidence / float64(count)
	}
	return active, closed, timelineEntries, averageConfidence
}
