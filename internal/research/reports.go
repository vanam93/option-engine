package research

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ReportGenerator builds unified reports from PostgreSQL data.
type ReportGenerator struct {
	repo Repository
}

// NewReportGenerator creates a report builder backed by the repository.
func NewReportGenerator(repo Repository) *ReportGenerator {
	return &ReportGenerator{repo: repo}
}

// Generate loads research data from PostgreSQL and builds a unified report.
func (g *ReportGenerator) Generate(ctx context.Context, experimentID string, at time.Time) (UnifiedReport, int, error) {
	start := time.Now()
	bundle, err := g.repo.GetResearchBundle(ctx, experimentID)
	_ = start // read latency tracked by engine health

	if err != nil {
		return UnifiedReport{}, 0, err
	}

	version, err := g.repo.LatestReportVersion(ctx, experimentID)
	if err != nil {
		return UnifiedReport{}, 0, err
	}
	version++

	researchID := fmt.Sprintf("research-%s", uuid.NewString())
	report := buildUnifiedReport(researchID, version, bundle, at)
	return report, version, nil
}
