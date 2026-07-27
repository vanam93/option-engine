# Roadmap

Each stage produces a **fully working, testable system** before the next begins.

## Stage 0 — Requirements & Domain Design ✅

- Domain models: Tick, Candle, OptionContract, OptionChainSnapshot, Signal, Recommendation, Trade, MarketContext, IndicatorValue
- Event envelope and event types
- Module interface contracts (`application/ports`)
- Performance targets and coding standards

See [docs/domain/README.md](domain/README.md).

## Stage 1 — Foundation & Architecture ✅

- Go module with Clean Architecture layout
- Configuration (Viper + YAML + env overrides)
- Structured logging (slog)
- Dependency injection container
- HTTP server (Gin) with health/ready/status endpoints
- WebSocket hub with heartbeat
- PostgreSQL connection pool (pgx)
- Docker + Docker Compose
- Unit tests + CI/CD (GitHub Actions)

## Stage 2 — Market Data Engine

- Provider abstraction (Zerodha, Upstox, Mock)
- WebSocket management, auto-reconnect, heartbeat
- Tick normalization, duplicate filtering
- Unified internal market event model

## Stage 3 — Storage & Replay Engine

- Tick/candle/option chain persistence
- Replay mode for backtesting and debugging

## Stage 4 — Technical Indicator Engine

- Incremental indicator updates (EMA, RSI, MACD, etc.)

## Stage 5 — Option Intelligence Engine

- OI analysis, PCR, Max Pain, Greeks, writer traps

## Stage 6 — Market Context Engine

- Breadth, VIX, sector strength, correlations

## Stage 7 — Strategy Engine

- Independent, configurable strategies

## Stage 8 — Decision Engine

- Signal aggregation, weighted scoring, no-trade detection

## Stage 9 — Trade Management Engine

- Position tracking, dynamic SL, trailing stops

## Stage 10 — Backtesting Engine

- Historical replay, performance metrics

## Stage 11 — Dashboard & Alerts

- React dashboard with live charts and alerts

## Stage 12 — AI Explanation Engine

- Human-readable explanations (never generates signals)
