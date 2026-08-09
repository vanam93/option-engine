package strategylib

// Strategy is a research trading strategy plugin.
// Each instance owns its parameters and incremental indicator state.
type Strategy interface {
	Name() string
	Description() string
	Metadata() Metadata
	Parameters() map[string]any
	DefaultParameters() map[string]any
	ParameterRanges() []ParameterRange
	Validate(params map[string]any) error
	WarmupBars() int
	Evaluate(ctx Context) Signal
}
