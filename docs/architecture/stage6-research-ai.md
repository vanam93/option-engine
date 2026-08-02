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
