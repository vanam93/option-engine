package delivery

import "sync/atomic"

// Repository exposes read-only delivery document queries.
type Repository struct {
	cache       *Cache
	cacheHits   uint64
	cacheMisses uint64
}

// NewRepository creates a delivery repository over the cache.
func NewRepository(cache *Cache) *Repository {
	return &Repository{cache: cache}
}

// GetRecommendation returns a delivery document by recommendation ID.
func (r *Repository) GetRecommendation(id string) (DeliveryDocument, bool) {
	doc, ok := r.cache.Get(id)
	if ok {
		atomic.AddUint64(&r.cacheHits, 1)
	} else {
		atomic.AddUint64(&r.cacheMisses, 1)
	}
	return doc, ok
}

// ListRecommendations returns documents matching the filter.
func (r *Repository) ListRecommendations(filter Filter) []DeliveryDocument {
	return r.cache.List(filter)
}

// ListBySymbol returns documents for a symbol ordered by latest update.
func (r *Repository) ListBySymbol(symbol string) []DeliveryDocument {
	return r.cache.List(Filter{Symbol: symbol})
}

// ListByStrategy returns documents for a strategy ordered by latest update.
func (r *Repository) ListByStrategy(strategy string) []DeliveryDocument {
	return r.cache.List(Filter{Strategy: strategy})
}

// ListActive returns non-closed documents ordered by latest update.
func (r *Repository) ListActive() []DeliveryDocument {
	all := r.cache.List(Filter{})
	out := make([]DeliveryDocument, 0, len(all))
	for _, doc := range all {
		if doc.CurrentStatus == StatusClosed {
			continue
		}
		out = append(out, doc)
	}
	return out
}

// ListClosed returns closed documents ordered by latest update.
func (r *Repository) ListClosed() []DeliveryDocument {
	return r.cache.List(Filter{Status: StatusClosed})
}

// LatestRecommendations returns documents ordered by latest update.
func (r *Repository) LatestRecommendations(limit int) []DeliveryDocument {
	return r.cache.List(Filter{Limit: limit})
}

// HighestConfidence returns documents ordered by confidence descending.
func (r *Repository) HighestConfidence(limit int) []DeliveryDocument {
	all := r.cache.List(Filter{})
	r.cache.sortDocuments(all, "confidence")
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// Newest returns documents ordered by creation time descending.
func (r *Repository) Newest(limit int) []DeliveryDocument {
	all := r.cache.List(Filter{})
	r.cache.sortDocuments(all, "newest")
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// Stats returns cache hit/miss counters.
func (r *Repository) Stats() (hits, misses uint64) {
	return atomic.LoadUint64(&r.cacheHits), atomic.LoadUint64(&r.cacheMisses)
}
