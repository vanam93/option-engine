package catalog

import (
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/adx_trend"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/bollinger"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/breakout"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/donchian"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_pullback"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/mean_reversion"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/macd_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/opening_range"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/rsi_reversal"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/supertrend"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/trend_following"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/vwap_pullback"
)

// RegisterAll registers every built-in research strategy with the default registry.
func RegisterAll() {
	strategylib.Register(ema_cross.NewDefault())
	strategylib.Register(ema_pullback.NewDefault())
	strategylib.Register(macd_cross.NewDefault())
	strategylib.Register(rsi_reversal.NewDefault())
	strategylib.Register(bollinger.NewDefault())
	strategylib.Register(vwap_pullback.NewDefault())
	strategylib.Register(supertrend.NewDefault())
	strategylib.Register(opening_range.NewDefault())
	strategylib.Register(donchian.NewDefault())
	strategylib.Register(adx_trend.NewDefault())
	strategylib.Register(trend_following.NewDefault())
	strategylib.Register(mean_reversion.NewDefault())
	strategylib.Register(breakout.NewDefault())
}
