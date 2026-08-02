package optimization

const (
	profitFactorCap = 5.0
	expectancyCap   = 100.0
	drawdownCap     = 1000.0
)

// ComputeMetrics derives evaluation metrics from incremental trade state.
func ComputeMetrics(
	totalTrades int,
	winRate, realizedPnL, unrealizedPnL float64,
	grossProfit, grossLoss float64,
	avgWin, avgLoss float64,
	maxDrawdown float64,
) EvaluationMetrics {
	netPnL := realizedPnL + unrealizedPnL
	pf := profitFactor(grossProfit, grossLoss)
	avgTrade := averageTrade(realizedPnL, totalTrades)
	exp := expectancy(winRate, avgWin, avgLoss)
	rr := riskReward(avgWin, avgLoss)

	return EvaluationMetrics{
		TotalTrades:  totalTrades,
		NetPnL:       netPnL,
		RealizedPnL:  realizedPnL,
		WinRate:      winRate,
		ProfitFactor: pf,
		AverageTrade: avgTrade,
		Expectancy:   exp,
		MaxDrawdown:  maxDrawdown,
		RiskReward:   rr,
		SharpeRatio:  sharpePlaceholder(),
	}
}

// Score computes a weighted optimization score from metrics and scoring config.
func Score(metrics EvaluationMetrics, weights ScoringConfig) float64 {
	pfNorm := normalize(metrics.ProfitFactor, profitFactorCap)
	expNorm := normalize(metrics.Expectancy, expectancyCap)
	ddNorm := normalize(metrics.MaxDrawdown, drawdownCap)

	return weights.ProfitFactorWeight*pfNorm +
		weights.WinRateWeight*metrics.WinRate +
		weights.ExpectancyWeight*expNorm -
		weights.DrawdownPenalty*ddNorm
}

func profitFactor(grossProfit, grossLoss float64) float64 {
	if grossLoss <= 0 {
		if grossProfit > 0 {
			return profitFactorCap
		}
		return 0
	}
	pf := grossProfit / grossLoss
	if pf > profitFactorCap {
		return profitFactorCap
	}
	return pf
}

func averageTrade(realizedPnL float64, totalTrades int) float64 {
	if totalTrades == 0 {
		return 0
	}
	return realizedPnL / float64(totalTrades)
}

func expectancy(winRate, avgWin, avgLoss float64) float64 {
	lossRate := 1.0 - winRate
	return winRate*avgWin - lossRate*avgLoss
}

func riskReward(avgWin, avgLoss float64) float64 {
	if avgLoss <= 0 {
		if avgWin > 0 {
			return avgWin
		}
		return 0
	}
	return avgWin / avgLoss
}

func normalize(value, cap float64) float64 {
	if value <= 0 || cap <= 0 {
		return 0
	}
	if value >= cap {
		return 1
	}
	return value / cap
}

func sharpePlaceholder() float64 {
	return 0
}
