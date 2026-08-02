package walkforward

import (
	"math"
	"sort"

	"github.com/vanam-gangireddy/option-engine/internal/experiments"
)

// SelectBest chooses the highest optimization score from training results.
func SelectBest(results []experiments.RunResult) (experiments.RunResult, bool) {
	if len(results) == 0 {
		return experiments.RunResult{}, false
	}
	best := results[0]
	for _, result := range results[1:] {
		if result.OptimizationScore > best.OptimizationScore {
			best = result
			continue
		}
		if result.OptimizationScore == best.OptimizationScore && result.RunID < best.RunID {
			best = result
		}
	}
	return best, true
}

// AggregateValidation computes cross-window summary metrics.
func AggregateValidation(completed []WindowResult) AggregatedValidation {
	if len(completed) == 0 {
		return AggregatedValidation{ParameterDrift: map[string]float64{}}
	}

	validationScores := make([]float64, len(completed))
	trainingScores := make([]float64, len(completed))
	for i, result := range completed {
		validationScores[i] = result.ValidationScore
		trainingScores[i] = result.TrainingScore
	}

	meanVal := mean(validationScores)
	meanTrain := mean(trainingScores)
	stdVal := stdDev(validationScores, meanVal)

	stability := 0.0
	if meanVal > 0 {
		stability = 1.0 - (stdVal / meanVal)
		if stability < 0 {
			stability = 0
		}
	}

	return AggregatedValidation{
		WindowCount:           len(completed),
		MeanValidationScore:   meanVal,
		ScoreStdDev:           stdVal,
		MeanTrainingScore:     meanTrain,
		TrainingValidationGap: meanTrain - meanVal,
		StabilityScore:        stability,
		ParameterDrift:        computeParameterDrift(completed),
	}
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(values)))
}

func computeParameterDrift(completed []WindowResult) map[string]float64 {
	if len(completed) < 2 {
		return map[string]float64{}
	}

	type accum struct {
		values []float64
	}
	series := make(map[string]*accum)

	for _, result := range completed {
		for key, raw := range result.BestParameters {
			if key == "run_id" || key == "experiment_id" || key == "walkforward_id" ||
				key == "window_index" || key == "phase" || key == "symbol" || key == "timeframe" ||
				key == "train_start" || key == "train_end" || key == "validation_start" || key == "validation_end" {
				continue
			}
			val, ok := toFloat64(raw)
			if !ok {
				continue
			}
			entry, exists := series[key]
			if !exists {
				entry = &accum{}
				series[key] = entry
			}
			entry.values = append(entry.values, val)
		}
	}

	drift := make(map[string]float64, len(series))
	for key, entry := range series {
		if len(entry.values) < 2 {
			continue
		}
		sort.Float64s(entry.values)
		m := mean(entry.values)
		if m == 0 {
			continue
		}
		drift[key] = stdDev(entry.values, m) / math.Abs(m)
	}
	return drift
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
