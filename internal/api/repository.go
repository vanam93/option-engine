package api

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/alerts"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
	"github.com/vanam-gangireddy/option-engine/internal/research"
)

const activeSource = "recommendation_state"

// RecommendationReader reads active recommendation lifecycle state.
type RecommendationReader interface {
	List(symbol, strategy, timeframe, status string, confidenceMin float64) []recommendationstate.Recommendation
	Get(id string) (recommendationstate.Recommendation, []recommendationstate.TimelineEntry, bool)
}

// AlertReader reads generated alerts.
type AlertReader interface {
	List(symbol, strategy, timeframe, status string, confidenceMin float64) []alerts.AlertGenerated
}

// OpportunityReader reads opportunity rankings.
type OpportunityReader interface {
	Snapshot() opportunity.OpportunitySnapshot
}

// PerformanceReader reads performance snapshots.
type PerformanceReader interface {
	State() performance.PerformanceSnapshot
}

// ResearchReader reads persisted research artifacts from PostgreSQL.
type ResearchReader interface {
	ListExperiments(ctx context.Context, filter research.QueryFilter) ([]research.ResearchExperiment, error)
	GetResearchBundle(ctx context.Context, experimentID string) (research.ResearchBundle, error)
}

// ComponentHealth exposes component health for intelligence aggregation.
type ComponentHealth interface {
	Health() health.Report
}

// Repository aggregates read-only data sources for the Intelligence API.
type Repository struct {
	cfg             Config
	recommendations RecommendationReader
	alerts          AlertReader
	opportunities   OpportunityReader
	performance     PerformanceReader
	research        ResearchReader
	intelligence    []ComponentHealth
}

// NewRepository creates a read-only API repository.
func NewRepository(
	cfg Config,
	recommendations RecommendationReader,
	alerts AlertReader,
	opportunities OpportunityReader,
	performance PerformanceReader,
	research ResearchReader,
	intelligence ...ComponentHealth,
) *Repository {
	return &Repository{
		cfg:             cfg.withDefaults(),
		recommendations: recommendations,
		alerts:          alerts,
		opportunities:   opportunities,
		performance:     performance,
		research:        research,
		intelligence:    intelligence,
	}
}

func (r *Repository) ListRecommendations(ctx context.Context, filter Filter) ([]RecommendationView, Pagination, error) {
	_ = ctx
	if r.recommendations == nil {
		return nil, Pagination{}, nil
	}
	items := r.recommendations.List(filter.Symbol, filter.Strategy, filter.Timeframe, filter.Status, filter.ConfidenceMin)
	out := make([]RecommendationView, 0, len(items))
	for _, item := range items {
		if !inTimeRange(item.CreatedAt, filter.From, filter.To) {
			continue
		}
		out = append(out, toRecommendationView(item, activeSource))
	}
	sortRecommendations(out, filter.Sort, filter.Order)
	page, pagination := paginate(out, filter, r.cfg.DefaultLimit, r.cfg.MaxLimit)
	return page, pagination, nil
}

func (r *Repository) GetRecommendation(ctx context.Context, id string) (RecommendationView, bool, error) {
	_ = ctx
	if r.recommendations == nil {
		return RecommendationView{}, false, nil
	}
	rec, _, ok := r.recommendations.Get(id)
	if !ok {
		return RecommendationView{}, false, nil
	}
	return toRecommendationView(rec, activeSource), true, nil
}

func (r *Repository) GetTimeline(ctx context.Context, id string) (TimelineView, bool, error) {
	_ = ctx
	if r.recommendations == nil {
		return TimelineView{}, false, nil
	}
	_, timeline, ok := r.recommendations.Get(id)
	if !ok {
		return TimelineView{}, false, nil
	}
	return TimelineView{RecommendationID: id, Timeline: timeline}, true, nil
}

func (r *Repository) ListAlerts(ctx context.Context, filter Filter) ([]AlertView, Pagination, error) {
	_ = ctx
	if r.alerts == nil {
		return nil, Pagination{}, nil
	}
	items := r.alerts.List(filter.Symbol, filter.Strategy, filter.Timeframe, filter.Status, filter.ConfidenceMin)
	filtered := make([]AlertView, 0, len(items))
	for _, item := range items {
		if !inTimeRange(item.GeneratedAt, filter.From, filter.To) {
			continue
		}
		filtered = append(filtered, item)
	}
	sortAlerts(filtered, filter.Sort, filter.Order)
	page, pagination := paginate(filtered, filter, r.cfg.DefaultLimit, r.cfg.MaxLimit)
	return page, pagination, nil
}

