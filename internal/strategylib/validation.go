package strategylib

import (
	"fmt"
	"strings"
)

// ValidateParameters checks that required keys exist and int params are positive.
func ValidateParameters(params map[string]any, required []string, intKeys []string) error {
	for _, key := range required {
		if params == nil {
			return fmt.Errorf("missing parameter %q", key)
		}
		if _, ok := params[key]; !ok {
			return fmt.Errorf("missing parameter %q", key)
		}
	}
	for _, key := range intKeys {
		if v := IntParam(params, key, 0); v <= 0 {
			return fmt.Errorf("parameter %q must be > 0", key)
		}
	}
	return nil
}

// ValidateAgainstRanges ensures each parameter value is in its declared range when present.
func ValidateAgainstRanges(params map[string]any, ranges []ParameterRange) error {
	if params == nil {
		return nil
	}
	for _, r := range ranges {
		v, ok := params[r.Name]
		if !ok {
			continue
		}
		if len(r.Values) == 0 {
			continue
		}
		found := false
		for _, allowed := range r.Values {
			if valuesEqual(v, allowed) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parameter %q value %v not in allowed set", r.Name, v)
		}
	}
	return nil
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case int:
		switch bv := b.(type) {
		case int:
			return av == bv
		case int64:
			return int64(av) == bv
		case float64:
			return float64(av) == bv
		}
	case float64:
		switch bv := b.(type) {
		case float64:
			return av == bv
		case int:
			return av == float64(bv)
		case int64:
			return av == float64(bv)
		}
	case string:
		return av == b
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

// OptimizableNames returns parameter names marked for optimization from ranges.
func OptimizableNames(ranges []ParameterRange) []string {
	out := make([]string, 0, len(ranges))
	for _, r := range ranges {
		if len(r.Values) > 1 {
			out = append(out, r.Name)
		}
	}
	return out
}

// MergeTags combines strategy name with optional tags.
func MergeTags(name string, tags ...string) []string {
	out := []string{strings.TrimSpace(name)}
	for _, t := range tags {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
