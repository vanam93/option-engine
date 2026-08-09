package researchengine

import (
	"time"

	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// Direction is trade direction.
type Direction string

const (
	DirectionLong  Direction = "LONG"
	DirectionShort Direction = "SHORT"
)

// SimulatedTrade is an institutional-grade completed trade record.
type SimulatedTrade struct {
	TradeID              uuid.UUID          `json:"trade_id"`
	Strategy             string             `json:"strategy"`
	StrategyVersion      string             `json:"strategy_version"`
	ParameterSet         map[string]any     `json:"parameter_set"`
	EntrySignal          strategylib.Signal `json:"entry_signal"`
	ExitSignal           strategylib.Signal `json:"exit_signal,omitempty"`
	Symbol               string             `json:"symbol"`
	Timeframe            string             `json:"timeframe"`
	EntryTime            time.Time          `json:"entry_time"`
	ExitTime             time.Time          `json:"exit_time"`
	EntryPrice           float64            `json:"entry_price"`
	ExitPrice            float64            `json:"exit_price"`
	Quantity             int                `json:"quantity"`
	Direction            Direction          `json:"direction"`
	Commission           float64            `json:"commission"`
	Taxes                float64            `json:"taxes"`
	Slippage             float64            `json:"slippage"`
	GrossProfit          float64            `json:"gross_profit"`
	NetProfit            float64            `json:"net_profit"`
	ReturnPercent        float64            `json:"return_percent"`
	MaxFavorableExcursion float64           `json:"max_favorable_excursion"`
	MaxAdverseExcursion  float64            `json:"max_adverse_excursion"`
	BarsHeld             int                `json:"bars_held"`
	HoldingDuration      time.Duration      `json:"holding_duration"`
	MarketRegime         string             `json:"market_regime,omitempty"`
	ExitReason           string             `json:"exit_reason"`
	RiskReward           float64            `json:"risk_reward"`
	ExpectancyContribution float64          `json:"expectancy_contribution"`
}
