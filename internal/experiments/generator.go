package experiments

import (
	"encoding/json"

	"github.com/google/uuid"
)

// GenerateExperimentID creates a new experiment batch identifier.
func GenerateExperimentID() string {
	return uuid.NewString()
}

// GenerateRunID creates a new run identifier.
func GenerateRunID() string {
	return uuid.NewString()
}

// GenerateMatrix builds the cartesian product of configured parameter ranges.
func GenerateMatrix(cfg Config, experimentID string) []ExperimentRun {
	cfg = cfg.WithDefaults()
	ranges := cfg.ParameterRanges

	dimensions := []struct {
		name   string
		values []any
	}{
		{"ema_fast", intSliceToAny(ranges.EMAFast)},
		{"ema_slow", intSliceToAny(ranges.EMASlow)},
		{"rsi_period", intSliceToAny(ranges.RSIPeriod)},
		{"rsi_overbought", floatSliceToAny(ranges.RSIOverbought)},
		{"rsi_oversold", floatSliceToAny(ranges.RSIOversold)},
		{"macd_fast", intSliceToAny(ranges.MACDFast)},
		{"macd_slow", intSliceToAny(ranges.MACDSlow)},
		{"macd_signal", intSliceToAny(ranges.MACDSignal)},
		{"min_confidence", floatSliceToAny(ranges.MinConfidence)},
		{"max_positions", intSliceToAny(ranges.MaxPositions)},
	}

	active := make([]struct {
		name   string
		values []any
	}, 0, len(dimensions))
	for _, dim := range dimensions {
		if len(dim.values) > 0 {
			active = append(active, dim)
		}
	}

	var combos []ParameterSet
	combos = append(combos, ParameterSet{})

	for _, dim := range active {
		next := make([]ParameterSet, 0, len(combos)*len(dim.values))
		for _, base := range combos {
			for _, value := range dim.values {
				clone := cloneParams(base)
				clone[dim.name] = value
				next = append(next, clone)
			}
		}
		combos = next
	}

	if len(combos) == 0 {
		combos = []ParameterSet{{}}
	}

	runs := make([]ExperimentRun, 0, len(combos)*len(cfg.Symbols)*len(cfg.Timeframes))
	for _, symbol := range cfg.Symbols {
		for _, timeframe := range cfg.Timeframes {
			for _, params := range combos {
				runID := GenerateRunID()
				enriched := cloneParams(params)
				enriched["run_id"] = runID
				enriched["experiment_id"] = experimentID
				enriched["symbol"] = symbol
				enriched["timeframe"] = timeframe

				runs = append(runs, ExperimentRun{
					ExperimentID: experimentID,
					RunID:        runID,
					BacktestID:   experimentID,
					Strategy:     cfg.Strategy,
					Symbol:       symbol,
					Timeframe:    timeframe,
					Parameters:   enriched,
					Status:       RunStatusCreated,
				})
			}
		}
	}
	return runs
}

// SerializeParameters encodes a parameter set for event correlation.
func SerializeParameters(params ParameterSet) string {
	data, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	return string(data)
}

// RunIDFromParameters extracts run_id from serialized parameters.
func RunIDFromParameters(serialized string) string {
	if serialized == "" {
		return ""
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(serialized), &raw); err != nil {
		return ""
	}
	if v, ok := raw["run_id"].(string); ok {
		return v
	}
	return ""
}

func intSliceToAny(values []int) []any {
	if len(values) == 0 {
		return nil
	}
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func floatSliceToAny(values []float64) []any {
	if len(values) == 0 {
		return nil
	}
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func cloneParams(src ParameterSet) ParameterSet {
	if len(src) == 0 {
		return ParameterSet{}
	}
	out := make(ParameterSet, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
