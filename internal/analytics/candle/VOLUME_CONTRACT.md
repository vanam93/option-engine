# Candle volume contract

`Tick.Volume` on canonical market ticks is interpreted according to `analytics.candle.volume_mode`.

## cumulative (default)

Matches NSE broker feeds where `Volume` is the running session total.

| Condition | Volume delta applied |
|-----------|---------------------|
| First tick with volume V | V |
| Volume increased | `current - previous` |
| Volume unchanged | 0 (OHLC still updates) |
| Volume decreased (session reset) | `current` |

`TradeCount` increments on every tick regardless of volume delta.

## incremental

Each tick's `Volume` field is the trade size for that tick only.

| Condition | Volume delta applied |
|-----------|---------------------|
| Volume > 0 | `Volume` |
| Volume == 0 | 0 |
