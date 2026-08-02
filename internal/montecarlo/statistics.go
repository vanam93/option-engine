package montecarlo

import (
	"math"
	"sort"
)

// SimulationOutcome holds metrics from a single Monte Carlo path.
type SimulationOutcome struct {
	TotalReturn float64
	MaxDrawdown float64
}

// ComputeConfidenceInterval derives percentile bounds from sorted samples.
func ComputeConfidenceInterval(samples []float64, level float64) ConfidenceInterval {
	if len(samples) == 0 {
		return ConfidenceInterval{Level: level}
	}

	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)

	mean := mean(sorted)
	median := percentile(sorted, 0.5)
	tail := (1 - level) / 2
	lower := percentile(sorted, tail)
	upper := percentile(sorted, 1-tail)

	return ConfidenceInterval{
		Level:  level,
		Lower:  lower,
		Upper:  upper,
		Mean:   mean,
		Median: median,
	}
}

// ComputeDistributionSummary aggregates simulation outcomes.
func ComputeDistributionSummary(returns []float64, drawdowns []float64) DistributionSummary {
	sortedReturns := append([]float64(nil), returns...)
	sort.Float64s(sortedReturns)
	sortedDrawdowns := append([]float64(nil), drawdowns...)
	sort.Float64s(sortedDrawdowns)

	summary := DistributionSummary{
		MeanReturn:        mean(returns),
		MedianReturn:      percentile(sortedReturns, 0.5),
		StdDevReturn:      stdDev(returns),
		MeanMaxDrawdown:   mean(drawdowns),
		MedianMaxDrawdown: percentile(sortedDrawdowns, 0.5),
	}
	if len(sortedDrawdowns) > 0 {
		summary.WorstDrawdown = sortedDrawdowns[len(sortedDrawdowns)-1]
		summary.BestDrawdown = sortedDrawdowns[0]
	}
	return summary
}

// TotalReturn sums trade PnL values.
func TotalReturn(trades []float64) float64 {
	var total float64
	for _, t := range trades {
		total += t
	}
	return total
}

// MaxDrawdown computes peak-to-trough decline on a cumulative equity curve.
func MaxDrawdown(trades []float64) float64 {
	if len(trades) == 0 {
		return 0
	}
	peak := 0.0
	equity := 0.0
	maxDD := 0.0
	for _, trade := range trades {
		equity += trade
		if equity > peak {
			peak = equity
		}
		dd := peak - equity
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

// ProbabilityOfProfit returns the fraction of samples with positive return.
func ProbabilityOfProfit(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	wins := 0
	for _, r := range returns {
		if r > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(returns))
}

// ProbabilityOfLoss returns the fraction of samples with negative return.
func ProbabilityOfLoss(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	losses := 0
	for _, r := range returns {
		if r < 0 {
			losses++
		}
	}
	return float64(losses) / float64(len(returns))
}

// RiskOfRuin returns the fraction of paths exceeding a drawdown threshold.
func RiskOfRuin(drawdowns []float64, startingCapital float64, ruinDrawdownPct float64) float64 {
	if len(drawdowns) == 0 {
		return 0
	}
	threshold := startingCapital * ruinDrawdownPct
	if threshold <= 0 {
		threshold = math.Max(1, mean(drawdowns))
	}
	ruined := 0
	for _, dd := range drawdowns {
		if dd >= threshold {
			ruined++
		}
	}
	return float64(ruined) / float64(len(drawdowns))
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := mean(values)
	var sumSq float64
	for _, v := range values {
		d := v - m
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(values)))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
