package query

import (
	"context"
	"errors"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

// API provides read-only query operations for REST handlers.
type API struct {
	cfg  Config
	repo *Repository
}

// NewAPI creates a query API facade.
func NewAPI(cfg Config, repo *Repository) (*API, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, errors.New("query: nil repository")
	}
	return &API{cfg: cfg, repo: repo}, nil
}

func (a *API) enabled() bool {
	return a.cfg.Enabled
}

func (a *API) normalizeFilter(filter Filter) Filter {
	out := filter
	if out.Limit <= 0 {
		out.Limit = a.cfg.DefaultLimit
	}
	if out.Limit > a.cfg.MaxLimit {
		out.Limit = a.cfg.MaxLimit
	}
	if out.Offset < 0 {
		out.Offset = 0
	}
	return out
}

func (a *API) metadata(filter Filter) Metadata {
	return Metadata{
		Timestamp: time.Now().UTC(),
		Filters:   filter,
	}
}

// ListRecommendations returns paginated recommendations.
func (a *API) ListRecommendations(filter Filter) (ListResponse[RecommendationView], error) {
	if !a.enabled() {
		return ListResponse[RecommendationView]{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	filter = a.normalizeFilter(filter)
	items := a.repo.ListRecommendations(filter)
	page, pagination := paginate(items, filter.Limit, filter.Offset)
	return ListResponse[RecommendationView]{
		Metadata: Metadata{Timestamp: time.Now().UTC(), Filters: filter, Pagination: pagination},
		Data:     page,
	}, nil
}

// GetRecommendation returns a single recommendation.
func (a *API) GetRecommendation(id string, filter Filter) (ItemResponse[RecommendationView], error) {
	if !a.enabled() {
		return ItemResponse[RecommendationView]{}, ErrDisabled
	}
	start := time.Now()
	item, ok := a.repo.GetRecommendation(id)
	var err error
	if !ok {
		err = ErrNotFound
	}
	recordRequest(time.Since(start), err)
	if !ok {
		return ItemResponse[RecommendationView]{}, ErrNotFound
	}
	return ItemResponse[RecommendationView]{
		Metadata: a.metadata(filter),
		Data:     item,
	}, nil
}

// GetTimeline returns a recommendation timeline.
func (a *API) GetTimeline(id string, filter Filter) (TimelineResponse, error) {
	if !a.enabled() {
		return TimelineResponse{}, ErrDisabled
	}
	start := time.Now()
	timeline, ok := a.repo.GetTimeline(id)
	var err error
	if !ok {
		err = ErrNotFound
	}
	recordRequest(time.Since(start), err)
	if !ok {
		return TimelineResponse{}, ErrNotFound
	}
	return TimelineResponse{
		Metadata: a.metadata(filter),
		ID:       id,
		Timeline: timeline,
	}, nil
}

// ListAlerts returns paginated alerts.
func (a *API) ListAlerts(filter Filter) (ListResponse[AlertView], error) {
	if !a.enabled() {
		return ListResponse[AlertView]{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	filter = a.normalizeFilter(filter)
	items := a.repo.ListAlerts(filter)
	page, pagination := paginate(items, filter.Limit, filter.Offset)
	return ListResponse[AlertView]{
		Metadata: Metadata{Timestamp: time.Now().UTC(), Filters: filter, Pagination: pagination},
		Data:     page,
	}, nil
}

// GetOpportunities returns the current opportunity snapshot.
func (a *API) GetOpportunities(filter Filter) (OpportunityResponse, error) {
	if !a.enabled() {
		return OpportunityResponse{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	snapshot := a.repo.Opportunities()
	if filter.Symbol != "" || filter.Timeframe != "" {
		filtered := make([]opportunity.RankedOpportunity, 0, len(snapshot.Ranked))
		for _, item := range snapshot.Ranked {
			if filter.Symbol != "" && item.Symbol != filter.Symbol {
				continue
			}
			if filter.Timeframe != "" && item.Timeframe != filter.Timeframe {
				continue
			}
			filtered = append(filtered, item)
		}
		snapshot.Ranked = filtered
	}
	return OpportunityResponse{Metadata: a.metadata(filter), Snapshot: snapshot}, nil
}

// GetScanner returns the current scanner snapshot.
func (a *API) GetScanner(filter Filter) (ScannerResponse, error) {
	if !a.enabled() {
		return ScannerResponse{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	return ScannerResponse{
		Metadata: a.metadata(filter),
		Snapshot: a.repo.Scanner(filter),
	}, nil
}

// GetPerformance returns the current performance snapshot.
func (a *API) GetPerformance(filter Filter) (PerformanceResponse, error) {
	if !a.enabled() {
		return PerformanceResponse{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	return PerformanceResponse{
		Metadata: a.metadata(filter),
		Snapshot: a.repo.Performance(),
	}, nil
}

// GetOptimization returns the current optimization snapshot.
func (a *API) GetOptimization(filter Filter) (OptimizationResponse, error) {
	if !a.enabled() {
		return OptimizationResponse{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	return OptimizationResponse{
		Metadata: a.metadata(filter),
		Snapshot: a.repo.Optimization(filter),
	}, nil
}

// GetResearch returns a research bundle by experiment ID.
func (a *API) GetResearch(ctx context.Context, id string, filter Filter) (ResearchResponse, error) {
	if !a.enabled() {
		return ResearchResponse{}, ErrDisabled
	}
	start := time.Now()
	bundle, err := a.repo.Research(ctx, id)
	recordRequest(time.Since(start), err)
	if err != nil {
		if errors.Is(err, research.ErrNotFound) {
			return ResearchResponse{}, ErrNotFound
		}
		return ResearchResponse{}, err
	}
	return ResearchResponse{Metadata: a.metadata(filter), Bundle: bundle}, nil
}

// GetIntelligenceHealth returns aggregated intelligence component health.
func (a *API) GetIntelligenceHealth(filter Filter) (IntelligenceHealthResponse, error) {
	if !a.enabled() {
		return IntelligenceHealthResponse{}, ErrDisabled
	}
	start := time.Now()
	defer func() { recordRequest(time.Since(start), nil) }()

	return IntelligenceHealthResponse{
		Metadata:   a.metadata(filter),
		Components: a.repo.IntelligenceHealth(),
	}, nil
}
