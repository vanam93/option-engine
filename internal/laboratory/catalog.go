package laboratory

import (
	"strings"
	"sync"
)

// Catalog maintains lookup indexes for completed research studies.
type Catalog struct {
	mu sync.RWMutex

	studies    map[string]string
	strategies map[string][]string
	symbols    map[string][]string
	timeframes map[string][]string
	versions   map[string][]string
	statuses   map[StudyStatus][]string
}

// NewCatalog creates an empty study catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		studies:    make(map[string]string),
		strategies: make(map[string][]string),
		symbols:    make(map[string][]string),
		timeframes: make(map[string][]string),
		versions:   make(map[string][]string),
		statuses:   make(map[StudyStatus][]string),
	}
}

// Index registers a study in all catalog indexes.
func (c *Catalog) Index(study Study) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.studies[study.StudyID] = study.StudyID
	appendUniqueString(&c.strategies, study.Strategy, study.StudyID)
	appendUniqueString(&c.versions, study.ResearchVersion, study.StudyID)
	appendUniqueStatus(&c.statuses, study.Status, study.StudyID)

	for _, symbol := range study.Symbols {
		appendUniqueString(&c.symbols, symbol, study.StudyID)
	}
	for _, tf := range study.Timeframes {
		appendUniqueString(&c.timeframes, string(tf), study.StudyID)
	}
}

// UpdateStatus moves a study between status indexes.
func (c *Catalog) UpdateStatus(studyID string, from, to StudyStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses[from] = removeFromIndex(c.statuses[from], studyID)
	appendUniqueStatus(&c.statuses, to, studyID)
}

// LookupByStrategy returns study IDs for a strategy.
func (c *Catalog) LookupByStrategy(strategy string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.strategies[strategy]...)
}

// LookupBySymbol returns study IDs containing a symbol.
func (c *Catalog) LookupBySymbol(symbol string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.symbols[symbol]...)
}

// LookupByTimeframe returns study IDs for a timeframe.
func (c *Catalog) LookupByTimeframe(timeframe string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.timeframes[timeframe]...)
}

// LookupByVersion returns study IDs for a research version.
func (c *Catalog) LookupByVersion(version string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.versions[version]...)
}

// LookupByStatus returns study IDs for a status.
func (c *Catalog) LookupByStatus(status StudyStatus) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.statuses[status]...)
}

// Count returns the number of indexed studies.
func (c *Catalog) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.studies)
}

func appendUniqueString(index *map[string][]string, key string, studyID string) {
	ids := (*index)[key]
	for _, existing := range ids {
		if existing == studyID {
			return
		}
	}
	(*index)[key] = append(ids, studyID)
}

func appendUniqueStatus(index *map[StudyStatus][]string, key StudyStatus, studyID string) {
	ids := (*index)[key]
	for _, existing := range ids {
		if existing == studyID {
			return
		}
	}
	(*index)[key] = append(ids, studyID)
}

func removeFromIndex(ids []string, studyID string) []string {
	out := ids[:0]
	for _, id := range ids {
		if id != studyID {
			out = append(out, id)
		}
	}
	return out
}

// parameterKey builds a stable key for parameter set comparison.
func parameterKey(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(params))
	for k, v := range params {
		pairs = append(pairs, k+"="+v)
	}
	// Sort for stability
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j] < pairs[i] {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	return strings.Join(pairs, ";")
}
