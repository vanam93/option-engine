package quality

import "time"

// EvaluateOutcome determines SUCCESS, FAILED, NEUTRAL, or EXPIRED from metrics and config.
func EvaluateOutcome(cfg Config, returnPct float64, expired bool) Outcome {
	if expired {
		return OutcomeExpired
	}
	if returnPct >= cfg.SuccessReturnPct {
		return OutcomeSuccess
	}
	if returnPct <= cfg.FailureReturnPct {
		return OutcomeFailed
	}
	return OutcomeNeutral
}

// Evaluator builds quality reports from tracker state.
type Evaluator struct {
	cfg Config
}

// NewEvaluator creates an evaluator with the given configuration.
func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{cfg: cfg}
}

// BuildReport constructs a quality report from an active tracker.
func (e *Evaluator) BuildReport(tracker activeTracker, at time.Time, completed bool, expired bool) QualityReport {
	exitPrice := 0.0
	var exitTime *time.Time
	if completed {
		exitPrice = tracker.currentPrice
		t := at
		exitTime = &t
	}

	priceStats := computePriceStatistics(tracker, exitPrice, at, completed)
	metrics := computeQualityMetrics(tracker, at)

	outcome := OutcomeNeutral
	if completed || expired {
		outcome = EvaluateOutcome(e.cfg, metrics.ReturnPct, expired)
	}

	holdingMins := priceStats.HoldingDuration.Minutes()
	score := ComputeQualityScore(ScoreInput{
		ReturnPct:   metrics.ReturnPct,
		MFE:         metrics.MFE,
		MAE:         metrics.MAE,
		HoldingMins: holdingMins,
		Confidence:  tracker.confidence,
		Level:       tracker.level,
		Outcome:     outcome,
	})

	classification := Classify(e.cfg, score, outcome)
	if !completed && !expired {
		classification = ""
		outcome = ""
	}

	return QualityReport{
		RecommendationID:    tracker.recommendationID,
		Symbol:              tracker.symbol,
		Timeframe:           tracker.timeframe,
		Strategy:            tracker.strategy,
		RecommendationLevel: tracker.level,
		Confidence:          tracker.confidence,
		EntryTime:           tracker.entryTime,
		ExitTime:            exitTime,
		CurrentStatus:       tracker.status,
		Outcome:             outcome,
		Classification:      classification,
		QualityScore:        score,
		PriceStatistics:     priceStats,
		QualityMetrics:      metrics,
		TrackingActive:      !completed && !expired,
		Completed:           completed || expired,
		EvaluatedAt:         at,
	}
}
