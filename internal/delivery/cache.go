package delivery

import (
	"sort"
	"sync"
	"time"
)

type indexSet map[string]struct{}

// Cache stores delivery documents with secondary indexes.
type Cache struct {
	mu      sync.RWMutex
	builder *Builder
	byID    map[string]*storedDocument

	bySymbol           map[string]indexSet
	byStrategy         map[string]indexSet
	byTimeframe        map[string]indexSet
	byStatus           map[string]indexSet
	byLevel            map[string]indexSet
	byConfidenceBucket map[string]indexSet
	byCreatedDate      map[string]indexSet
	byUpdatedDate      map[string]indexSet
}

// NewCache creates an empty delivery cache.
func NewCache(builder *Builder) *Cache {
	return &Cache{
		builder:            builder,
		byID:               make(map[string]*storedDocument),
		bySymbol:           make(map[string]indexSet),
		byStrategy:         make(map[string]indexSet),
		byTimeframe:        make(map[string]indexSet),
		byStatus:           make(map[string]indexSet),
		byLevel:            make(map[string]indexSet),
		byConfidenceBucket: make(map[string]indexSet),
		byCreatedDate:      make(map[string]indexSet),
		byUpdatedDate:      make(map[string]indexSet),
	}
}

func (c *Cache) getOrCreate(id string) *storedDocument {
	stored, ok := c.byID[id]
	if !ok || stored == nil {
		stored = &storedDocument{}
		c.byID[id] = stored
	}
	return stored
}

func (c *Cache) ApplyState(input StateInput, at time.Time) DeliveryDocument {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := c.getOrCreate(input.RecommendationID)
	c.unindex(input.RecommendationID, stored.document)
	c.builder.ApplyState(stored, input, at)
	c.index(input.RecommendationID, stored.document)
	return c.builder.Build(stored)
}

func (c *Cache) ApplyIntelligence(input IntelligenceInput) DeliveryDocument {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := c.getOrCreate(input.RecommendationID)
	c.unindex(input.RecommendationID, stored.document)
	c.builder.ApplyIntelligence(stored, input)
	c.index(input.RecommendationID, stored.document)
	return c.builder.Build(stored)
}

func (c *Cache) ApplyQuality(input QualityInput) DeliveryDocument {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := c.getOrCreate(input.RecommendationID)
	c.unindex(input.RecommendationID, stored.document)
	c.builder.ApplyQuality(stored, input)
	c.index(input.RecommendationID, stored.document)
	return c.builder.Build(stored)
}

func (c *Cache) ApplyFeedback(input FeedbackInput) []DeliveryDocument {
	c.mu.Lock()
	defer c.mu.Unlock()

	updated := make([]DeliveryDocument, 0, len(c.byID))
	for id, stored := range c.byID {
		if stored == nil || stored.document.RecommendationID == "" {
			continue
		}
		c.unindex(id, stored.document)
		c.builder.ApplyFeedback(stored, input)
		c.index(id, stored.document)
		updated = append(updated, c.builder.Build(stored))
	}
	return updated
}

func (c *Cache) ApplyAlert(input AlertInput) (DeliveryDocument, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored, ok := c.byID[input.RecommendationID]
	if !ok || stored == nil {
		stored = c.getOrCreate(input.RecommendationID)
	}
	c.unindex(input.RecommendationID, stored.document)
	c.builder.ApplyAlert(stored, input)
	c.index(input.RecommendationID, stored.document)
	return c.builder.Build(stored), ok
}

func (c *Cache) Get(id string) (DeliveryDocument, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stored, ok := c.byID[id]
	if !ok || stored == nil || stored.document.RecommendationID == "" {
		return DeliveryDocument{}, false
	}
	return c.builder.Build(stored), true
}

func (c *Cache) List(filter Filter) []DeliveryDocument {
	c.mu.RLock()
	defer c.mu.RUnlock()

	candidates := c.candidateIDs(filter)
	out := make([]DeliveryDocument, 0, len(candidates))
	for _, id := range candidates {
		stored := c.byID[id]
		if stored == nil || stored.document.RecommendationID == "" {
			continue
		}
		doc := stored.document
		if filter.Symbol != "" && doc.Symbol != filter.Symbol {
			continue
		}
		if filter.Strategy != "" && doc.Strategy != filter.Strategy {
			continue
		}
		if filter.Timeframe != "" && doc.Timeframe != filter.Timeframe {
			continue
		}
		if filter.Status != "" && doc.CurrentStatus != filter.Status {
			continue
		}
		if filter.Level != "" && doc.CurrentRecommendationLevel != filter.Level {
			continue
		}
		if filter.ConfidenceMin > 0 && doc.CurrentConfidence < filter.ConfidenceMin {
			continue
		}
		if filter.ConfidenceBucket != "" && ConfidenceBucket(doc.CurrentConfidence) != filter.ConfidenceBucket {
			continue
		}
		if !filter.CreatedAfter.IsZero() && doc.CreatedAt.Before(filter.CreatedAfter) {
			continue
		}
		if !filter.UpdatedAfter.IsZero() && doc.UpdatedAt.Before(filter.UpdatedAfter) {
			continue
		}
		out = append(out, c.builder.Build(stored))
	}
	c.sortDocuments(out, "")
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out
}

