package stratutil

import "github.com/vanam-gangireddy/option-engine/internal/strategylib"

// MergeParams overlays params onto defaults.
func MergeParams(defaults map[string]any, params map[string]any) map[string]any {
	out := strategylib.CloneParams(defaults)
	for k, v := range params {
		out[k] = v
	}
	return out
}
