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
Signal Engine          ← Phase 3 (implemented)
    ↓  SignalGenerated
Strategy Engine        ← Phase 4 (implemented)
    ↓  StrategyDecision
Decision/Risk Engine   ← Phase 5 (implemented)
    ↓  ApprovedTradeIntent
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

## Phase 2C — MACD and Bollinger Bands

- Incremental MACD: fast EMA, slow EMA, signal EMA, and histogram.
- Incremental Bollinger Bands: middle (SMA), upper/lower bands, bandwidth, and %B.
- Configurable via `analytics.indicator.macd` and `analytics.indicator.bollinger`.
- Default MACD: fast 12, slow 26, signal 9.
- Default Bollinger: period 20, stddev 2.0.
- Warm-up: MACD publishes after slow EMA and signal EMA are satisfied; Bollinger after the SMA window fills.
- Health includes `macd_instances`, `bollinger_instances`, and `warmed_instances`.

### Configuration

```yaml
analytics:
  indicator:
    macd:
      fast_period: 12
      slow_period: 26
      signal_period: 9
    bollinger:
      period: 20
      stddev: 2.0
```

## Phase 3 — Signal Engine

### Responsibility

- Subscribe to `IndicatorUpdated` events on the runtime bus.
- Evaluate configurable technical rules and publish `SignalGenerated` events.
- Does **not** execute trades; downstream Strategy Engine consumes signals.

### Packages

| Package | Role |
|---------|------|
| `internal/analytics/signal` | Rule evaluation, crossover state, health |

### Signal types

`BUY`, `SELL`, `EXIT_LONG`, `EXIT_SHORT`, `NEUTRAL`

### Initial rules

| Rule | Trigger |
|------|---------|
| EMA crossover | Fast EMA crosses above/below slow EMA |
| MACD crossover | MACD line crosses signal line |
| RSI | RSI below oversold → BUY; above overbought → SELL |
| Bollinger | Close below lower band → BUY; above upper band → SELL |

Rules are individually enabled/disabled. Crossover rules maintain minimal prior-value state per `(symbol, timeframe)`; threshold rules are stateless per event.

### Configuration

```yaml
analytics:
  signal:
    enabled: true
    ema_cross:
      enabled: true
      fast_period: 9
      slow_period: 21
    macd_cross:
      enabled: true
    rsi:
      enabled: true
      oversold: 30
      overbought: 70
    bollinger:
      enabled: true
```

### Lifecycle

1. Signal engine subscribes to the bus **before** the indicator engine starts.
2. Indicator engine publishes `IndicatorUpdated` events.
3. On shutdown, gateway → candle → indicator → signal (reverse startup order).

### Event payload (`SignalGenerated`)

- `symbol`, `timeframe`, `signal`, `strategy`, `confidence`, `timestamp`
- `indicators`: map of indicator values used in the evaluation

### Health

`GET /health/components` includes `signal_engine` with `signals_generated`, `buy_count`, `sell_count`, `neutral_count`, and `active_rules`.

## Phase 4 — Strategy Engine

### Responsibility

- Subscribe to `SignalGenerated` events on the runtime bus.
- Combine aligned signals into strategy decisions and publish `StrategyDecision` events.
- Does **not** execute trades; downstream Decision/Risk Engine consumes decisions.

### Packages

| Package | Role |
|---------|------|
| `internal/analytics/strategy` | Strategy rules, position state, health |

### Strategies

| Strategy | Inputs | Outputs |
|----------|--------|---------|
| Trend following | EMA crossover + MACD confirmation | `LONG_ENTRY`, `SHORT_ENTRY` |
| Mean reversion | RSI + Bollinger confirmation | `LONG_ENTRY`, `SHORT_ENTRY` |
| Breakout | Placeholder (disabled by default) | — |

Decision states: `LONG_ENTRY`, `SHORT_ENTRY`, `LONG_EXIT`, `SHORT_EXIT`, `HOLD`.

### Configuration

```yaml
analytics:
  strategy:
    enabled: true
    trend_following:
      enabled: true
    mean_reversion:
      enabled: true
    breakout:
      enabled: false
```

### Lifecycle

1. Strategy engine subscribes to the bus **before** the signal engine starts.
2. Signal engine publishes `SignalGenerated` events.
3. On shutdown, gateway → candle → indicator → strategy → signal (reverse startup order).

### Event payload (`StrategyDecision`)

- `symbol`, `timeframe`, `decision`, `strategy`, `confidence`, `timestamp`, `reason`

### Health

`GET /health/components` includes `strategy_engine` with `strategies_enabled`, `decisions_generated`, `long_entries`, `short_entries`, `exits`, and `holds`.

## Phase 5 — Decision & Risk Engine

### Responsibility

- Subscribe to `StrategyDecision` events on the runtime bus.
- Apply risk controls and publish `ApprovedTradeIntent` events.
- Does **not** execute trades; downstream Execution Engine consumes approved intents.

### Packages

| Package | Role |
|---------|------|
| `internal/analytics/risk` | Risk rules, position limits, health |

### Risk controls

| Control | Behavior |
|---------|----------|
| Confidence filter | Reject when confidence is below `min_confidence` |
| Position limit | Reject new flat entries when `max_positions` is reached |
| Duplicate position | Reject `LONG_ENTRY` when already long; reject `SHORT_ENTRY` when already short |
| Daily trade limit | Reject when `max_trades_per_day` approved trades are exceeded |

### Configuration

```yaml
analytics:
  risk:
    enabled: true
    min_confidence: 0.70
    max_positions: 5
    max_trades_per_day: 20
```

### Lifecycle

1. Risk engine subscribes to the bus **before** the strategy engine starts.
2. Strategy engine publishes `StrategyDecision` events.
3. On shutdown, gateway → candle → indicator → signal → strategy → risk (reverse startup order).

### Event payload (`ApprovedTradeIntent`)

- `symbol`, `timeframe`, `status`, `action`, `quantity`, `strategy`, `confidence`, `reason`, `timestamp`
- `status`: `APPROVED` or `REJECTED`

### Health

`GET /health/components` includes `risk_engine` with `decisions_received`, `approved_trades`, `rejected_trades`, `active_positions`, and `rejection_reasons`.

## Next phases

- **Execution Engine**: consume `ApprovedTradeIntent` and open/update/close trades.
