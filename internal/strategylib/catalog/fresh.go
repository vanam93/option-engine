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

// AllFresh returns new strategy instances for championship runs.
// Registry prototypes must not be reused across simulations because strategies hold incremental state.
func AllFresh() []strategylib.Strategy {
	return []strategylib.Strategy{
		ema_cross.New(nil),
		ema_pullback.New(nil),
		macd_cross.New(nil),
		rsi_reversal.New(nil),
		bollinger.New(nil),
		vwap_pullback.New(nil),
		supertrend.New(nil),
		opening_range.New(nil),
		donchian.New(nil),
		adx_trend.New(nil),
		trend_following.New(nil),
		mean_reversion.New(nil),
		breakout.New(nil),
	}
}
