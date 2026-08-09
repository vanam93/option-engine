package validation

import (
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// scriptedStrategy emits predetermined signals at specific bar indices.
type scriptedStrategy struct {
	name     string
	warmup   int
	schedule map[int]strategylib.Signal
}

func newScripted(name string, warmup int, schedule map[int]strategylib.Signal) *scriptedStrategy {
	return &scriptedStrategy{name: name, warmup: warmup, schedule: schedule}
}

func (s *scriptedStrategy) Name() string { return s.name }

func (s *scriptedStrategy) Description() string { return "validation scripted strategy" }

func (s *scriptedStrategy) Metadata() strategylib.Metadata {
	return strategylib.BaseMetadata(s.name, s.Description(), "", strategylib.CategoryTrend)
}

func (s *scriptedStrategy) DefaultParameters() map[string]any { return map[string]any{} }

func (s *scriptedStrategy) ParameterRanges() []strategylib.ParameterRange { return nil }

func (s *scriptedStrategy) Validate(map[string]any) error { return nil }

func (s *scriptedStrategy) WarmupBars() int { return s.warmup }

func (s *scriptedStrategy) Parameters() map[string]any { return map[string]any{} }

func (s *scriptedStrategy) Evaluate(ctx strategylib.Context) strategylib.Signal {
	if sig, ok := s.schedule[ctx.BarIndex]; ok {
		return sig
	}
	return strategylib.Signal{Decision: strategylib.DecisionIgnore}
}

func buySignal() strategylib.Signal {
	return strategylib.Signal{Decision: strategylib.DecisionBuy, Reasons: []string{"test buy"}}
}

func sellSignal() strategylib.Signal {
	return strategylib.Signal{Decision: strategylib.DecisionSell, Reasons: []string{"test sell"}}
}

func exitSignal() strategylib.Signal {
	return strategylib.Signal{Decision: strategylib.DecisionExit, Reasons: []string{"test exit"}}
}
