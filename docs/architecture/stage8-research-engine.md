# Stage 8 — Research Engine (Phase 2)

The research engine is separate from the runtime event pipeline. It replays historical candles, runs `strategylib` strategies, simulates fills, journals trades, and computes statistics.

## Flow

```text
CSV candles → researchengine.Engine → strategylib.Strategy.Evaluate → Signal
    → Simulator (fills, costs, MFE/MAE) → Journal → Statistics → Report
```

## CLI

```bash
go run ./cmd/backtest --strategy ema_cross --csv data/raw/nifty50/5min.csv --symbol NIFTY50 --timeframe 5m
```

## Packages

| Package | Role |
|---------|------|
| `internal/strategylib` | Strategy plugins returning rich `Signal` |
| `internal/researchengine` | Simulator, journal, statistics, engine |
| `cmd/backtest` | Standalone research CLI |

Runtime pipeline (Gateway, EventBus, Recommendation) is not modified.
