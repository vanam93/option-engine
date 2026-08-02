package laboratory

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// ComparisonCriteria selects completed studies for read-only comparison.
type ComparisonCriteria struct {
	Strategy        string            `json:"strategy,omitempty"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	Symbol          string            `json:"symbol,omitempty"`
	Timeframe       string            `json:"timeframe,omitempty"`
	StartTime       time.Time         `json:"start_time,omitempty"`
	EndTime         time.Time         `json:"end_time,omitempty"`
	ResearchVersion string            `json:"research_version,omitempty"`
}

// Comparison is a read-only aggregate of matching completed studies.
type Comparison struct {
	ComparisonID string             `json:"comparison_id"`
	Criteria     ComparisonCriteria `json:"criteria"`
	StudyIDs     []string           `json:"study_ids"`
	Studies      []Study            `json:"studies"`
	CreatedAt    time.Time          `json:"created_at"`
}

// CompareStudies finds completed studies matching criteria without recalculation.
func (e *Engine) CompareStudies(criteria ComparisonCriteria) (Comparison, error) {
	if !e.cfg.Enabled {
		return Comparison{}, ErrEngineClosed
	}

	matches := e.findMatchingStudies(criteria)
	if len(matches) == 0 {
		return Comparison{}, ErrComparisonNotFound
	}

	studyIDs := make([]string, 0, len(matches))
	for _, study := range matches {
		studyIDs = append(studyIDs, study.StudyID)
	}

	comparison := Comparison{
		ComparisonID: generateComparisonID(e.clk.Now().UTC()),
		Criteria:     criteria,
		StudyIDs:     studyIDs,
		Studies:      matches,
		CreatedAt:    e.clk.Now().UTC(),
	}

	e.repo.SaveComparison(comparison)
	e.health.recordComparison()
	e.publishCompared(comparison)

	return comparison, nil
}

// GetComparison returns a stored comparison by ID.
func (e *Engine) GetComparison(id string) (Comparison, bool) {
	return e.repo.GetComparison(id)
}

func (e *Engine) findMatchingStudies(criteria ComparisonCriteria) []Study {
	var candidates []Study

	switch {
	case criteria.Strategy != "":
		candidates = e.repo.ListByStrategy(criteria.Strategy)
	default:
		candidates = e.repo.ListStudies()
	}

	paramKey := parameterKey(criteria.Parameters)
	matches := make([]Study, 0)
	for _, study := range candidates {
		if study.Status != StudyStatusCompleted {
			continue
		}
		if criteria.ResearchVersion != "" && study.ResearchVersion != criteria.ResearchVersion {
			continue
		}
		if len(criteria.Parameters) > 0 && parameterKey(study.Parameters) != paramKey {
			continue
		}
		if criteria.Symbol != "" && !containsString(study.Symbols, criteria.Symbol) {
			continue
		}
		if criteria.Timeframe != "" && !containsTimeframe(study.Timeframes, criteria.Timeframe) {
			continue
		}
		if !overlapsDateRange(study, criteria.StartTime, criteria.EndTime) {
			continue
		}
		matches = append(matches, study)
	}
	return matches
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsTimeframe(items []market.Timeframe, target string) bool {
	for _, item := range items {
		if string(item) == target {
			return true
		}
	}
	return false
}

func generateComparisonID(at time.Time) string {
	return fmt.Sprintf("CMP-%s-%s", at.UTC().Format("20060102T150405"), uuid.NewString()[:8])
}
