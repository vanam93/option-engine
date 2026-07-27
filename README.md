# Option Engine

Production-grade NSE trading intelligence platform — Bloomberg Terminal + TradingView + Option Analytics + Trading Assistant.

## Architecture

```
Market Data → Storage & Replay → [Technical | Option | Context] → Strategy → Decision → Trade Manager → Dashboard → AI
```

Built with **Clean Architecture**:

```
cmd/server/          Entry point
internal/
  domain/            Pure business models (zero dependencies)
  application/ports/ Interface contracts between modules
  infrastructure/    Config, logger, postgres, DI
  adapters/          HTTP, WebSocket handlers
configs/             YAML configuration
deployments/docker/  Docker & Compose
docs/domain/         Stage 0 domain design
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose (for PostgreSQL)

### Run locally

```bash
# Start PostgreSQL
make docker-up

# Run the server
make run
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe (checks PostgreSQL) |
| GET | `/api/v1/status` | Service info |
| GET | `/ws` | WebSocket connection |

### Configuration

Edit `configs/config.yaml` or override via environment variables (prefix `OE_`):

```
OE_HTTP_PORT=8080
OE_POSTGRES_HOST=localhost
OE_LOGGING_LEVEL=debug
```

## Development

```bash
make test      # Run unit tests
make build     # Build binary
make lint      # Run golangci-lint
make tidy      # Update go.sum
```

## Roadmap

| Stage | Status | Description |
|-------|--------|-------------|
| 0 | Done | Domain models, events, interfaces |
| 1 | Done | Foundation & architecture |
| 2 | Next | Market Data Engine |
| 3 | Planned | Storage & Replay Engine |
| 4 | Planned | Technical Indicator Engine |
| 5 | Planned | Option Chain Intelligence |
| 6 | Planned | Market Context Engine |
| 7 | Planned | Strategy Engine |
| 8 | Planned | Decision Engine |
| 9 | Planned | Trade Management Engine |
| 10 | Planned | Backtesting Engine |
| 11 | Planned | Dashboard & Alerts |
| 12 | Planned | AI Explanation Engine |

## License

Private — all rights reserved.