func (r *Repository) GetOpportunities(ctx context.Context, filter Filter) (OpportunityView, error) {
	_ = ctx
	if r.opportunities == nil {
		return OpportunityView{}, nil
	}
	snapshot := r.opportunities.Snapshot()
	if filter.Symbol != "" || filter.Timeframe != "" {
		ranked := make([]opportunity.RankedOpportunity, 0, len(snapshot.Ranked))
		for _, item := range snapshot.Ranked {
			if filter.Symbol != "" && item.Symbol != filter.Symbol {
				continue
			}
			if filter.Timeframe != "" && item.Timeframe != filter.Timeframe {
				continue
			}
			ranked = append(ranked, item)
		}
		snapshot.Ranked = ranked
	}
	return OpportunityView{Snapshot: snapshot}, nil
}

func (r *Repository) GetPerformance(ctx context.Context, filter Filter) (PerformanceView, error) {
	_ = ctx
	if r.performance == nil {
		return PerformanceView{}, nil
	}
	return PerformanceView{Snapshot: r.performance.State()}, nil
}

func (r *Repository) ListOptimization(ctx context.Context, filter Filter) ([]OptimizationView, Pagination, error) {
	if r.research == nil {
		return nil, Pagination{}, nil
	}
	start := time.Now()
	experiments, err := r.research.ListExperiments(ctx, research.QueryFilter{
		Strategy:  filter.Strategy,
		Symbol:    filter.Symbol,
		Timeframe: filter.Timeframe,
	})
	recordRepositoryLatency(time.Since(start), err == nil)
	if err != nil {
		return nil, Pagination{}, err
	}

	filtered := make([]research.ResearchExperiment, 0, len(experiments))
	for _, exp := range experiments {
		if !inTimeRange(exp.CreatedAt, filter.From, filter.To) {
			continue
		}
		filtered = append(filtered, exp)
	}
	sortExperiments(filtered, filter.Sort, filter.Order)

	pageExps, pagination := paginate(filtered, filter, r.cfg.DefaultLimit, r.cfg.MaxLimit)
	out := make([]OptimizationView, 0, len(pageExps))
	for _, exp := range pageExps {
		start := time.Now()
		bundle, err := r.research.GetResearchBundle(ctx, exp.ExperimentID)
		recordRepositoryLatency(time.Since(start), err == nil)
		if err != nil {
			continue
		}
		out = append(out, OptimizationView{
			Experiment:   bundle.Experiment,
			Optimization: bundle.Optimization,
		})
	}
	return out, pagination, nil
}

func (r *Repository) GetResearch(ctx context.Context, id string) (ResearchView, error) {
	if r.research == nil {
		return ResearchView{}, ErrNotFound
	}
	start := time.Now()
	bundle, err := r.research.GetResearchBundle(ctx, id)
	recordRepositoryLatency(time.Since(start), err == nil)
	if err != nil {
		if err == research.ErrNotFound {
			return ResearchView{}, ErrNotFound
		}
		return ResearchView{}, err
	}
	return bundle, nil
}

func (r *Repository) IntelligenceHealth() []health.Report {
	reports := make([]health.Report, 0, len(r.intelligence))
	for _, reporter := range r.intelligence {
		if reporter == nil {
			continue
		}
		reports = append(reports, reporter.Health())
	}
	return reports
}

func inTimeRange(at time.Time, from, to time.Time) bool {
	if !from.IsZero() && at.Before(from) {
		return false
	}
	if !to.IsZero() && at.After(to) {
		return false
	}
	return true
}

func sortRecommendations(items []RecommendationView, sortField, order string) {
	desc := strings.EqualFold(order, "desc")
	switch strings.ToLower(sortField) {
	case "confidence":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].Confidence > items[j].Confidence
			}
			return items[i].Confidence < items[j].Confidence
		})
	case "symbol":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].Symbol > items[j].Symbol
			}
			return items[i].Symbol < items[j].Symbol
		})
	case "created_at":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].CreatedAt.After(items[j].CreatedAt)
			}
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		})
	}
}

func sortAlerts(items []AlertView, sortField, order string) {
	desc := strings.EqualFold(order, "desc")
	switch strings.ToLower(sortField) {
	case "symbol":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].Symbol > items[j].Symbol
			}
			return items[i].Symbol < items[j].Symbol
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].GeneratedAt.Before(items[j].GeneratedAt)
			}
			return items[i].GeneratedAt.After(items[j].GeneratedAt)
		})
	}
}

func sortExperiments(items []research.ResearchExperiment, sortField, order string) {
	desc := strings.EqualFold(order, "desc")
	switch strings.ToLower(sortField) {
	case "symbol":
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].Symbol > items[j].Symbol
			}
			return items[i].Symbol < items[j].Symbol
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			if desc {
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			}
			return items[i].CreatedAt.After(items[j].CreatedAt)
		})
	}
}
