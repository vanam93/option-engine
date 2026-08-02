package laboratory

import (
	"sync"
	"time"
)

// Repository stores completed research studies and comparisons.
type Repository struct {
	mu          sync.RWMutex
	studies     map[string]Study
	comparisons map[string]Comparison
	studyOrder  []string
	compareOrder []string
}

// NewRepository creates an empty study repository.
func NewRepository() *Repository {
	return &Repository{
		studies:     make(map[string]Study),
		comparisons: make(map[string]Comparison),
	}
}

// SaveStudy stores a study record.
func (r *Repository) SaveStudy(study Study) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.studies[study.StudyID]; !exists {
		r.studyOrder = append(r.studyOrder, study.StudyID)
	}
	r.studies[study.StudyID] = study
}

// GetStudy returns a study by ID.
func (r *Repository) GetStudy(id string) (Study, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	study, ok := r.studies[id]
	return study, ok
}

// LatestStudy returns the most recently created study.
func (r *Repository) LatestStudy() (Study, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.studyOrder) - 1; i >= 0; i-- {
		if study, ok := r.studies[r.studyOrder[i]]; ok {
			return study, true
		}
	}
	return Study{}, false
}

// ListStudies returns all studies in creation order.
func (r *Repository) ListStudies() []Study {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Study, 0, len(r.studyOrder))
	for _, id := range r.studyOrder {
		if study, ok := r.studies[id]; ok {
			out = append(out, study)
		}
	}
	return out
}

// ListByStrategy returns studies matching a strategy name.
func (r *Repository) ListByStrategy(strategy string) []Study {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Study, 0)
	for _, id := range r.studyOrder {
		study, ok := r.studies[id]
		if !ok || study.Strategy != strategy {
			continue
		}
		out = append(out, study)
	}
	return out
}

// ListByVersion returns studies for a research version.
func (r *Repository) ListByVersion(version string) []Study {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Study, 0)
	for _, id := range r.studyOrder {
		study, ok := r.studies[id]
		if !ok || study.ResearchVersion != version {
			continue
		}
		out = append(out, study)
	}
	return out
}

// NextVersion returns the next research version for a strategy and parameter set.
func (r *Repository) NextVersion(strategy string, params map[string]string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paramKey := parameterKey(params)
	maxVersion := 0
	for _, study := range r.studies {
		if study.Strategy != strategy {
			continue
		}
		if parameterKey(study.Parameters) != paramKey {
			continue
		}
		version := parseVersionNumber(study.ResearchVersion)
		if version > maxVersion {
			maxVersion = version
		}
	}
	return formatVersion(maxVersion + 1)
}

// SaveComparison stores a comparison record.
func (r *Repository) SaveComparison(comparison Comparison) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.comparisons[comparison.ComparisonID]; !exists {
		r.compareOrder = append(r.compareOrder, comparison.ComparisonID)
	}
	r.comparisons[comparison.ComparisonID] = comparison
}

// GetComparison returns a comparison by ID.
func (r *Repository) GetComparison(id string) (Comparison, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	comparison, ok := r.comparisons[id]
	return comparison, ok
}

// ListComparisons returns all stored comparisons.
func (r *Repository) ListComparisons() []Comparison {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Comparison, 0, len(r.compareOrder))
	for _, id := range r.compareOrder {
		if comparison, ok := r.comparisons[id]; ok {
			out = append(out, comparison)
		}
	}
	return out
}

// Count returns the number of stored studies.
func (r *Repository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.studies)
}

// ComparisonCount returns the number of stored comparisons.
func (r *Repository) ComparisonCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.comparisons)
}

func parseVersionNumber(version string) int {
	if version == "" {
		return 0
	}
	n := 0
	for _, ch := range version {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

func formatVersion(n int) string {
	if n <= 0 {
		return "v1"
	}
	return "v" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// overlapsDateRange reports whether a study period overlaps the given range.
func overlapsDateRange(study Study, start, end time.Time) bool {
	if start.IsZero() && end.IsZero() {
		return true
	}
	if !start.IsZero() && study.EndTime.Before(start) {
		return false
	}
	if !end.IsZero() && study.StartTime.After(end) {
		return false
	}
	return true
}
