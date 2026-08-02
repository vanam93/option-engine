# Stage 3 — Market Analytics Layer

Stage 3 consumes **only canonical events** from the Stage 2 `EventBus`. No analytics component may import providers, gateway code, or broker adapters.

## Pipeline

```text
EventBus (Stage 2, frozen)
    ↓  MarketDataReceived
Candle Engine          ← Phase 1 (implemented)
    ↓  CandleClosed
Indicator Engine       ← Phase 2A (implemented)
    ↓  IndicatorUpdated
Signal Engine          ← Phase 3 (planned)
    ↓  StrategySignalGenerated
```

## Phase 1 — Candle Engine

### Responsibility

- Subscribe to `MarketDataReceived` events on the runtime bus.
- Aggregate canonical `market.Tick` payloads into OHLCV candles per symbol and timeframe.
- Publish `CandleClosed` events when a bar rolls into the next bucket.

### Packages

| Package | Role |
|---------|------|
| `internal/analytics/ports` | Minimal bus contract for analytics engines |
| `internal/analytics/candle` | Aggregation engine, bucket logic, health |

### Configuration

```yaml
analytics:
  candle:
    enabled: true
    timeframes:
      - 1m
      - 5m
    timezone: Asia/Kolkata
    subscriber_buffer: 256
    flush_on_shutdown: true
    volume_mode: cumulative
    order_policy: reject_older
    idle_evict_after: 0s
```

### Lifecycle

1. Candle engine subscribes to the bus **before** the gateway starts (no startup race).
2. Gateway publishes canonical ticks to the bus.
3. On shutdown, the candle engine closes **before** the gateway.
4. When `flush_on_shutdown` is true, in-progress bars are published as final `CandleClosed` events.

### Volume contract

See `internal/analytics/candle/VOLUME_CONTRACT.md`. Default mode is `cumulative` (NSE session volume).

### Thread safety

- A single consumer goroutine processes bus events sequentially.
- The aggregator mutex protects in-progress bar state.
- `Close()` cancels context, waits on `WaitGroup`, optionally flushes in-progress bars, then closes the subscription.

### Health

`GET /health/components` includes `candle_engine` with processed/published counts and subscription drop metrics.

## Ownership rules

- Stage 2 components remain frozen (see `docs/ARCHITECTURE_RULES.md`).
- Analytics engines may **publish** new event types but must not mutate Stage 2 cache or subscription state.
- All time bucketing uses the injected `clock.Clock` and configured timezone.

## Phase 2A — Indicator Engine (foundation)

- Subscribes only to `CandleClosed` events.
- Incremental EMA and SMA per `(symbol, timeframe)`.
- Publishes `IndicatorUpdated` with `domain/indicator.IndicatorValue` payload.
- Warm-up: no publish until lookback period is satisfied.
- Startup order: Indicator → Candle → Gateway. Shutdown: Gateway → Candle → Indicator.

## Phase 2B — RSI and ATR

- Incremental Wilder's RSI (close-based) and ATR (OHLC-based).
- Configurable periods via `analytics.indicator.rsi` and `analytics.indicator.atr`.
- Reuses the existing indicator cache and `IndicatorUpdated` contract.
- Default periods: RSI(14), ATR(14).

## Next phases

- **Phase 2C**: Additional indicators (MACD, Bollinger Bands, etc.)
- **Phase 3**: Signal engine consuming `IndicatorUpdated`.
