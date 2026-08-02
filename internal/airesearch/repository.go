package airesearch

import "sync"

// Repository stores generated research reports in memory.
type Repository struct {
	mu     sync.RWMutex
	items  map[string]ResearchReport
	order  []string
}

// NewRepository creates an empty report repository.
func NewRepository() *Repository {
	return &Repository{
		items: make(map[string]ResearchReport),
	}
}

// Save stores a research report.
func (r *Repository) Save(report ResearchReport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[report.ReportID]; !exists {
		r.order = append(r.order, report.ReportID)
	}
	r.items[report.ReportID] = report
}

// Get returns a report by ID.
func (r *Repository) Get(id string) (ResearchReport, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	report, ok := r.items[id]
	return report, ok
}

// Latest returns the most recently stored report.
func (r *Repository) Latest() (ResearchReport, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		if report, ok := r.items[r.order[i]]; ok {
			return report, true
		}
	}
	return ResearchReport{}, false
}

// List returns all reports in generation order.
func (r *Repository) List() []ResearchReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ResearchReport, 0, len(r.order))
	for _, id := range r.order {
		if report, ok := r.items[id]; ok {
			out = append(out, report)
		}
	}
	return out
}

// Count returns the number of stored reports.
func (r *Repository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}
