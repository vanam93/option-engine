package feedback

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type bucketDef struct {
	label     string
	lower     float64
	upper     float64
	threshold float64
	isBelow   bool
}

// Aggregator maintains dimensional learning statistics.
type Aggregator struct {
	cfg            Config
	overall        runningTotals
	strategies     map[string]runningTotals
	symbols        map[string]runningTotals
	timeframes     map[string]runningTotals
	buckets        []bucketDef
	bucketTotals   []runningTotals
	rollingWindows []int
	rollingHistory []QualityInput
}

// NewAggregator creates an aggregator from engine configuration.
func NewAggregator(cfg Config) *Aggregator {
	buckets := buildBucketDefs(cfg.ConfidenceBuckets)
	bucketTotals := make([]runningTotals, len(buckets))
	return &Aggregator{
		cfg:            cfg,
		strategies:     make(map[string]runningTotals),
		symbols:        make(map[string]runningTotals),
		timeframes:     make(map[string]runningTotals),
		buckets:        buckets,
		bucketTotals:   bucketTotals,
		rollingWindows: append([]int(nil), cfg.RollingWindows...),
	}
}

func buildBucketDefs(thresholds []float64) []bucketDef {
	defs := make([]bucketDef, 0, len(thresholds)+1)
	for i := len(thresholds) - 1; i >= 0; i-- {
		lower := thresholds[i]
		upper := 1.0
		if i < len(thresholds)-1 {
			upper = thresholds[i+1]
		}
		defs = append(defs, bucketDef{
			label:     fmt.Sprintf("%.2f-%.2f", lower, upper),
			lower:     lower,
			upper:     upper,
			threshold: lower,
		})
	}
	defs = append(defs, bucketDef{
		label:     fmt.Sprintf("below_%.2f", thresholds[0]),
		lower:     0,
		upper:     thresholds[0],
		threshold: thresholds[0],
		isBelow:   true,
	})
	return defs
}

// Apply records one completed recommendation and returns the updated snapshot.
func (a *Aggregator) Apply(input QualityInput) FeedbackSnapshot {
	a.overall.add(input)

	strategyKey := normalizeKey(input.Strategy, "unknown")
	s := a.strategies[strategyKey]
	s.add(input)
	a.strategies[strategyKey] = s

	symbolKey := normalizeKey(input.Symbol, "unknown")
	sy := a.symbols[symbolKey]
	sy.add(input)
	a.symbols[symbolKey] = sy

	timeframeKey := normalizeKey(input.Timeframe, "unknown")
	tf := a.timeframes[timeframeKey]
	tf.add(input)
	a.timeframes[timeframeKey] = tf

	bucketIdx := a.bucketIndex(input.Confidence)
	if bucketIdx >= 0 {
		b := a.bucketTotals[bucketIdx]
		b.add(input)
		a.bucketTotals[bucketIdx] = b
	}

	a.rollingHistory = append(a.rollingHistory, input)

	return a.snapshot(input.EvaluatedAt)
}

func (a *Aggregator) bucketIndex(confidence float64) int {
	for i, bucket := range a.buckets {
		if bucket.isBelow {
			if confidence < bucket.upper {
				return i
			}
			continue
		}
		if confidence >= bucket.lower && confidence < bucket.upper {
			return i
		}
		if bucket.upper == 1.0 && confidence >= bucket.lower {
			return i
		}
	}
	return -1
}

func (a *Aggregator) snapshot(at time.Time) FeedbackSnapshot {
	strategies := make([]StrategyStatistics, 0, len(a.strategies))
	for key, totals := range a.strategies {
		strategies = append(strategies, buildStrategyStats(key, totals))
	}
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Strategy < strategies[j].Strategy
	})

	symbols := make([]SymbolStatistics, 0, len(a.symbols))
	for key, totals := range a.symbols {
		symbols = append(symbols, buildSymbolStats(key, totals))
	}
	sort.Slice(symbols, func(i, j int) bool {
		return symbols[i].Symbol < symbols[j].Symbol
	})

	timeframes := make([]TimeframeStatistics, 0, len(a.timeframes))
	for key, totals := range a.timeframes {
		timeframes = append(timeframes, buildTimeframeStats(key, totals))
	}
	sort.Slice(timeframes, func(i, j int) bool {
		return timeframes[i].Timeframe < timeframes[j].Timeframe
	})

	calibration := make([]ConfidenceBucketStatistics, 0, len(a.buckets))
	for i, bucket := range a.buckets {
		calibration = append(calibration, buildBucketStats(bucket.label, bucket.lower, bucket.upper, a.bucketTotals[i]))
	}

	rolling := make([]RollingWindowStatistics, 0, len(a.rollingWindows))
	for _, window := range a.rollingWindows {
		entries := tailEntries(a.rollingHistory, window)
		rolling = append(rolling, buildRollingStats(window, entries))
	}

	version := fmt.Sprintf("%d", a.overall.count)

	return FeedbackSnapshot{
		Overall:               buildOverallStats(a.overall),
		Strategies:            strategies,
		Symbols:               symbols,
		Timeframes:            timeframes,
		ConfidenceCalibration: calibration,
		Rolling:               rolling,
		Timestamp:             at,
		Version:               version,
	}
}

func tailEntries(history []QualityInput, window int) []QualityInput {
	if window <= 0 || len(history) == 0 {
		return nil
	}
	if len(history) <= window {
		return append([]QualityInput(nil), history...)
	}
	return append([]QualityInput(nil), history[len(history)-window:]...)
}

func normalizeKey(value, fallback string) string {
	key := strings.TrimSpace(value)
	if key == "" {
		return fallback
	}
	return key
}

// Stats returns aggregate counts for health reporting.
func (a *Aggregator) Stats() (strategies, symbols, timeframes, recommendations int) {
	return len(a.strategies), len(a.symbols), len(a.timeframes), a.overall.count
}
