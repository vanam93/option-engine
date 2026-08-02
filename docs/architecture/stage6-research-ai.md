# Stage 6 — Research AI

Stage 6 extends the platform with **research-backed AI surfaces** for live observation and future decision support. Stage 6 components consume **only canonical events** from prior stages. No Stage 6 component may import providers, market gateway code, broker adapters, or mutate frozen engine state.

**This platform is not an automated trading system.** Console output exists solely for live observation of recommendation lifecycle. No broker integration, OMS, or live order placement is planned.

## Pipeline

```text
Stage 5 Intelligence Layer (frozen)
    ↓  recommendation.delivery.updated
Stage 6 Research AI Layer
    ↓
Recommendation Console         ← Phase 1 (implemented)
Historical Backtest Runner       ← Phase 2 (implemented)
(Future) AI Explanation Surface
(Future) Research Query APIs
```

---

## Stage 6 Phase 1 — Recommendation Console

Phase 1 introduces the **Recommendation Console** (`internal/console`). This engine is a **read-only terminal observer** that renders consolidated delivery documents for live operational visibility.

### Purpose

Provide production-grade terminal output for live observation of recommendation lifecycle events. No UI, dashboard, web, mobile, or notification channels.

### Architecture

```text
recommendation.delivery.updated
            ↓
Recommendation Console (internal/console)
            ↓
Pretty formatted terminal output
```

| File | Responsibility |
|------|----------------|
| `engine.go` | Lifecycle, subscribe, consume, render dispatch, shutdown |
| `renderer.go` | Block rendering and in-place overwrite for existing IDs |
| `formatter.go` | Structured field formatting for terminal blocks |
| `config.go` | Runtime configuration |
| `health.go` | Observability counters |
| `errors.go` | Structured errors |
| `engine_test.go` | Rendering, updates, timeline, shutdown tests |

### Responsibilities

| Responsibility | Detail |
|----------------|--------|
| Consume events | Only `recommendation.delivery.updated` |
| Render blocks | One formatted block per `RecommendationID` |
| Incremental updates | Overwrite previous block when ID already exists |
| Periodic refresh | Re-render tracked recommendations on `refresh_interval` |
| Read-only | No EventBus writes |
| Health | `recommendation_console` metrics |

### Display fields

Each recommendation block includes:

- Recommendation ID, Time, Symbol, Option, Strategy
- Recommendation Level, Confidence, Current Status
- Optimization Score, Walk Forward, Monte Carlo, Historical Win Rate
- Entry, Current Price, Target, Stop Loss, PnL
- Quality, Feedback, Timeline, Research Summary

Fields not present in the delivery document (e.g. Target, Stop Loss) render as `—`.

### Incremental update model

- Engine stores the latest `DeliveryDocument` per `RecommendationID`.
- First render for an ID counts as `documents_rendered`.
- Subsequent renders for the same ID count as `updates_rendered` and overwrite the prior terminal block using ANSI cursor control.
- `refresh_interval` triggers a full re-render of all tracked recommendations.

### Event contract

**Input:** `recommendation.delivery.updated` only

```json
{
  "recommendation_id": "REC-20260802-NIFTY-000001",
  "symbol": "NIFTY",
  "timeframe": "1m",
  "strategy": "ema_cross",
  "document": { ... },
  "generated_at": "2026-08-02T10:30:00Z"
}
```

**Output:** Terminal stdout (no EventBus publish)

### Configuration

```yaml
console:
  enabled: true
  refresh_interval: 1s
```

### Health

Component name: `recommendation_console`

| Metric | Description |
|--------|-------------|
| `documents_rendered` | First-time recommendation blocks rendered |
| `updates_rendered` | Overwritten blocks for existing recommendation IDs |
| `render_errors` | Failed render attempts |
| `average_render_latency_ms` | Mean render duration |

### Testing

| Test | Validates |
|------|-----------|
| Console rendering | Full block output with required fields |
| Incremental updates | Same ID overwrites and updates health counters |
| Timeline rendering | Timeline section in output |
| Shutdown | Graceful close without goroutine leak |
| Event isolation | Ignores non-delivery events |

