# Stage 8 — Research Engine (Phase 2)

The research engine is separate from the runtime event pipeline. It replays historical candles, runs `strategylib` strategies, simulates fills, journals trades, and computes statistics.

## Flow (v2 — shared indicator cache)

```text
CSV candles → Dataset (immutable candles + IndicatorStore)
    → Engine.RunAll (one store per dataset)
    → Simulator (BarIndex + shared store, no history copy)
    → strategylib via indaccess (read cached indicators)
    → Signal → fills → Journal → Statistics → Qualification → Leaderboard
```

Indicators (EMA, SMA, RSI, MACD, ATR, Bollinger, ADX, SuperTrend, Donchian, Opening Range, Session VWAP) are computed once per parameter set and cached in `indicatorstore.Store` (lazy, thread-safe).

## Qualification

Each `PerformanceReport` includes automatic classification:

| Status | Rules |
|--------|-------|
| `rejected` | PF < 0.8 or DD > 30% |
| `poor` | PF 0.8–1.2 |
| `average` | PF > 1.2 |
| `good` | PF > 1.5 and DD < 15% |
| `excellent` | PF > 2.0, win rate > 60%, DD < 10% |

Single-strategy CLI prints qualification after the report. Championship leaderboard includes a `Status` column.

## CLI

Single strategy:

```bash
go run ./cmd/backtest --strategy ema_cross --csv data/raw/nifty50/5min.csv --symbol NIFTY50 --timeframe 5m
```

Strategy championship (all registered strategies, one candle load):

```bash
go run ./cmd/backtest --all --csv data/raw/nifty50/5min.csv --symbol NIFTY50 --timeframe 5m
```

Exports (with `--all`):

- `output/leaderboard.json`
- `output/leaderboard.csv`

Override with `--export-json` and `--export-csv`.

## Validation

Before `--all` championship runs, the validation suite executes automatically:

```bash
go test ./internal/researchengine/validation/... -count=1
```

Checks cover simulator PnL, position rules, statistics invariants, and deterministic strategy fixtures. Championship also prints per-strategy diagnostics (signal counts, ignored duplicates, trade frequency warnings).

## Championship

`ChampionshipEngine` loads candles once, runs `catalog.AllFresh()` strategies on the identical stream, scores each with configurable `RankingWeights` (default: 30% profit factor, 20% Sharpe, 20% drawdown penalty, 15% win rate, 10% expectancy, 5% trade count), and builds a `StrategyLeaderboard`.

## Packages

| Package | Role |
|---------|------|
| `internal/strategylib` | Strategy plugins returning rich `Signal` |
| `internal/researchengine` | Simulator, journal, statistics, engine, qualification |
| `internal/researchengine/indicatorstore` | Lazy shared indicator cache |
| `internal/strategylib/indaccess` | Store-aware indicator reads for strategies |
| `cmd/backtest` | Standalone research CLI |

Runtime pipeline (Gateway, EventBus, Recommendation) is not modified.
