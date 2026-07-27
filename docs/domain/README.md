# Stage 0 — Requirements & Domain Design

This document defines the shared vocabulary, events, module boundaries, and performance targets for the NSE trading intelligence platform.

## Vision

Bloomberg Terminal + TradingView + Option Analytics + Trading Assistant — each stage produces a fully working, testable system before the next begins.

## Domain Models

| Model | Package | Purpose |
|-------|---------|---------|
| `Tick` | `domain/market` | Single price update (spot, future, option) |
| `Candle` | `domain/market` | OHLCV bar for any timeframe |
| `OptionContract` | `domain/option` | Single strike/expiry contract |
| `OptionChainSnapshot` | `domain/option` | Full chain at a point in time |
| `Signal` | `domain/signal` | Output from any analysis module |
| `Recommendation` | `domain/decision` | Aggregated trade suggestion |
| `Trade` | `domain/trade` | Active or closed position |
| `MarketContext` | `domain/context` | Breadth, VIX, sector strength |
| `IndicatorValue` | `domain/indicator` | Typed indicator output |

## Event Flow

```
MarketDataReceived → TickPersisted → IndicatorUpdated
                                   → OptionChainAnalyzed
                                   → ContextEvaluated
                                   → StrategySignalGenerated
                                   → DecisionMade
                                   → TradeUpdated
                                   → AlertFired
```

All events carry: `EventID`, `EventType`, `Timestamp`, `Source`, `Payload`.

## Module Interfaces

| Interface | Consumer | Producer |
|-----------|----------|----------|
| `MarketDataProvider` | Market Data Engine | Broker adapters |
| `MarketEventBus` | All engines | Market Data Engine |
| `MarketDataStore` | Storage, Replay, Backtest | Storage Engine |
| `IndicatorEngine` | Strategy Engine | TA Engine |
| `OptionAnalyzer` | Strategy, Decision | Option Intelligence |
| `ContextEvaluator` | Strategy, Decision | Market Context |
| `Strategy` | Decision Engine | Strategy Engine |
| `DecisionMaker` | Trade Manager, Dashboard | Decision Engine |
| `TradeManager` | Dashboard, Alerts | Trade Manager |
| `ReplayEngine` | Backtest, Debug | Storage Engine |
| `ExplanationProvider` | Dashboard | AI Engine (Stage 12) |

## Supported Data Providers (planned)

| Provider | Priority | Capabilities |
|----------|----------|--------------|
| Zerodha Kite | P0 | WebSocket ticks, option chain, historical |
| Upstox | P1 | WebSocket ticks, option chain |
| Dhan | P1 | WebSocket ticks |
| Mock/Replay | P0 | Deterministic testing |

## Performance Targets

| Metric | Target |
|--------|--------|
| Tick ingestion → normalized event | < 5 ms |
| End-to-end analysis after tick | < 100 ms |
| Option chain snapshot analysis | < 50 ms |
| Decision engine latency | < 20 ms |
| WebSocket fan-out to dashboard | < 10 ms |
| Replay tick throughput | > 10,000 ticks/sec |

## Coding Standards

- Clean Architecture: domain has zero external dependencies.
- All public interfaces live in `internal/application/ports`.
- Infrastructure implements ports; adapters expose HTTP/WS.
- Table-driven unit tests for all domain logic.
- Integration tests use Docker Compose PostgreSQL.
- Structured logging via `slog`; no printf debugging.
- Context propagation on every I/O boundary.
- Graceful shutdown with configurable drain timeout.

## Testing Requirements

| Layer | Coverage Target | Tooling |
|-------|-----------------|---------|
| Domain | 90%+ | testify |
| Application | 80%+ | testify + mocks |
| Infrastructure | 70%+ | testcontainers |
| Adapters | 70%+ | httptest |