### Phase 1 roadmap status

| Phase | Name | Status |
|-------|------|--------|
| 1 | Recommendation Console | ✅ Complete |

---

## Stage 6 Phase 2 — Historical Backtest Runner

Phase 2 introduces the **Historical Backtest Runner** (`internal/backtestrunner`). This engine is the **only entry point for executing historical research** across the full trading intelligence platform. It orchestrates replay, waits for pipeline completion, aggregates downstream outputs, and produces immutable session summaries.

### Purpose

Execute the complete trading intelligence platform over historical market data. This phase is **not** another replay engine — Stage 4 already owns replay. The runner reuses existing replay infrastructure and acts purely as an orchestration layer.

### Architecture

```text
Historical Data
      ↓
Stage 4 Replay Engine (internal/backtest — reused, not duplicated)
      ↓
Stage 2 → Stage 3 → Stage 4 → Stage 5 → Recommendation Delivery
      ↓
Historical Backtest Runner (internal/backtestrunner)
      ↓
Backtest Session Summary
```

| File | Responsibility |
|------|----------------|
| `engine.go` | Session lifecycle, event collection, summary assembly, publish |
| `runner.go` | Replay orchestration via existing backtest engine |
| `provider_binder.go` | Wires replay provider into runtime market pipeline |
| `session.go` | Session model, request validation, run modes |
| `summary.go` | Immutable session summary aggregation |
| `repository.go` | Completed session storage and lookup |
| `events.go` | `backtest.session.started` / `backtest.session.completed` payloads |
| `config.go` | Runtime configuration |
| `health.go` | Observability counters |
| `errors.go` | Structured errors |
| `engine_test.go` | Session, summary, repository, concurrency, shutdown tests |

### Responsibilities

| Responsibility | Detail |
|----------------|--------|
| Start sessions | Configure replay period, symbols, expiries, run mode |
| Orchestrate replay | Reuse `internal/backtest` replay provider — no duplicate replay engine |
| Wait for completion | Poll replay status until completed or stopped |
| Collect outputs | Delivery documents, research, quality, feedback, alerts, optimization, walk-forward, Monte Carlo |
| Produce summary | One immutable `SessionSummary` per session |
| Publish events | `backtest.session.started`, `backtest.session.completed` |
| Repository | Store and query completed sessions |
| Read-only orchestration | No duplicated analytics, recommendation, or replay logic |

### Session lifecycle

1. `StartSession(request)` validates input and checks concurrent session limit.
2. Runner publishes `backtest.session.started`.
3. Session collector subscribes to downstream intelligence events.
4. Replay runner binds the backtest provider, connects, subscribes symbols, and waits for replay completion.
5. Runner merges collector snapshots with delivery repository documents.
6. Runner builds immutable `SessionSummary` and stores completed `Session`.
7. Runner publishes `backtest.session.completed`.

### Backtest session

Every run receives:

| Field | Description |
|-------|-------------|
| `BacktestID` | Unique session identifier (`BT-{timestamp}-{uuid}`) |
| `StartTime` | Configured replay start |
| `EndTime` | Configured replay end |
| `ReplayDuration` | Wall-clock replay execution time |
| `Status` | `PENDING`, `RUNNING`, `COMPLETED`, `FAILED` |
| `CreatedAt` | Session creation timestamp |
| `CompletedAt` | Session completion timestamp |

### Session summary

Produces:

- Recommendations generated / closed
- BUY, WATCH, AVOID counts
- Average / highest / lowest confidence
- Average holding time
- Best / worst recommendation by return
- Average return, win rate, loss rate
- Quality distribution
- Feedback summary
- Strategy / symbol / timeframe distributions
- Alerts generated
- Research reports generated
- Optimization / walk-forward / Monte Carlo run counts

### Run modes

| Mode | Description |
|------|-------------|
| `single_day` | One trading day (start and end on same calendar day) |
| `date_range` | Arbitrary start/end window |
| `multi_day` | Multiple trading days within the configured range |

Supports multiple symbols and optional expiries in the session request.

### Replay integration

