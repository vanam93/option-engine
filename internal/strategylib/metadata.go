package strategylib

// Category classifies a research strategy style.
type Category string

const (
	CategoryTrend         Category = "trend"
	CategoryMeanReversion Category = "mean_reversion"
	CategoryBreakout      Category = "breakout"
)

// Regime identifies a market regime a strategy is suited for.
type Regime string

const (
	RegimeTrending      Regime = "trending"
	RegimeSideways      Regime = "sideways"
	RegimeVolatile      Regime = "volatile"
	RegimeLowVolatility Regime = "low_volatility"
	RegimeGapDay        Regime = "gap_day"
	RegimeExpiryWeek    Regime = "expiry_week"
	RegimeHighMomentum  Regime = "high_momentum"
	RegimeLowMomentum   Regime = "low_momentum"
)

// RiskLevel describes expected strategy risk profile.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// TradeFrequency describes how often a strategy typically trades.
type TradeFrequency string

const (
	TradeFrequencyLow    TradeFrequency = "low"
	TradeFrequencyMedium TradeFrequency = "medium"
	TradeFrequencyHigh   TradeFrequency = "high"
)

// HoldingPeriod describes typical position duration.
type HoldingPeriod string

const (
	HoldingShort  HoldingPeriod = "short"
	HoldingMedium HoldingPeriod = "medium"
	HoldingLong   HoldingPeriod = "long"
)

// ParameterRange documents accepted optimization values for a parameter.
type ParameterRange struct {
	Name   string `json:"name"`
	Values []any  `json:"values"`
}

// DefaultTimeframes returns the standard timeframe set strategies may subset.
func DefaultTimeframes() []string {
	return []string{"1m", "3m", "5m", "15m", "30m", "1h", "1d"}
}

// Metadata describes a strategy for registry search, optimization, and AI explanation.
type Metadata struct {
	Name                  string           `json:"name"`
	Description           string           `json:"description"`
	Version               string           `json:"version"`
	Author                string           `json:"author"`
	Reference             string           `json:"reference"`
	Category              Category         `json:"category"`
	DefaultParameters     map[string]any   `json:"default_parameters"`
	OptimizableParameters []string         `json:"optimizable_parameters"`
	ParameterRanges       []ParameterRange `json:"parameter_ranges"`
	SupportedTimeframes   []string         `json:"supported_timeframes"`
	PreferredRegimes      []Regime         `json:"preferred_regimes"`
	TradeFrequency        TradeFrequency   `json:"trade_frequency"`
	HoldingPeriod         HoldingPeriod    `json:"holding_period"`
	RiskLevel             RiskLevel        `json:"risk_level"`
	SymbolTypes           []string         `json:"symbol_types"`
	MinimumHistory        int              `json:"minimum_history"`
	SupportsLong          bool             `json:"supports_long"`
	SupportsShort         bool             `json:"supports_short"`
	SupportsExit          bool             `json:"supports_exit"`
	SupportsIntraday      bool             `json:"supports_intraday"`
	SupportsSwing         bool             `json:"supports_swing"`
	SupportsPositional    bool             `json:"supports_positional"`
}

// StrategyDescriptor is a registry read model for UI and optimizer discovery.
type StrategyDescriptor struct {
	Name              string           `json:"name"`
	Version           string           `json:"version"`
	Category          Category         `json:"category"`
	Metadata          Metadata         `json:"metadata"`
	WarmupBars        int              `json:"warmup_bars"`
	DefaultParameters map[string]any   `json:"default_parameters"`
	ParameterRanges   []ParameterRange `json:"parameter_ranges"`
}

const (
	defaultStrategyVersion = "1.0.0"
	defaultStrategyAuthor  = "option-engine"
)

// BaseMetadata fills common metadata fields for built-in strategies.
func BaseMetadata(name, description, reference string, category Category) Metadata {
	return Metadata{
		Name:                  name,
		Description:           description,
		Version:               defaultStrategyVersion,
		Author:                defaultStrategyAuthor,
		Reference:             reference,
		Category:              category,
		SupportedTimeframes:   DefaultTimeframes(),
		SymbolTypes:           []string{"index", "large_cap"},
		SupportsLong:          true,
		SupportsShort:         true,
		SupportsExit:          true,
		SupportsIntraday:      true,
		SupportsSwing:         true,
		SupportsPositional:    false,
	}
}
