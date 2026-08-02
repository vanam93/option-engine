package research

import "context"

// Repository persists and queries research artifacts in PostgreSQL.
type Repository interface {
	EnsureSchema(ctx context.Context) error
	EnsureExperiment(ctx context.Context, exp ResearchExperiment) error
	UpsertExperiment(ctx context.Context, exp ResearchExperiment) error
	InsertOptimizationResult(ctx context.Context, result OptimizationResult) error
	InsertWalkForwardResult(ctx context.Context, result WalkForwardResult) error
	InsertMonteCarloResult(ctx context.Context, result MonteCarloResult) error
	GetExperiment(ctx context.Context, experimentID string) (ResearchExperiment, error)
	ListExperiments(ctx context.Context, filter QueryFilter) ([]ResearchExperiment, error)
	GetResearchBundle(ctx context.Context, experimentID string) (ResearchBundle, error)
	InsertResearchReport(ctx context.Context, report ResearchReport) error
	LatestReportVersion(ctx context.Context, experimentID string) (int, error)
	CountEntries(ctx context.Context) (int64, error)
}