- Uses `backtest.Engine` and `BacktestReplayRunner` — the same replay infrastructure used by experiments and walk-forward.
- `ManagerBinder` calls `providers.Manager.InitWithProvider` to wire the replay provider into the existing gateway pipeline.
- Does **not** subscribe directly to market data.
- Does **not** create a duplicate replay provider or replay engine.

### Repository

| Method | Description |
|--------|-------------|
| `GetSession(id)` | Lookup by backtest ID |
| `LatestSession()` | Most recently stored session |
| `ListSessions()` | All sessions in creation order |
| `ListByDate(day)` | Sessions overlapping a calendar day |
| `ListBySymbol(symbol)` | Sessions containing a symbol |

Protected by `sync.RWMutex`.

### Event contract

**Outputs (append-only):**

`backtest.session.started`

```json
{
  "backtest_id": "BT-20260802T091500-abc12345",
  "start_time": "2026-08-02T09:15:00Z",
  "end_time": "2026-08-02T15:30:00Z",
  "symbols": ["NIFTY"],
  "mode": "single_day",
  "started_at": "2026-08-02T08:00:00Z"
}
```

`backtest.session.completed`

```json
{
  "backtest_id": "BT-20260802T091500-abc12345",
  "status": "COMPLETED",
  "replay_duration": 1200000000,
  "summary": { ... },
  "completed_at": "2026-08-02T08:02:00Z"
}
```

### Configuration

```yaml
backtest_runner:
  enabled: true
  auto_start: false
  concurrent_sessions: 1
```

When `auto_start: true`, the runner starts one session from `backtest` configuration at engine start.

### Health

Component name: `backtest_runner`

| Metric | Description |
|--------|-------------|
| `sessions_started` | Total sessions initiated |
| `sessions_completed` | Successfully completed sessions |
| `sessions_failed` | Failed sessions |
| `active_sessions` | Currently running sessions |
| `average_session_duration_ms` | Mean session wall-clock duration |
| `recommendations_processed` | Recommendations aggregated across completed sessions |

### Thread safety

| Component | Mechanism |
|-----------|-----------|
| Engine session counter | `sync.Mutex` |
| Repository | `sync.RWMutex` |
| Replay runner | `sync.Mutex` serializes replay execution |
| Session collector | `sync.Mutex` on snapshot fields |
| Concurrent sessions | Configurable semaphore via `concurrent_sessions` |

Every goroutine accepts `context.Context` and terminates on shutdown. No goroutine leaks.

### Failure handling

- Invalid session requests return structured errors without starting replay.
- Concurrent limit breaches return `ErrConcurrentLimit`.
- Replay failures mark session `FAILED`, still persist partial summary, and publish `backtest.session.completed` with error detail.
- Shutdown cancels in-flight session context and waits for active sessions.

### Trade-offs

| Decision | Rationale |
|----------|-----------|
| Reuse `internal/backtest` | Avoids duplicate replay engine; honors Stage 4 ownership |
| Provider bind per session | Allows session-specific replay engines without modifying frozen stages |
| Event collector + delivery repo | Ensures summary uses both live session events and final delivery read model |
| Serial replay mutex | Prevents concurrent replay on shared provider instances |
| `auto_start: false` default | Production services start sessions explicitly; batch jobs opt in |

### Future Strategy Laboratory integration

Future Strategy Laboratory, AI Research, and Report Generation phases **must** invoke `BacktestRunnerEngine.StartSession` rather than creating alternative execution paths. This runner is the single historical research entry point.

### Testing

| Test | Validates |
|------|-----------|
| Single session | Start/complete lifecycle and event publish |
| Multiple sessions | Sequential session storage |
| Session summary | Aggregation metrics |
| Repository lookup | Get/Latest/List/ByDate/BySymbol |
| Graceful shutdown | Clean close with active session |
| Replay completion | Runner returns after replay |
| Concurrent session protection | Limit enforcement |
| Health metrics | Counters after completion |

### Phase 2 roadmap status

| Phase | Name | Status |
|-------|------|--------|
| 2 | Historical Backtest Runner | ✅ Complete |
