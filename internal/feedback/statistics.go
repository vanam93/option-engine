package feedback

// runningTotals accumulates sum fields for average computation.
type runningTotals struct {
	count              int
	wins               int
	losses             int
	successful         int
	failed             int
	expired            int
	neutral            int
	falsePositives     int
	falseNegatives     int
	totalReturn        float64
	totalQuality       float64
	totalConfidence    float64
	totalHoldingMs     int64
	totalMFE           float64
	totalMAE           float64
	totalDrawdown      float64
	confidenceAccuracy float64
}

func (r *runningTotals) add(input QualityInput) {
	r.count++
	switch input.Outcome {
	case OutcomeSuccess:
		r.successful++
		r.wins++
	case OutcomeFailed:
		r.failed++
		r.losses++
	case OutcomeExpired:
		r.expired++
	case OutcomeNeutral:
		r.neutral++
	}

	r.totalReturn += input.ReturnPct
	r.totalQuality += input.QualityScore
	r.totalConfidence += input.Confidence
	r.totalHoldingMs += input.HoldingDurationMs
	r.totalMFE += input.MFE
	r.totalMAE += input.MAE
	drawdown := input.MaxDrawdown
	if drawdown == 0 {
		drawdown = input.MAE
	}
	r.totalDrawdown += drawdown
	r.confidenceAccuracy += instanceConfidenceAccuracy(input)

	if isFalsePositive(input) {
		r.falsePositives++
	}
	if isFalseNegative(input) {
		r.falseNegatives++
	}
}

func (r runningTotals) averageReturn() float64 {
	if r.count == 0 {
		return 0
	}
	return r.totalReturn / float64(r.count)
}

func (r runningTotals) averageQuality() float64 {
	if r.count == 0 {
		return 0
	}
	return r.totalQuality / float64(r.count)
}

func (r runningTotals) averageConfidence() float64 {
	if r.count == 0 {
		return 0
	}
	return r.totalConfidence / float64(r.count)
}

func (r runningTotals) averageHoldingMs() int64 {
	if r.count == 0 {
		return 0
	}
	return r.totalHoldingMs / int64(r.count)
}

func (r runningTotals) averageMFE() float64 {
	if r.count == 0 {
		return 0
	}
	return r.totalMFE / float64(r.count)
}

func (r runningTotals) averageMAE() float64 {
	if r.count == 0 {
		return 0
	}
	return r.totalMAE / float64(r.count)
}

func (r runningTotals) averageDrawdown() float64 {
	if r.count == 0 {
		return 0
	}
	return r.totalDrawdown / float64(r.count)
}

func (r runningTotals) averageConfidenceAccuracy() float64 {
	if r.count == 0 {
		return 0
	}
	return r.confidenceAccuracy / float64(r.count)
}

func successRate(successful, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(successful) / float64(total)
}

func winRate(wins, losses int) float64 {
	decided := wins + losses
	if decided == 0 {
		return 0
	}
	return float64(wins) / float64(decided)
}

func instanceConfidenceAccuracy(input QualityInput) float64 {
	switch input.Outcome {
	case OutcomeSuccess:
		return input.Confidence
	case OutcomeFailed:
		return 1 - input.Confidence
	default:
		return 0.5
	}
}

func isFalsePositive(input QualityInput) bool {
	if input.Outcome != OutcomeFailed {
		return false
	}
	return input.RecommendationLevel == LevelStrongBuy || input.RecommendationLevel == LevelBuy
}

func isFalseNegative(input QualityInput) bool {
	if input.Outcome != OutcomeSuccess {
		return false
	}
	return input.RecommendationLevel == LevelWatch || input.RecommendationLevel == LevelAvoid
}

func buildOverallStats(t runningTotals) OverallStatistics {
	return OverallStatistics{
		TotalRecommendations:     t.count,
		Successful:               t.successful,
		Failed:                   t.failed,
		Expired:                  t.expired,
		Neutral:                  t.neutral,
		SuccessRate:              successRate(t.successful, t.count),
		WinRate:                  winRate(t.wins, t.losses),
		AverageReturn:            t.averageReturn(),
		AverageQuality:           t.averageQuality(),
		AverageConfidence:        t.averageConfidence(),
		AverageHoldingDurationMs: t.averageHoldingMs(),
		AverageMFE:               t.averageMFE(),
		AverageMAE:               t.averageMAE(),
		AverageDrawdown:          t.averageDrawdown(),
		FalsePositives:           t.falsePositives,
		FalseNegatives:           t.falseNegatives,
		ConfidenceAccuracy:       t.averageConfidenceAccuracy(),
	}
}

func buildStrategyStats(strategy string, t runningTotals) StrategyStatistics {
	return StrategyStatistics{
		Strategy:                 strategy,
		Recommendations:          t.count,
		Wins:                     t.wins,
		Losses:                   t.losses,
		Expired:                  t.expired,
		AverageReturn:            t.averageReturn(),
		AverageQuality:           t.averageQuality(),
		AverageConfidence:        t.averageConfidence(),
		AverageHoldingDurationMs: t.averageHoldingMs(),
		WinRate:                  winRate(t.wins, t.losses),
		SuccessRate:              successRate(t.successful, t.count),
	}
}

func buildSymbolStats(symbol string, t runningTotals) SymbolStatistics {
	return SymbolStatistics{
		Symbol:                   symbol,
		Recommendations:          t.count,
		Wins:                     t.wins,
		Losses:                   t.losses,
		AverageReturn:            t.averageReturn(),
		AverageQuality:           t.averageQuality(),
		AverageConfidence:        t.averageConfidence(),
		AverageHoldingDurationMs: t.averageHoldingMs(),
	}
}

func buildTimeframeStats(timeframe string, t runningTotals) TimeframeStatistics {
	return TimeframeStatistics{
		Timeframe:                timeframe,
		Recommendations:          t.count,
		AverageReturn:            t.averageReturn(),
		AverageQuality:           t.averageQuality(),
		AverageHoldingDurationMs: t.averageHoldingMs(),
		WinRate:                  winRate(t.wins, t.losses),
	}
}

func buildBucketStats(label string, lower, upper float64, t runningTotals) ConfidenceBucketStatistics {
	successes := t.successful
	failures := t.failed
	return ConfidenceBucketStatistics{
		Label:                    label,
		LowerBound:               lower,
		UpperBound:               upper,
		Recommendations:          t.count,
		Successes:                successes,
		Failures:                 failures,
		SuccessRate:              successRate(successes, successes+failures),
		AverageReturn:            t.averageReturn(),
		AverageQuality:           t.averageQuality(),
		AverageHoldingDurationMs: t.averageHoldingMs(),
	}
}

func buildRollingStats(window int, entries []QualityInput) RollingWindowStatistics {
	t := runningTotals{}
	for _, input := range entries {
		t.add(input)
	}
	return RollingWindowStatistics{
		WindowSize:               window,
		SuccessRate:              successRate(t.successful, t.count),
		AverageReturn:            t.averageReturn(),
		AverageQuality:           t.averageQuality(),
		AverageConfidence:        t.averageConfidence(),
		AverageHoldingDurationMs: t.averageHoldingMs(),
	}
}
