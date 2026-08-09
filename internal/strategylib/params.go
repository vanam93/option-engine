package strategylib

// CloneParams returns a shallow copy of a parameter map.
func CloneParams(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// IntParam reads an int parameter with fallback.
func IntParam(params map[string]any, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return fallback
	}
}

// FloatParam reads a float64 parameter with fallback.
func FloatParam(params map[string]any, key string, fallback float64) float64 {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return fallback
	}
}
