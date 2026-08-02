package quality

import "time"

// computePriceStatistics derives price metrics from tracker state.
func computePriceStatistics(tracker activeTracker, exitPrice float64, at time.Time, completed bool) PriceStatistics {
	stats := PriceStatistics{
		EntryPrice:      tracker.entryPrice,
		LatestPrice:     tracker.currentPrice,
		HighestPrice:    tracker.highestPrice,
		LowestPrice:     tracker.lowestPrice,
		HoldingDuration: tracker.holdingDuration(at),
	}

	if completed && exitPrice > 0 {
		stats.ExitPrice = exitPrice
		stats.LatestPrice = exitPrice
	}

	if stats.EntryPrice > 0 && stats.LatestPrice > 0 {
		stats.AbsoluteReturn = stats.LatestPrice - stats.EntryPrice
		stats.PercentageReturn = stats.AbsoluteReturn / stats.EntryPrice
	}

	return stats
}

// computeQualityMetrics derives MFE, MAE, and drawdown from tracker state.
func computeQualityMetrics(tracker activeTracker, at time.Time) QualityMetrics {
	metrics := QualityMetrics{
		HoldingDuration: tracker.holdingDuration(at).Milliseconds(),
	}

	if tracker.entryPrice <= 0 || !tracker.hasPrice {
		return metrics
	}

	entry := tracker.entryPrice
	metrics.MFE = (tracker.highestPrice - entry) / entry
	metrics.MAE = (entry - tracker.lowestPrice) / entry
	if metrics.MAE < 0 {
		metrics.MAE = 0
	}
	metrics.MaximumReturn = metrics.MFE
	metrics.MaximumDrawdown = metrics.MAE

	if tracker.currentPrice > 0 {
		metrics.ReturnPct = (tracker.currentPrice - entry) / entry
	}

	return metrics
}

// aggregateHistoricalStats summarizes completed reports.
type aggregateHistoricalStats struct {
	Count              int
	Successful         int
	Failed             int
	Expired            int
	TotalReturnPct     float64
	TotalQualityScore  float64
	TotalTrackingMins  float64
}

func (a *aggregateHistoricalStats) add(report QualityReport) {
	a.Count++
	switch report.Outcome {
	case OutcomeSuccess:
		a.Successful++
	case OutcomeFailed:
		a.Failed++
	case OutcomeExpired:
		a.Expired++
	}
	a.TotalReturnPct += report.QualityMetrics.ReturnPct
	a.TotalQualityScore += report.QualityScore
	a.TotalTrackingMins += report.PriceStatistics.HoldingDuration.Minutes()
}

func (a aggregateHistoricalStats) averageReturn() float64 {
	if a.Count == 0 {
		return 0
	}
	return a.TotalReturnPct / float64(a.Count)
}

func (a aggregateHistoricalStats) averageQualityScore() float64 {
	if a.Count == 0 {
		return 0
	}
	return a.TotalQualityScore / float64(a.Count)
}

func (a aggregateHistoricalStats) averageTrackingMinutes() float64 {
	if a.Count == 0 {
		return 0
	}
	return a.TotalTrackingMins / float64(a.Count)
}
