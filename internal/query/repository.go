package query

import (
	"context"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/alerts"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
	"github.com/vanam-gangireddy/option-engine/internal/research"
	"github.com/vanam-gangireddy/option-engine/internal/scanner"
)

// RecommendationReader reads recommendation lifecycle state.
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

// ScannerReader reads scanner state.
type ScannerReader interface {
	Snapshot() scanner.ScannerSnapshot
}

// PerformanceReader reads performance snapshots.
type PerformanceReader interface {
	State() performance.PerformanceSnapshot
}

// OptimizationReader reads optimization snapshots.
type OptimizationReader interface {
	State() optimization.StateSnapshot
}

// ResearchReader reads persisted research artifacts.
type ResearchReader interface {
	GetResearchBundle(ctx context.Context, experimentID string) (research.ResearchBundle, error)
}

// ComponentHealth exposes component health for intelligence aggregation.
type ComponentHealth interface {
	Health() health.Report
}

// Repository aggregates read-only data sources for the query API.
type Repository struct {
	recommendations RecommendationReader
	alerts          AlertReader
	opportunities   OpportunityReader
	scanner         ScannerReader
	performance     PerformanceReader
	optimization    OptimizationReader
	research        ResearchReader
	intelligence    []ComponentHealth
}

// NewRepository creates a read-only query repository.
func NewRepository(
	recommendations RecommendationReader,
	alerts AlertReader,
	opportunities OpportunityReader,
	scanner ScannerReader,
	performance PerformanceReader,
	optimization OptimizationReader,
	research ResearchReader,
	intelligence ...ComponentHealth,
) *Repository {
	return &Repository{
		recommendations: recommendations,
		alerts:          alerts,
		opportunities:   opportunities,
		scanner:         scanner,
		performance:     performance,
		optimization:    optimization,
		research:        research,
		intelligence:    intelligence,
	}
}

func (r *Repository) ListRecommendations(filter Filter) []RecommendationView {
	if r.recommendations == nil {
		return nil
	}
	items := r.recommendations.List(filter.Symbol, filter.Strategy, filter.Timeframe, filter.Status, filter.ConfidenceMin)
	out := make([]RecommendationView, 0, len(items))
	for _, item := range items {
		out = append(out, toRecommendationView(item))
	}
	return out
}

func (r *Repository) GetRecommendation(id string) (RecommendationView, bool) {
	if r.recommendations == nil {
		return RecommendationView{}, false
	}
	rec, _, ok := r.recommendations.Get(id)
	if !ok {
		return RecommendationView{}, false
	}
	return toRecommendationView(rec), true
}

func (r *Repository) GetTimeline(id string) ([]recommendationstate.TimelineEntry, bool) {
	if r.recommendations == nil {
		return nil, false
	}
	_, timeline, ok := r.recommendations.Get(id)
	if !ok {
		return nil, false
	}
	return timeline, true
}

func (r *Repository) ListAlerts(filter Filter) []AlertView {
	if r.alerts == nil {
		return nil
	}
	return r.alerts.List(filter.Symbol, filter.Strategy, filter.Timeframe, filter.Status, filter.ConfidenceMin)
}

func (r *Repository) Opportunities() opportunity.OpportunitySnapshot {
	if r.opportunities == nil {
		return opportunity.OpportunitySnapshot{}
	}
	return r.opportunities.Snapshot()
}

func (r *Repository) Scanner(filter Filter) scanner.ScannerSnapshot {
	if r.scanner == nil {
		return scanner.ScannerSnapshot{}
	}
	snapshot := r.scanner.Snapshot()
	if filter.Symbol == "" && filter.Timeframe == "" && filter.Strategy == "" {
		return snapshot
	}

	filteredResults := make([]scanner.ScanResult, 0, len(snapshot.Results))
	for _, item := range snapshot.Results {
		if filter.Symbol != "" && item.Symbol != filter.Symbol {
			continue
		}
		if filter.Timeframe != "" && item.Timeframe != filter.Timeframe {
			continue
		}
		filteredResults = append(filteredResults, item)
	}

	filteredStates := make([]scanner.SymbolState, 0, len(snapshot.States))
	for _, item := range snapshot.States {
		if filter.Symbol != "" && item.Symbol != filter.Symbol {
			continue
		}
		if filter.Timeframe != "" && item.Timeframe != filter.Timeframe {
			continue
		}
		if filter.Strategy != "" {
			if item.HasSignal && item.LastSignal.Strategy != filter.Strategy {
				continue
			}
			if item.HasDecision && item.LastDecision.Strategy != filter.Strategy {
				continue
			}
			if item.HasPerf && item.Performance.Strategy != filter.Strategy {
				continue
			}
		}
		filteredStates = append(filteredStates, item)
	}

	return scanner.ScannerSnapshot{Results: filteredResults, States: filteredStates}
}

func (r *Repository) Performance() performance.PerformanceSnapshot {
	if r.performance == nil {
		return performance.PerformanceSnapshot{}
	}
	return r.performance.State()
}

func (r *Repository) Optimization(filter Filter) optimization.StateSnapshot {
	if r.optimization == nil {
		return optimization.StateSnapshot{}
	}
	snapshot := r.optimization.State()
	if filter.Symbol == "" && filter.Strategy == "" && filter.Timeframe == "" {
		return snapshot
	}

	filterEval := func(rec optimization.EvaluationRecord) bool {
		if filter.Strategy != "" && rec.Key.Strategy != filter.Strategy {
			return false
		}
		if filter.Symbol != "" && rec.Key.Symbol != filter.Symbol {
			return false
		}
		if filter.Timeframe != "" && rec.Key.Timeframe != filter.Timeframe {
			return false
		}
		return true
	}

	evals := make([]optimization.EvaluationRecord, 0, len(snapshot.Evaluations))
	for _, rec := range snapshot.Evaluations {
		if filterEval(rec) {
			evals = append(evals, rec)
		}
	}
	ranks := make([]optimization.EvaluationRecord, 0, len(snapshot.Rankings))
	for _, rec := range snapshot.Rankings {
		if filterEval(rec) {
			ranks = append(ranks, rec)
		}
	}
	return optimization.StateSnapshot{Evaluations: evals, Rankings: ranks}
}

func (r *Repository) Research(ctx context.Context, id string) (research.ResearchBundle, error) {
	if r.research == nil {
		return research.ResearchBundle{}, ErrNotFound
	}
	start := time.Now()
	bundle, err := r.research.GetResearchBundle(ctx, id)
	recordRepositoryLatency(time.Since(start), err == nil)
	return bundle, err
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
