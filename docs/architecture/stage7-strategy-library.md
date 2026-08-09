# Stage 7 — Strategy Library (Phase 1)

Research trading strategies live in `internal/strategylib`. They are independent plugins used by the research engine (simulator, backtest, optimizer).

## Naming

| Package | Role |
|---------|------|
| `internal/analytics/strategy` | Runtime meta-strategy engine (frozen) |
| `internal/analytics/signal` | Indicator signal rules |
| `internal/strategylib` | Research trading strategies |

## Signal contract

`Evaluate(ctx Context) Signal` returns a rich signal object:

- `Decision` — BUY, SELL, EXIT, IGNORE
- `Confidence`, `Strength`, `Score`
- `Reasons`, `Tags`
- `Indicators` — snapshot for reports (no recalculation)
- `Parameters` — parameter set used for this evaluation
- `GeneratedAt`

The trade simulator consumes `Signal` directly; it should not re-derive strategy logic.

## Strategy interface

- `DefaultParameters()`, `ParameterRanges()`, `Validate(params)`, `WarmupBars()`
- `Metadata()` — version, author, reference, optimizable params, supports long/short/exit, intraday/swing/positional

## Registration

```go
import (
    "github.com/vanam-gangireddy/option-engine/internal/strategylib"
    "github.com/vanam-gangireddy/option-engine/internal/strategylib/catalog"
)

catalog.RegisterAll()
desc, _ := strategylib.GetDescriptor("supertrend")
s, _ := strategylib.Get("supertrend")
sig := s.Evaluate(ctx)
```

Registry exposes: `Names()`, `Descriptors()`, `GetDescriptor()`, `ByCategory()`, `ByRegime()`, etc.

## Built-in strategies (13)

`ema_cross`, `ema_pullback`, `macd_cross`, `rsi_reversal`, `bollinger`, `vwap_pullback`, `supertrend`, `opening_range`, `donchian`, `adx_trend`, `trend_following`, `mean_reversion`, `breakout`

## Runtime pipeline

Not wired to the event bus or DI. CSV replay and recommendation confidence remain unchanged until research evidence exists.

