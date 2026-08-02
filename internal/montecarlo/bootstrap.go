package montecarlo

import (
	"math"
	"math/rand"

	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// ExtractTradeReturns synthesizes per-trade PnL values from aggregated metrics.
func ExtractTradeReturns(m optimization.EvaluationMetrics) []float64 {
	if m.TotalTrades <= 0 {
		if m.NetPnL != 0 {
			return []float64{m.NetPnL}
		}
		if m.AverageTrade != 0 {
			return []float64{m.AverageTrade}
		}
		return nil
	}

	n := m.TotalTrades
	wins := int(math.Round(m.WinRate * float64(n)))
	if wins < 0 {
		wins = 0
	}
	if wins > n {
		wins = n
	}
	losses := n - wins

	if wins == 0 {
		perLoss := m.NetPnL / float64(losses)
		trades := make([]float64, losses)
		for i := range trades {
			trades[i] = perLoss
		}
		return trades
	}
	if losses == 0 {
		perWin := m.NetPnL / float64(wins)
		trades := make([]float64, wins)
		for i := range trades {
			trades[i] = perWin
		}
		return trades
	}

	avgWin := m.AverageTrade
	if avgWin <= 0 {
		avgWin = math.Abs(m.NetPnL) / float64(n) * 2
	}
	avgLoss := (m.NetPnL - float64(wins)*avgWin) / float64(losses)

	trades := make([]float64, 0, n)
	for i := 0; i < wins; i++ {
		trades = append(trades, avgWin)
	}
	for i := 0; i < losses; i++ {
		trades = append(trades, avgLoss)
	}
	return trades
}

// BootstrapSample resamples trade returns with replacement.
func BootstrapSample(returns []float64, count int, rng *rand.Rand) []float64 {
	if len(returns) == 0 || count <= 0 {
		return nil
	}
	out := make([]float64, count)
	for i := 0; i < count; i++ {
		out[i] = returns[rng.Intn(len(returns))]
	}
	return out
}

// ShuffleOrder randomizes trade ordering without changing composition.
func ShuffleOrder(returns []float64, rng *rand.Rand) []float64 {
	out := append([]float64(nil), returns...)
	rng.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}
