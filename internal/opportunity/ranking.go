package opportunity

import (
	"sort"
	"time"
)

// RankedOpportunity is a scored and ranked symbol opportunity.
type RankedOpportunity struct {
	ScoreResult
	Rank      int
	Timestamp time.Time
}

// Ranker orders opportunities and selects top candidates.
type Ranker struct {
	cfg Config
}

// NewRanker creates a ranker from configuration.
func NewRanker(cfg Config) *Ranker {
	return &Ranker{cfg: cfg.withDefaults()}
}

// Rank scores all symbols and returns opportunities sorted by confidence descending.
func (r *Ranker) Rank(states []SymbolState, platform PlatformState, scorer *Scorer, at time.Time) []RankedOpportunity {
	scored := make([]RankedOpportunity, 0, len(states))
	for _, state := range states {
		result := scorer.Compute(state, platform)
		scored = append(scored, RankedOpportunity{
			ScoreResult: result,
			Timestamp:   at,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Confidence == scored[j].Confidence {
			return scored[i].Symbol < scored[j].Symbol
		}
		return scored[i].Confidence > scored[j].Confidence
	})

	for i := range scored {
		scored[i].Rank = i + 1
	}
	return scored
}

// TopN returns the highest-ranked opportunities up to configured N.
func (r *Ranker) TopN(ranked []RankedOpportunity) []RankedOpportunity {
	if len(ranked) <= r.cfg.TopN {
		return ranked
	}
	return ranked[:r.cfg.TopN]
}

// Summary aggregates classification counts across all ranked opportunities.
type Summary struct {
	OpportunitiesRanked int
	TopCandidates       int
	AverageConfidence   float64
	BuyCount            int
	WatchCount          int
	IgnoreCount         int
}

// Summarize computes health and ranking summary statistics.
func Summarize(ranked []RankedOpportunity, topN []RankedOpportunity) Summary {
	summary := Summary{
		OpportunitiesRanked: len(ranked),
		TopCandidates:       len(topN),
	}
	if len(ranked) == 0 {
		return summary
	}

	var total float64
	for _, item := range ranked {
		total += item.Confidence
		switch item.Classification {
		case ClassificationBuy:
			summary.BuyCount++
		case ClassificationWatch:
			summary.WatchCount++
		case ClassificationIgnore:
			summary.IgnoreCount++
		}
	}
	summary.AverageConfidence = total / float64(len(ranked))
	return summary
}
