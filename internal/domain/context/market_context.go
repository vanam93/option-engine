package context

import (
	"time"

	"github.com/google/uuid"
)

// VolatilityRegime classifies current market volatility.
type VolatilityRegime string

const (
	RegimeLow    VolatilityRegime = "LOW"
	RegimeNormal VolatilityRegime = "NORMAL"
	RegimeHigh   VolatilityRegime = "HIGH"
	RegimeExtreme VolatilityRegime = "EXTREME"
)

// MarketContext captures the broader market environment.
type MarketContext struct {
	ID              uuid.UUID        `json:"id"`
	Advances        int              `json:"advances"`
	Declines        int              `json:"declines"`
	Unchanged       int              `json:"unchanged"`
	IndiaVIX        float64          `json:"india_vix"`
	NiftyChange     float64          `json:"nifty_change_pct"`
	BankNiftyChange float64          `json:"banknifty_change_pct"`
	FuturesPremium  float64          `json:"futures_premium_pct"`
	GapPercent      float64          `json:"gap_percent"`
	TrendStrength   float64          `json:"trend_strength"` // 0.0 – 1.0
	VolatilityRegime VolatilityRegime `json:"volatility_regime"`
	SectorStrength  map[string]float64 `json:"sector_strength"`
	Correlations    map[string]float64 `json:"correlations"`
	EvaluatedAt     time.Time        `json:"evaluated_at"`
}