func (c *Cache) candidateIDs(filter Filter) []string {
	switch {
	case filter.Symbol != "":
		return setKeys(c.bySymbol[filter.Symbol])
	case filter.Strategy != "":
		return setKeys(c.byStrategy[filter.Strategy])
	case filter.Timeframe != "":
		return setKeys(c.byTimeframe[filter.Timeframe])
	case filter.Status != "":
		return setKeys(c.byStatus[string(filter.Status)])
	case filter.Level != "":
		return setKeys(c.byLevel[string(filter.Level)])
	case filter.ConfidenceBucket != "":
		return setKeys(c.byConfidenceBucket[filter.ConfidenceBucket])
	default:
		return setKeysFromMap(c.byID)
	}
}

func (c *Cache) sortDocuments(docs []DeliveryDocument, mode string) {
	sort.Slice(docs, func(i, j int) bool {
		switch mode {
		case "confidence":
			if docs[i].CurrentConfidence == docs[j].CurrentConfidence {
				return docs[i].UpdatedAt.After(docs[j].UpdatedAt)
			}
			return docs[i].CurrentConfidence > docs[j].CurrentConfidence
		case "newest":
			return docs[i].CreatedAt.After(docs[j].CreatedAt)
		default:
			return docs[i].UpdatedAt.After(docs[j].UpdatedAt)
		}
	})
}

func (c *Cache) Stats() (documents, active, closed, timelineEntries int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	documents = len(c.byID)
	for _, stored := range c.byID {
		if stored == nil {
			continue
		}
		timelineEntries += len(stored.document.Timeline)
		switch stored.document.CurrentStatus {
		case StatusClosed:
			closed++
		default:
			if stored.document.RecommendationID != "" {
				active++
			}
		}
	}
	return documents, active, closed, timelineEntries
}

func (c *Cache) index(id string, doc DeliveryDocument) {
	if doc.RecommendationID == "" {
		return
	}
	addIndex(c.bySymbol, doc.Symbol, id)
	addIndex(c.byStrategy, doc.Strategy, id)
	addIndex(c.byTimeframe, doc.Timeframe, id)
	addIndex(c.byStatus, string(doc.CurrentStatus), id)
	addIndex(c.byLevel, string(doc.CurrentRecommendationLevel), id)
	addIndex(c.byConfidenceBucket, ConfidenceBucket(doc.CurrentConfidence), id)
	if !doc.CreatedAt.IsZero() {
		addIndex(c.byCreatedDate, doc.CreatedAt.UTC().Format("2006-01-02"), id)
	}
	if !doc.UpdatedAt.IsZero() {
		addIndex(c.byUpdatedDate, doc.UpdatedAt.UTC().Format("2006-01-02"), id)
	}
}

func (c *Cache) unindex(id string, doc DeliveryDocument) {
	if doc.RecommendationID == "" {
		return
	}
	removeIndex(c.bySymbol, doc.Symbol, id)
	removeIndex(c.byStrategy, doc.Strategy, id)
	removeIndex(c.byTimeframe, doc.Timeframe, id)
	removeIndex(c.byStatus, string(doc.CurrentStatus), id)
	removeIndex(c.byLevel, string(doc.CurrentRecommendationLevel), id)
	removeIndex(c.byConfidenceBucket, ConfidenceBucket(doc.CurrentConfidence), id)
	if !doc.CreatedAt.IsZero() {
		removeIndex(c.byCreatedDate, doc.CreatedAt.UTC().Format("2006-01-02"), id)
	}
	if !doc.UpdatedAt.IsZero() {
		removeIndex(c.byUpdatedDate, doc.UpdatedAt.UTC().Format("2006-01-02"), id)
	}
}

func addIndex(index map[string]indexSet, key, id string) {
	if key == "" {
		return
	}
	set, ok := index[key]
	if !ok {
		set = make(indexSet)
		index[key] = set
	}
	set[id] = struct{}{}
}

func removeIndex(index map[string]indexSet, key, id string) {
	if key == "" {
		return
	}
	set, ok := index[key]
	if !ok {
		return
	}
	delete(set, id)
	if len(set) == 0 {
		delete(index, key)
	}
}

func setKeys(set indexSet) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func setKeysFromMap(byID map[string]*storedDocument) []string {
	out := make([]string, 0, len(byID))
	for id := range byID {
		out = append(out, id)
	}
	return out
}
