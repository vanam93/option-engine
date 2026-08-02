# Stage 4 — Research & Optimization Layer

Stage 4 consumes **only canonical events** from upstream analytics and backtest pipelines. No Stage 4 component may import providers, gateway code, broker adapters, or mutate Stage 2/3 engine state.

## Pipeline

```text
Backtest Replay Engine     ← Phase 1 (implemented, frozen)
    ↓  MarketDataReceived (via gateway)
Stage 3 Analytics Pipeline (frozen)
    ↓  … → portfolio.updated
Performance Engine         ← Stage 3 (frozen)
    ↓  performance.updated
Optimization Engine        ← Phase 2 (implemented)
    ↓  optimization.updated
Strategy Evaluation / Ranking Report
```

## 1. Stage 4 Goals

- Enable **offline research** by replaying historical market data through the full analytics pipeline.
- Evaluate completed backtest runs and **rank strategy performance** across symbols, timeframes, and parameter sets.
- Provide a modular foundation for future Stage 4 phases: walk-forward analysis, parameter sweeps, Monte Carlo simulation, and report export.
- Remain **analytics-only**: no trade execution, no strategy mutation, no broker interaction, no live-trading side effects.

## 2. Scope

### In scope (Phase 1 — Backtest Replay)

- Historical candle loading and replay through the provider contract.
- Replay clock integration for deterministic simulated time.
- Health reporting for replay lifecycle and throughput.

### In scope (Phase 2 — Strategy Optimization)

- Consume `performance.updated` events exclusively.
- Evaluate strategy performance per `(strategy, symbol, timeframe, parameters)` key.
- Compute ranking metrics and a configurable optimization score.
- Publish `optimization.updated` events with metrics, score, and rank.
- Maintain in-memory evaluations, rankings, and score history.

### Out of scope (current phases)

- Live parameter mutation or auto-tuning.
- Broker order routing.
- Persistent optimization result storage (future phase).
- Walk-forward and parameter grid search (future phases).

## 3. Design Principles

1. **Event-only integration** — downstream engines consume bus events; never read another engine's internal cache.
2. **Append-only contracts** — event payloads and public interfaces grow via new fields; breaking changes are forbidden.
3. **Single ownership** — each engine owns its mutable state; expose immutable read models only.
4. **Incremental computation** — apply updates on each event; avoid full recomputation.
5. **Configurable scoring** — weights and penalties are runtime-configurable, not hardcoded.
6. **Graceful lifecycle** — context-driven shutdown, WaitGroup for goroutines, no leaks.
7. **Stage freeze** — Phase 1 replay engine is frozen; Phase 2 does not modify Stage 2, Stage 3, or Phase 1.

## 4. Architecture Overview

```mermaid
flowchart TB
    subgraph Stage4["Stage 4 — Research & Optimization"]
        BT[Backtest Replay Engine]
        OPT[Optimization Engine]
    end

    subgraph Stage3["Stage 3 — Analytics (frozen)"]
        CANDLE[Candle Engine]
        IND[Indicator Engine]
        SIG[Signal Engine]
        STRAT[Strategy Engine]
        RISK[Risk Engine]
        PAPER[Paper Execution]
        PORT[Portfolio Engine]
        PERF[Performance Engine]
    end

    subgraph Stage2["Stage 2 — Market Engine (frozen)"]
        GW[Gateway]
        BUS[EventBus]
    end

    BT -->|MarketDataReceived| GW
    GW --> BUS
    BUS --> CANDLE --> IND --> SIG --> STRAT --> RISK --> PAPER --> PORT --> PERF
    PERF -->|performance.updated| OPT
    OPT -->|optimization.updated| BUS
```

## 5. Component Responsibilities

### Backtest Replay Engine (Phase 1, frozen)

| Responsibility | Detail |
|----------------|--------|
| Load historical candles | From file path or in-memory slice |
| Replay provider | Implements `api.Provider`; emits ticks at configured speed |
| Replay clock | Drives simulated time for the full pipeline |
| Health | `backtest_engine` with processed candles, status, speed |

### Optimization Engine (Phase 2)

| Responsibility | Detail |
|----------------|--------|
| Subscribe | `performance.updated` events only |
| Parse | Strategy, symbol, timeframe, parameters, and performance metrics |
| Evaluate | Compute derived metrics (profit factor, expectancy, risk/reward, etc.) |
| Score | Apply configurable weighted scoring formula |
| Rank | Order evaluations by score; assign rank per key |
| Publish | `optimization.updated` with metrics, score, rank |
| State | In-memory evaluations, rankings, historical scores |
| Health | `optimization_engine` with processed counts |

## 6. Event Flow

### Backtest → Optimization (end-to-end)

```mermaid
sequenceDiagram
    participant BT as Backtest Replay
    participant GW as Gateway
    participant S3 as Stage 3 Pipeline
    participant PERF as Performance Engine
    participant OPT as Optimization Engine
    participant BUS as EventBus

    BT->>GW: MarketDataReceived (replay tick)
    GW->>BUS: Publish tick
    BUS->>S3: Candle → … → Portfolio
    S3->>BUS: portfolio.updated
    BUS->>PERF: portfolio.updated
    PERF->>BUS: performance.updated
    BUS->>OPT: performance.updated
    OPT->>OPT: Evaluate, score, rank
    OPT->>BUS: optimization.updated
```

### Optimization event handling

```mermaid
sequenceDiagram
    participant BUS as EventBus
    participant OPT as Optimization Engine
    participant CACHE as Cache
    participant EVAL as Evaluator

    BUS->>OPT: performance.updated
    OPT->>OPT: parse payload
    OPT->>CACHE: Apply(update)
    CACHE->>EVAL: ComputeMetrics(state)
    EVAL-->>CACHE: EvaluationMetrics
    CACHE->>EVAL: Score(metrics, weights)
    EVAL-->>CACHE: score
    CACHE->>CACHE: Re-rank all evaluations
    OPT->>BUS: optimization.updated (per key)
```

## 7. Package Layout

```text
internal/
├── backtest/                  # Phase 1 — frozen
│   ├── engine.go
│   ├── config.go
│   ├── replay.go
│   ├── loader.go
│   ├── events.go
│   └── health.go
└── optimization/              # Phase 2
    ├── engine.go              # Lifecycle, bus subscription, publish
    ├── config.go              # Enabled flag, scoring weights
    ├── evaluator.go           # Metric computation and scoring
    ├── cache.go               # Thread-safe evaluations and rankings
    ├── events.go              # Input/output event payloads
    └── health.go              # Health reporter
```

Configuration wiring lives in `internal/infrastructure/config/optimization.go`.

## 8. Event Contracts

All contracts are **append-only**. New fields may be added; existing fields must not change semantics.

### Input: `performance.updated`

Consumed exclusively by the Optimization Engine.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `strategy` | string | no | Strategy identifier; defaults to `"default"` |
| `symbol` | string | no | Instrument symbol; defaults to `"portfolio"` |
| `timeframe` | string | no | Bar timeframe; defaults to `"1m"` |
| `parameters` | string | no | Serialized parameter set identifier |
| `total_trades` | int | yes | Completed trade count |
| `win_rate` | float64 | yes | Fraction of winning trades |
| `realized_pnl` | float64 | yes | Cumulative realized PnL |
| `unrealized_pnl` | float64 | yes | Current unrealized PnL |
| `drawdown` | float64 | yes | Current drawdown from peak |
| `profit_factor` | float64 | no | Optional; computed if absent |
| `max_drawdown` | float64 | no | Optional; tracked if absent |
| `timestamp` | time | yes | Event timestamp |

### Output: `optimization.updated`

Published after each evaluation update.

| Field | Type | Description |
|-------|------|-------------|
| `strategy` | string | Strategy identifier |
| `symbol` | string | Instrument symbol |
| `timeframe` | string | Bar timeframe |
| `parameters` | string | Parameter set identifier |
| `metrics` | object | Full evaluation metrics (see below) |
| `score` | float64 | Configurable optimization score |
| `rank` | int | Rank among all evaluated keys (1 = best) |
| `timestamp` | time | Evaluation timestamp |

#### `metrics` object

| Field | Type | Description |
|-------|------|-------------|
| `total_trades` | int | Completed trades |
| `net_pnl` | float64 | Realized + unrealized PnL |
| `realized_pnl` | float64 | Cumulative realized PnL |
| `win_rate` | float64 | Winning trade fraction |
| `profit_factor` | float64 | Gross profit / gross loss |
| `average_trade` | float64 | Mean PnL per trade |
| `expectancy` | float64 | Expected value per trade |
| `max_drawdown` | float64 | Maximum peak-to-trough decline |
| `risk_reward` | float64 | Average win / average loss |
| `sharpe_ratio` | float64 | Placeholder (0 until return series available) |

## 9. Engine Lifecycle

### Startup order

Optimization Engine subscribes **before** Performance Engine starts publishing:

```text
Optimization → Performance → Portfolio → Paper → Risk → Strategy → Signal → Indicator → Candle → Gateway
```

### Shutdown order (reverse)

```text
Gateway → Candle → Indicator → Signal → Strategy → Risk → Paper → Portfolio → Performance → Optimization
```

### Lifecycle steps

1. `New(cfg, bus, clk)` — validate config, allocate cache.
2. `Start(ctx)` — subscribe to `performance.updated`, launch consumer goroutine.
3. Consumer loop — parse, evaluate, score, rank, publish.
4. `Close()` — cancel context, drain subscription, wait on WaitGroup, close subscription.

## 10. Configuration

```yaml
optimization:
  enabled: true
  subscriber_buffer: 256

  scoring:
    profit_factor_weight: 0.40
    win_rate_weight: 0.30
    expectancy_weight: 0.20
    drawdown_penalty: 0.10
```

| Key | Default | Description |
|-----|---------|-------------|
| `enabled` | `true` | Enable optimization engine |
| `subscriber_buffer` | `256` | Event bus subscriber buffer |
| `scoring.profit_factor_weight` | `0.40` | Weight for normalized profit factor |
| `scoring.win_rate_weight` | `0.30` | Weight for win rate |
| `scoring.expectancy_weight` | `0.20` | Weight for normalized expectancy |
| `scoring.drawdown_penalty` | `0.10` | Penalty for normalized max drawdown |

### Scoring formula

```text
score = profit_factor_weight × norm(profit_factor)
      + win_rate_weight     × win_rate
      + expectancy_weight   × norm(expectancy)
      - drawdown_penalty    × norm(max_drawdown)
```

Normalization clamps values to `[0, 1]` using configurable reference scales (profit factor cap 5.0, expectancy cap 100.0, drawdown cap 1000.0).

## 11. Health Reporting

`GET /health/components` includes `optimization_engine`:

| Detail key | Description |
|------------|-------------|
| `enabled` | Whether engine is configured on |
| `evaluations_processed` | Total performance events processed |
| `reports_generated` | Total optimization.updated events published |
| `strategies_evaluated` | Distinct strategy keys evaluated |
| `rankings_generated` | Total re-ranking operations performed |
| `dropped` | Subscription dropped event count |

Status is `degraded` when disconnected or events are dropped.

## 12. Thread Safety

- A **single consumer goroutine** processes bus events sequentially.
- The **cache mutex** protects evaluations, rankings, and score history maps.
- Health counters use **atomic** operations.
- `State()` returns an **immutable copy** of evaluations and rankings.
- No lock is held during bus publish.

## 13. Failure Handling

| Failure | Behavior |
|---------|----------|
| Malformed `performance.updated` payload | Skip silently; no publish |
| Missing optional dimension fields | Apply defaults (`default`, `portfolio`, `1m`) |
| Bus publish error | Skip; health counter still increments processed |
| Shutdown mid-event | Drain subscription channel before exit |
| Disabled engine | `Start`/`Close` are no-ops |

No panics for runtime failures. Structured errors returned from `New` and `Validate` only.

## 14. Backtest Integration

The Optimization Engine does **not** import the backtest package. Integration is purely event-driven:

1. Enable `backtest.enabled` to replay historical candles.
2. Stage 3 pipeline processes replayed ticks through execution and portfolio.
3. Performance Engine publishes `performance.updated` at each portfolio snapshot.
4. Optimization Engine evaluates and ranks the resulting metrics.

For multi-strategy backtests, upstream callers should include `strategy`, `symbol`, `timeframe`, and `parameters` in the `performance.updated` payload (append-only extension). Until Performance Engine emits these fields, the Optimization Engine applies documented defaults.

## 15. Optimization Pipeline

```text
performance.updated
    ↓
Parse & key (strategy, symbol, timeframe, parameters)
    ↓
Cache.Apply — update incremental trade state
    ↓
Evaluator.ComputeMetrics — derive all metrics
    ↓
Evaluator.Score — apply weighted formula
    ↓
Cache.ReRank — sort all evaluations by score
    ↓
optimization.updated (metrics + score + rank)
```

## 16. Future Stage 4 Phases

| Phase | Name | Consumes | Produces |
|-------|------|----------|----------|
| 1 | Backtest Replay | Historical data | `MarketDataReceived` |
| 2 | Strategy Optimization | `performance.updated` | `optimization.updated` |
| 3 | Parameter Sweep | Config grid + replay | Multiple `performance.updated` streams |
| 4 | Walk-Forward Analysis | Rolling replay windows | `optimization.updated` series |
| 5 | Report Export | `optimization.updated` | CSV/JSON reports |
| 6 | Monte Carlo | Trade shuffle / resample | Distribution metrics |

Each future phase must follow the same event-only, append-only, single-ownership rules.

## 17. Sequence Diagrams

### Multi-strategy evaluation

```mermaid
sequenceDiagram
    participant PERF as Performance Engine
    participant OPT as Optimization Engine
    participant CACHE as Cache

    PERF->>OPT: performance.updated (strategy=A)
    OPT->>CACHE: Apply A → score 0.82, rank 1
    OPT->>OPT: publish optimization.updated (A)

    PERF->>OPT: performance.updated (strategy=B)
    OPT->>CACHE: Apply B → score 0.45, rank 2
    OPT->>CACHE: Re-rank A=1, B=2
    OPT->>OPT: publish optimization.updated (B)
```

### Graceful shutdown

```mermaid
sequenceDiagram
    participant App
    participant PERF as Performance Engine
    participant OPT as Optimization Engine

    App->>PERF: Close()
    PERF->>PERF: cancel, drain, wait
    App->>OPT: Close()
    OPT->>OPT: cancel context
    OPT->>OPT: drain subscription
    OPT->>OPT: wg.Wait()
    OPT->>OPT: close subscription
```

## 18. Component Diagram

```mermaid
flowchart LR
    subgraph optimization_pkg["internal/optimization"]
        ENG[Engine]
        CFG[Config]
        CACHE[Cache]
        EVAL[Evaluator]
        EVT[Events]
        HLTH[Health]
    end

    ENG --> CFG
    ENG --> CACHE
    ENG --> EVAL
    ENG --> EVT
    ENG --> HLTH
    CACHE --> EVAL
```

## 19. Data Flow

```text
┌─────────────────┐     performance.updated      ┌──────────────────┐
│ Performance     │ ────────────────────────────►│ Optimization     │
│ Engine (S3)     │                              │ Engine (S4)      │
└─────────────────┘                              └────────┬─────────┘
                                                          │
                        ┌─────────────────────────────────┤
                        ▼                                 ▼
                 ┌─────────────┐                  ┌───────────────┐
                 │ Cache       │                  │ optimization  │
                 │ evaluations │                  │ .updated      │
                 │ rankings    │                  └───────────────┘
                 │ score hist  │
                 └─────────────┘
```

## 20. Extension Guidelines

1. Add new scoring dimensions via config weights, not hardcoded branches.
2. Add new metrics to `EvaluationMetrics` as optional JSON fields.
3. New event types use new `events.Type` constants; never repurpose existing types.
4. Persistent storage belongs in a future phase with its own package.
5. Do not import `internal/backtest` or Stage 3 engine packages from optimization.

## 21. Testing Strategy

| Test | Validates |
|------|-----------|
| Profitable strategy | High optimization score |
| High drawdown strategy | Lower score due to drawdown penalty |
| Multiple strategies | Correct relative ranking |
| Engine lifecycle | Subscribe, publish, graceful close |
| Scoring weights | Configurable weight application |

Run `go build ./...` and `go test ./...` before completion.

## 22. Non-Goals

- Modifying live strategy parameters.
- Executing trades or interacting with brokers.
- Reading Performance Engine internal cache.
- Consuming provider or gateway events directly.
- O(n²) ranking on every event (ranking is O(n log n) over evaluation keys).

## 23. Design Decisions

| Decision | Rationale |
|----------|-----------|
| Consume only `performance.updated` | Maintains Stage 3 freeze; single upstream contract |
| Default dimension keys | Allows operation before Performance Engine emits strategy/symbol fields |
| In-memory state only | Phase 2 scope; persistence deferred |
| Modular `Evaluator` | Scoring formula can evolve without cache changes |
| Sharpe placeholder | Requires return series not yet in performance events |
| Re-rank on every update | Rankings stay current for multi-strategy comparisons |

## 24. Trade-offs

| Trade-off | Choice | Cost | Benefit |
|-----------|--------|------|---------|
| No persistence | In-memory only | Results lost on restart | Simple, fast, no DB dependency |
| Default dimension keys | `"default"` / `"portfolio"` | Less precise without upstream fields | Works with current Performance payload |
| Full re-rank per event | O(n log n) sort | Minor CPU at scale | Always-current rankings |
| Sharpe placeholder | Returns 0 | Incomplete risk metric | No false precision |
| Incremental trade tracking | Delta-based gross profit/loss | Approximation when events batch | No Stage 3 modification needed |

## 25. Experiment Framework

Phase 3 introduces an **Experiment Framework** that sits above the Backtest Replay and Optimization engines. The framework is purely orchestration: it generates parameter combinations, schedules backtest runs, correlates results via event metadata, and publishes aggregated experiment outcomes.

### Core concepts

| Concept | Description |
|---------|-------------|
| **Experiment** | A logical batch of parameter-sweep runs sharing an `experiment_id` |
| **ExperimentRun** | A single backtest execution with unique `run_id` and parameter set |
| **ParameterSet** | Serializable map of strategy/risk/indicator parameters for one run |
| **BacktestSession** | A single replay invocation through the existing Backtest Engine |
| **ExperimentResult** | Collected `optimization.updated` outcome correlated to a `run_id` |

### Design rules

- The Experiment Engine **never** duplicates analytics, backtest replay, or optimization logic.
- All execution flows through existing engines via event bus and injected runners.
- Experiment state is in-memory; persistence is deferred to Phase 5 (Report Export).
- Correlation uses append-only `performance.updated` metadata (`run_id`, `experiment_id`, `parameter_set`).

## 26. Parameter Sweep Architecture

Parameter sweeps generate a **Cartesian product** across configured ranges:

```text
symbols × timeframes × ema_fast × ema_slow × rsi_period × rsi_overbought ×
rsi_oversold × macd_fast × macd_slow × macd_signal × min_confidence × max_positions
```

### Generator responsibilities

1. Read `experiments.parameter_ranges` from configuration.
2. Expand ranges into discrete `ParameterSet` values.
3. Assign each combination a unique `run_id` within the parent `experiment_id`.
4. Serialize parameters into `parameter_set` JSON for event correlation.

### Sweep constraints

- Empty ranges are skipped (dimension omitted from product).
- `max_concurrent_runs` caps parallel backtest sessions.
- Duplicate `run_id` values are rejected by the scheduler.

```mermaid
flowchart TD
    CFG[Config parameter_ranges] --> GEN[Generator]
    GEN --> MATRIX[Experiment Matrix]
    MATRIX --> Q[Scheduler Queue]
    Q --> W1[Worker 1]
    Q --> W2[Worker 2]
    Q --> WN[Worker N]
    W1 --> BT[Backtest Engine]
    W2 --> BT
    WN --> BT
```

## 27. Experiment Lifecycle

```text
CREATED → QUEUED → RUNNING → COMPLETED
                          └→ FAILED
```

| State | Description |
|-------|-------------|
| `CREATED` | Run generated and registered in cache |
| `QUEUED` | Waiting for an available worker slot |
| `RUNNING` | Backtest session active; `run_id` registered as in-flight |
| `COMPLETED` | `optimization.updated` received; `experiment.completed` published |
| `FAILED` | Backtest or correlation error; run marked failed |

### Lifecycle sequence

```mermaid
sequenceDiagram
    participant EXP as Experiment Engine
    participant SCH as Scheduler
    participant BT as Backtest Engine
    participant PERF as Performance Engine
    participant OPT as Optimization Engine

    EXP->>EXP: Generate experiment matrix
    EXP->>SCH: Enqueue runs
    SCH->>BT: Execute run (run_id)
    BT->>PERF: portfolio.updated → performance.updated
    Note over PERF: Carries run_id, experiment_id, parameter_set
    PERF->>OPT: performance.updated
    OPT->>EXP: optimization.updated
    EXP->>EXP: Correlate by run_id
    EXP->>EXP: Publish experiment.completed
```

## 28. Backtest Session Model

Each `ExperimentRun` maps to one **BacktestSession**:

- One invocation of the existing Backtest Replay Engine (no fork or copy of replay logic).
- Session metadata (`run_id`, `experiment_id`, `backtest_id`) flows through `performance.updated`.
- Sessions are isolated by serializing concurrent access per worker when sharing a single engine instance.
- For parallel execution, the DI layer may inject a factory that creates independent `backtest.Engine` instances per run.

```text
ExperimentRun
    ↓
BacktestSession.Run(ctx)
    ↓
ReplayProvider.Connect → replayLoop → MarketDataReceived
    ↓
Stage 3 pipeline (unchanged)
    ↓
performance.updated (tagged with run_id)
```

## 29. Run ID / Experiment ID

| Identifier | Scope | Format |
|------------|-------|--------|
| `experiment_id` | Entire parameter sweep batch | UUID v4 |
| `run_id` | Single backtest execution | UUID v4 |
| `backtest_id` | Logical backtest session label | Optional string |
| `parameter_set` | Serialized parameter JSON | JSON string |

### Correlation contract

1. Experiment Engine assigns `experiment_id` at batch creation.
2. Generator assigns unique `run_id` per matrix row.
3. Runner tags the session; Performance Engine forwards tags in `performance.updated`.
4. Optimization Engine keys evaluations by `parameter_set` (unchanged).
5. Experiment Engine correlates `optimization.updated` → `run_id` via `parameter_set` content.

## 30. Optimization Aggregation

The Experiment Engine aggregates optimization results at the experiment level:

```text
optimization.updated (per run)
    ↓
Experiment Cache.StoreResult(run_id, score, metrics, rank)
    ↓
Experiment Cache.ReRank() — sort completed runs by optimization score
    ↓
experiment.completed (per run, includes experiment-level rank)
```

Aggregation rules:

- Only `COMPLETED` runs participate in experiment rankings.
- Rank 1 = highest `optimization_score` within the experiment batch.
- Failed runs are excluded from rankings but counted in health metrics.

## 31. Parallel Execution Model

```mermaid
flowchart LR
    Q[Run Queue] --> SEM[Semaphore max_concurrent_runs]
    SEM --> W1[Worker 1]
    SEM --> W2[Worker 2]
    SEM --> W3[Worker 3]
    SEM --> W4[Worker 4]
```

| Mechanism | Purpose |
|-----------|---------|
| `parallel_workers` | Number of worker goroutines |
| `max_concurrent_runs` | Semaphore limit on simultaneous backtests |
| `running` set | Prevents duplicate execution of the same `run_id` |
| Context cancellation | Graceful shutdown drains queue without new dispatches |

Workers:

1. Dequeue next `QUEUED` run.
2. Acquire semaphore slot.
3. Register `run_id` in `running` set.
4. Invoke `BacktestRunner.Execute(ctx, run)`.
5. Await `optimization.updated` correlation (or timeout).
6. Release slot, remove from `running`, publish `experiment.completed`.

## 32. Result Storage Model

Phase 3 uses **in-memory storage only**:

| Store | Content |
|-------|---------|
| `experiments` | Experiment metadata and batch state |
| `runs` | All `ExperimentRun` records by `run_id` |
| `running` | Currently executing `run_id` set |
| `completed` | Finished runs with optimization results |
| `rankings` | Sorted completed runs by score |

`State()` returns immutable snapshots. Persistence to PostgreSQL or file export is deferred to Phase 5.

## 33. Ranking Pipeline

```text
1. optimization.updated received
2. Match run_id from parameter_set JSON
3. Store RunResult { score, metrics, rank from optimization }
4. Re-rank all completed runs in experiment (descending score)
5. Publish experiment.completed with updated rank
```

Tie-breaking: higher score wins; equal scores broken by `run_id` lexicographic order.

## 34. Future Walk-Forward Validation

Walk-forward analysis will extend the Experiment Framework without architectural redesign:

| Extension | Integration point |
|-----------|-------------------|
| Rolling time windows | Generator adds `window_start` / `window_end` to `ParameterSet` |
| In-sample / out-of-sample split | Scheduler runs sequential windows per parameter set |
| Stability scoring | Optimization Aggregation computes cross-window score variance |
| New event | `walkforward.completed` (append-only) |

The existing `experiment_id` / `run_id` model accommodates window suffixes (e.g. `run_id_w3`).

## 35. Future Market Regime Detection

Market regime detection will consume Stage 3 indicator events and tag experiments:

| Extension | Integration point |
|-----------|-------------------|
| Regime classifier | New analytics consumer of `IndicatorUpdated` (Stage 3 extension) |
| Regime label in `performance.updated` | Append-only field `regime` |
| Regime-filtered ranking | Experiment Aggregation groups results by regime |
| New event | `regime.detected` (append-only) |

No changes to Experiment Engine orchestration logic; only new optional metadata fields.

## 36. Future Adaptive Strategy Selection

Adaptive strategy selection will close the loop between optimization rankings and live strategy configuration:

```text
optimization.updated → experiment.completed → strategy.selection.recommended
```

| Component | Role |
|-----------|------|
| Selection Engine (Phase 6+) | Consumes `experiment.completed` rankings |
| Recommendation event | Publishes top-ranked parameter set per symbol/timeframe |
| Live Strategy Engine | Consumes recommendations (read-only; no auto-mutation) |

The Experiment Framework provides ranked candidates; adaptive selection remains a separate, opt-in downstream consumer.

## 37. Phase 3 — Experiment Engine

### Responsibility

- Generate parameter sweep matrix from configuration.
- Schedule and execute backtest runs via injected `BacktestRunner`.
- Subscribe to `optimization.updated` for result collection.
- Publish `experiment.completed` per finished run.
- Maintain in-memory experiment state and rankings.

### Packages

| Package | Role |
|---------|------|
| `internal/experiments` | Generator, scheduler, cache, orchestration |
| `internal/infrastructure/config/experiments.go` | Configuration mapping |

### Configuration

```yaml
experiments:
  enabled: true
  parallel_workers: 4
  max_concurrent_runs: 4
  symbols:
    - NIFTY
  timeframes:
    - 5m
  parameter_ranges:
    ema_fast: [5, 9, 12]
    ema_slow: [21, 34, 50]
    rsi_period: [14, 21]
```

### Lifecycle

1. Experiment engine starts **before** optimization engine (subscribes first).
2. On `Start`, generate experiment batch and enqueue runs.
3. Workers invoke backtest runner; collect `optimization.updated`.
4. On shutdown: cancel workers, drain queue, wait on WaitGroup.

### Event payload (`experiment.completed`)

- `experiment_id`, `run_id`, `strategy`, `parameters`, `optimization_score`, `rank`, `metrics`, `timestamp`

### Health

`GET /health/components` includes `experiment_engine` with `experiments_created`, `runs_started`, `runs_completed`, `runs_failed`, `active_workers`, `queue_depth`.

## 38. Walk-Forward Analysis

Phase 4 introduces a **Walk-Forward Analysis Engine** that validates optimized strategies on unseen market data using rolling train/test windows. The engine orchestrates repeated experiment batches across multiple time windows without duplicating analytics, replay, optimization, or experiment generation logic.

### Purpose

- Detect overfitting by measuring out-of-sample (validation) performance.
- Produce cross-window stability metrics and parameter drift signals.
- Publish per-window `walkforward.completed` reports for downstream export and Monte Carlo integration.

### Pipeline

```text
Experiment Engine (parameter matrix generation)
    ↓
Walk-Forward Engine (window orchestration)
    ↓
Backtest Replay (training + validation sessions)
    ↓
Performance Engine
    ↓
Optimization Engine
    ↓
Walk-Forward Report (walkforward.completed)
```

The Walk-Forward Engine consumes `optimization.updated` events for result correlation. It delegates parameter matrix generation to `experiments.GenerateMatrix` and backtest execution to the injected `BacktestRunner`.

## 39. Rolling Window Architecture

Walk-forward analysis partitions historical data into sequential **windows**. Each window contains:

| Boundary | Description |
|----------|-------------|
| `train_start` | Inclusive start of the training (in-sample) period |
| `train_end` | Exclusive end of training; equals `validation_start` |
| `validation_start` | Start of out-of-sample validation period |
| `validation_end` | Exclusive end of validation period |

Windows are generated by `GenerateWindows(dataStart, dataEnd, trainDays, validationDays, stepDays)`:

```text
Window 0: train [T0, T0+30d)  →  validation [T0+30d, T0+40d)
Window 1: train [T0+10d, T0+40d)  →  validation [T0+40d, T0+50d)   (step=10d)
```

When `step_days < train_window_days`, consecutive training windows **overlap**. Validation always immediately follows training with no gap.

```mermaid
flowchart LR
    subgraph W0["Window 0"]
        T0[Train 30d]
        V0[Validate 10d]
        T0 --> V0
    end
    subgraph W1["Window 1 (step=10d)"]
        T1[Train 30d]
        V1[Validate 10d]
        T1 --> V1
    end
    W0 --> W1
```

## 40. Training vs Validation Windows

| Phase | Period | Objective |
|-------|--------|-----------|
| **Training** | `train_start` → `train_end` | Run full parameter sweep via Experiment matrix; rank by optimization score |
| **Validation** | `validation_start` → `validation_end` | Replay best-ranked parameter set on unseen data; measure out-of-sample performance |

Training and validation are tagged in event metadata (`phase: training | validation`) along with `walkforward_id`, `window_index`, and period boundaries. The Optimization Engine keys evaluations unchanged; the Walk-Forward Engine correlates results by `run_id` embedded in serialized `parameters`.

### Per-window flow

1. Generate parameter matrix for training period (`experiments.GenerateMatrix`).
2. Execute all training runs via scheduler + backtest runner.
3. Collect `optimization.updated` events; select highest-scoring parameter set.
4. Execute single validation backtest with best parameters.
5. Collect validation `optimization.updated`; store and publish `walkforward.completed`.

## 41. Window Scheduler

The Window Scheduler dispatches windows **sequentially** to avoid contention on a shared backtest engine instance:

```text
GenerateWindows → Enqueue → Worker → processWindow → next window
```

| Mechanism | Purpose |
|-----------|---------|
| Sequential dispatch | One window active at a time on shared replay |
| `active` set | Prevents duplicate window execution |
| Context cancellation | Graceful shutdown drains in-flight window |
| Training scheduler | Per-window parallel training runs (`parallel_workers`, `max_concurrent_runs`) |

Within each window, training runs use the Experiment Scheduler for parallel execution. Validation is a single sequential run after training completes.

## 42. Evaluation Aggregation

Cross-window aggregation computes summary metrics after each window completes:

```text
walkforward.completed (per window)
    ↓
Cache.StoreResult → rebuildAggregation
    ↓
AggregatedValidation { mean scores, std dev, stability, drift }
```

| Metric | Description |
|--------|-------------|
| `mean_validation_score` | Average out-of-sample optimization score |
| `mean_training_score` | Average in-sample optimization score |
| `training_validation_gap` | Mean training minus mean validation (overfitting signal) |
| `score_std_dev` | Standard deviation of validation scores across windows |
| `stability_score` | `1 - (std_dev / mean)` clamped to `[0, 1]` |
| `parameter_drift` | Coefficient of variation per numeric parameter across windows |

## 43. Stability Metrics

Stability metrics quantify consistency of walk-forward performance:

- **Score variance** — high variance indicates regime sensitivity.
- **Stability score** — normalized consistency measure derived from validation score distribution.
- **Training/validation gap** — persistent positive gap suggests overfitting to training data.

These metrics are computed incrementally in `AggregateValidation` and exposed via `State().Aggregated`.

## 44. Parameter Drift

Parameter drift tracks how the optimal parameter set changes across windows:

```text
For each numeric parameter key (ema_fast, rsi_period, …):
    drift[key] = std_dev(values) / |mean(values)|
```

Metadata keys (`run_id`, `walkforward_id`, `phase`, period boundaries) are excluded from drift calculation. High drift indicates the strategy requires frequent re-optimization.

## 45. Future Monte Carlo Integration

Monte Carlo simulation (Phase 6) will consume walk-forward validation results:

| Extension | Integration point |
|-----------|-------------------|
| Trade resampling | `walkforward.completed` → `performance_metrics` input |
| Return distribution | Aggregated validation scores as baseline |
| Confidence intervals | Cross-window `score_std_dev` as variance prior |
| New event | `montecarlo.completed` (append-only) |

The Walk-Forward Engine provides out-of-sample score distributions without architectural changes to upstream engines.

## 46. Phase 4 — Walk-Forward Engine

### Responsibility

- Generate rolling train/validation windows from configured data range.
- Orchestrate per-window training sweeps using Experiment matrix generation.
- Select best-ranked training parameters; run validation backtest.
- Aggregate cross-window stability and parameter drift metrics.
- Publish `walkforward.completed` per window.

### Packages

| Package | Role |
|---------|------|
| `internal/walkforward` | Windows, scheduler, evaluator, cache, orchestration |
| `internal/infrastructure/config/walkforward.go` | Configuration mapping |

### Configuration

```yaml
walkforward:
  enabled: true
  train_window_days: 30
  validation_window_days: 10
  step_days: 10
  data_start: "2024-01-01T00:00:00Z"
  data_end: "2024-06-30T00:00:00Z"
```

Parameter sweep dimensions reuse `experiments.parameter_ranges` configuration.

### Lifecycle

1. Walk-forward engine starts **before** experiment and optimization engines (subscribes first).
2. On `Start`, generate windows and enqueue for sequential processing.
3. Per window: training sweep → best selection → validation → publish.
4. On shutdown: cancel workers, drain subscription, wait on WaitGroup.

### Event payload (`walkforward.completed`)

| Field | Type | Description |
|-------|------|-------------|
| `walkforward_id` | string | Walk-forward batch identifier |
| `experiment_id` | string | Training experiment batch for this window |
| `run_id` | string | Validation run identifier |
| `train_period` | object | `{ start, end }` training bounds |
| `validation_period` | object | `{ start, end }` validation bounds |
| `best_parameters` | object | Selected parameter set (metadata stripped) |
| `training_score` | float64 | Best in-sample optimization score |
| `validation_score` | float64 | Out-of-sample optimization score |
| `performance_metrics` | object | Full validation evaluation metrics |
| `timestamp` | time | Completion timestamp |

### Health

`GET /health/components` includes `walkforward_engine` with `windows_created`, `windows_completed`, `active_windows`, `validation_runs`, `reports_generated`.

### Package layout

```text
internal/walkforward/
├── engine.go       # Lifecycle, bus subscription, window orchestration
├── config.go       # Window sizes, data range, experiment settings
├── scheduler.go    # Sequential window dispatch
├── windows.go      # Rolling window generation
├── evaluator.go    # Best selection, aggregation, drift
├── cache.go        # Thread-safe window and result state
├── events.go       # Input/output event payloads
└── health.go       # Health reporter
```

### Testing

| Test | Validates |
|------|-----------|
| Window generation | Correct rolling windows with train/validation bounds |
| Best parameter selection | Highest optimization score chosen |
| Validation aggregation | Correct summary metrics and stability |
| `walkforward.completed` | Event emitted with required payload fields |

## 47. Monte Carlo Architecture

Monte Carlo simulation (Phase 5) evaluates strategy robustness by statistically resampling walk-forward validation trade outcomes. It does not execute trades, modify optimization logic, duplicate backtesting, or re-run walk-forward analysis.

### Pipeline

```text
Walk-Forward Engine
        ↓
walkforward.completed
        ↓
Monte Carlo Engine
        ↓
montecarlo.completed
```

### Design principles

| Rule | Enforcement |
|------|-------------|
| Single input event | Subscribes only to `walkforward.completed` |
| No trade execution | Operates on synthesized trade returns from `performance_metrics` |
| No optimization | Does not publish or consume `optimization.updated` |
| No backtest replay | Does not invoke backtest or experiment runners |
| Append-only events | Publishes `montecarlo.completed` without mutating upstream payloads |
| Deterministic option | Optional `random_seed` for reproducible simulation paths |

### Package ownership

| Package | Role |
|---------|------|
| `internal/montecarlo` | Bootstrap resampling, statistics, orchestration, cache, health |
| `internal/infrastructure/config/montecarlo.go` | Configuration mapping |

## 48. Statistical Validation

The Monte Carlo Engine validates whether walk-forward out-of-sample results are robust under trade-order and sampling uncertainty.

### Validation workflow

1. Receive `walkforward.completed` with `performance_metrics`.
2. Extract per-trade PnL samples from aggregated metrics (`ExtractTradeReturns`).
3. Run `N` simulations alternating bootstrap resampling and randomized ordering.
4. Compute return and drawdown distributions across all paths.
5. Derive confidence intervals, profit/loss probabilities, and risk-of-ruin.
6. Publish immutable `montecarlo.completed` report.

### Metrics produced

| Metric | Description |
|--------|-------------|
| Mean return | Average total PnL across simulation paths |
| Median return | 50th percentile total PnL |
| Standard deviation | Dispersion of simulated returns |
| Max drawdown distribution | Per-path peak-to-trough decline |
| Worst / best drawdown | Extremes of drawdown distribution |
| Confidence interval | Lower/upper bounds at configured level (default 95%) |
| Probability of profit | Fraction of paths with positive total return |
| Probability of loss | Fraction of paths with negative total return |
| Risk of ruin | Fraction of paths exceeding drawdown ruin threshold |

## 49. Trade Resampling

Trade returns are synthesized from walk-forward `performance_metrics` when individual trade records are not available on the event payload.

### Extraction algorithm

```text
Input: total_trades, net_pnl, win_rate, average_trade
  → Compute win/loss counts from win_rate × total_trades
  → Derive avg_win and avg_loss that reconcile to net_pnl
  → Emit per-trade PnL slice for resampling
```

### Resampling modes

| Mode | Method | Purpose |
|------|--------|---------|
| Bootstrap | Sample with replacement (`BootstrapSample`) | Estimate return distribution under trade repetition |
| Shuffle | Randomize order without replacement (`ShuffleOrder`) | Test path dependence from trade sequencing |

Simulations alternate between bootstrap and shuffle paths for balanced coverage.

## 50. Bootstrap Simulation

Bootstrap simulation resamples the trade return population with replacement to produce `N` independent equity paths.

```text
For each simulation i in 1..N:
    path ← BootstrapSample(trades, len(trades), rng)
    total_return[i] ← sum(path)
    max_drawdown[i] ← MaxDrawdown(path)
```

Configuration:

```yaml
montecarlo:
  enabled: true
  simulations: 1000
  confidence_level: 0.95
  random_seed: 42
```

When `random_seed` is set, all paths are reproducible across runs.

## 51. Confidence Intervals

Confidence intervals are computed from the sorted simulated return distribution using percentile bounds.

```text
tail ← (1 - confidence_level) / 2
lower ← percentile(returns, tail)
upper ← percentile(returns, 1 - tail)
mean  ← arithmetic mean of returns
```

Default confidence level is `0.95` (95% interval). The interval is included in the `montecarlo.completed` payload as `confidence_interval`.

## 52. Distribution Metrics

`distribution_summary` on `montecarlo.completed` aggregates cross-simulation statistics:

| Field | Description |
|-------|-------------|
| `mean_return` | Mean simulated total return |
| `median_return` | Median simulated total return |
| `std_dev_return` | Standard deviation of returns |
| `mean_max_drawdown` | Average max drawdown across paths |
| `median_max_drawdown` | Median max drawdown across paths |
| `worst_drawdown` | Largest drawdown observed |
| `best_drawdown` | Smallest drawdown observed |

These metrics quantify the spread of outcomes without requiring additional backtest runs.

## 53. Risk-of-Ruin

Risk-of-ruin measures the fraction of simulation paths where max drawdown exceeds a configurable ruin threshold relative to starting capital.

```text
starting_capital ← sum(|trade_pnl|) for extracted trades
ruin_threshold   ← starting_capital × ruin_drawdown_pct
risk_of_ruin     ← count(max_drawdown >= ruin_threshold) / simulations
```

Default `ruin_drawdown_pct` is `1.0` (100% of estimated starting capital). Result is always in `[0, 1]`.

## 54. Future Reporting Integration

Report export (Phase 6) will consume `montecarlo.completed` alongside `walkforward.completed` and `optimization.updated`:

| Extension | Integration point |
|-----------|-------------------|
| CSV/JSON export | `montecarlo.completed` distribution summaries |
| Dashboard charts | Return histogram and confidence interval bands |
| Strategy selection | Combine walk-forward stability + Monte Carlo robustness scores |
| Persistence | PostgreSQL archival of simulation batches |

The Monte Carlo Engine provides distribution metrics without architectural changes to upstream engines.

## 55. Phase 5 — Monte Carlo Engine

### Responsibility

- Subscribe to `walkforward.completed` events.
- Extract trade returns from validation `performance_metrics`.
- Run bootstrap and shuffle Monte Carlo simulations.
- Compute statistical summaries and confidence intervals.
- Publish `montecarlo.completed` per walk-forward window.

### Configuration

```yaml
montecarlo:
  enabled: true
  simulations: 1000
  confidence_level: 0.95
  random_seed: 42
```

### Lifecycle

1. Monte Carlo engine starts **before** walk-forward engine (subscribes first).
2. On `walkforward.completed`, enqueue simulation in consumer goroutine.
3. Store result in thread-safe in-memory cache.
4. Publish `montecarlo.completed`.
5. On shutdown: cancel context, drain subscription, wait on WaitGroup.

### Event payload (`montecarlo.completed`)

| Field | Type | Description |
|-------|------|-------------|
| `simulation_id` | string | Monte Carlo batch identifier |
| `walkforward_id` | string | Source walk-forward batch |
| `experiment_id` | string | Source experiment batch |
| `simulations` | int | Number of paths executed |
| `confidence_interval` | object | `{ level, lower, upper, mean, median }` |
| `probability_of_profit` | float64 | Fraction of profitable paths |
| `probability_of_loss` | float64 | Fraction of losing paths |
| `risk_of_ruin` | float64 | Fraction of paths exceeding ruin drawdown |
| `distribution_summary` | object | Return and drawdown aggregates |
| `timestamp` | time | Completion timestamp |

### Health

`GET /health/components` includes `montecarlo_engine` with `simulations_started`, `simulations_completed`, `reports_generated`, `active_jobs`, `average_runtime_ms`.

### Package layout

```text
internal/montecarlo/
├── engine.go       # Lifecycle, bus subscription, orchestration
├── config.go       # Simulation count, confidence level, seed
├── simulator.go    # Path generation and summarization
├── bootstrap.go    # Trade resampling and shuffle
├── statistics.go   # Confidence intervals, drawdown, risk-of-ruin
├── cache.go        # Thread-safe simulation state
├── events.go       # Input/output event payloads
└── health.go       # Health reporter
```

### Testing

| Test | Validates |
|------|-----------|
| Bootstrap resampling | Correct sample count |
| Confidence interval | Lower < Mean < Upper |
| Risk of ruin | Valid probability in [0, 1] |
| `montecarlo.completed` | Event emitted with required payload fields |

## 56. Updated Stage 4 Roadmap

| Phase | Name | Status | Consumes | Produces |
|-------|------|--------|----------|----------|
| 1 | Backtest Replay | Complete | Historical data | `MarketDataReceived` |
| 2 | Strategy Optimization | Complete | `performance.updated` | `optimization.updated` |
| 3 | Experiment & Parameter Sweep | Complete | Config grid + replay | `experiment.completed` |
| 4 | Walk-Forward Analysis | Complete | Rolling replay windows | `walkforward.completed` |
| 5 | Monte Carlo Simulation | Complete | `walkforward.completed` | `montecarlo.completed` |
| 6 | Report Export | Planned | `montecarlo.completed` + upstream events | CSV/JSON reports |

## 57. Research Repository

The Research Repository (Phase 6) is the permanent storage layer for the Stage 4 Research Platform. It persists completed research artifacts from the optimization → walk-forward → Monte Carlo pipeline into PostgreSQL and generates reusable reports from stored data.

### Design principles

| Rule | Enforcement |
|------|-------------|
| PostgreSQL as source of truth | All artifacts written via `Repository` interface |
| No duplicate database | Reuses existing `postgres.Pool` and pgx connection |
| Event-only ingestion | Subscribes to `optimization.updated`, `walkforward.completed`, `montecarlo.completed` |
| Reports from DB | `ReportGenerator` loads `ResearchBundle` from PostgreSQL, not in-memory caches |
| Lightweight runtime cache | `Cache` tracks active report generation only |
| Append-only events | Publishes `research.updated` without mutating upstream payloads |

### Package ownership

| Package | Role |
|---------|------|
| `internal/research` | Repository, report generation, export, orchestration |
| `internal/infrastructure/config/research.go` | Configuration mapping |
| `deployments/postgres/002_research.sql` | Deployment migration DDL |

## 58. Persistent Research Storage

Research artifacts are stored in PostgreSQL immediately upon event receipt. In-memory engine caches from upstream phases (optimization, walk-forward, Monte Carlo) are not used as the persistence layer.

### Storage flow

```text
optimization.updated     → UpsertExperiment + InsertOptimizationResult
walkforward.completed    → UpsertExperiment + InsertWalkForwardResult
montecarlo.completed     → EnsureExperiment + InsertMonteCarloResult → Generate report
```

Schema is applied idempotently via `EnsureSchema()` on engine start using embedded `schema.sql`.

## 59. PostgreSQL Data Model

| Table | Purpose | Key columns |
|-------|---------|-------------|
| `research_experiments` | Experiment metadata | `experiment_id`, `strategy`, `symbol`, `timeframe`, `parameters` (JSONB) |
| `optimization_results` | Optimization evaluations | `experiment_id`, `score`, `win_rate`, `expectancy`, `profit_factor`, `drawdown`, `metrics` (JSONB) |
| `walkforward_results` | Walk-forward windows | `walkforward_id`, `experiment_id`, `train_score`, `validation_score`, `parameter_set` (JSONB) |
| `montecarlo_results` | Monte Carlo batches | `simulation_id`, `confidence_interval` (JSONB), `probability_of_profit`, `risk_of_ruin`, `distribution` (JSONB) |
| `research_reports` | Export tracking | `research_id`, `experiment_id`, `version`, `json_path`, `csv_path` |

Indexes on `strategy`, `symbol`, `timeframe`, and `experiment_id` support query filters.

## 60. Repository Layer

```text
Repository (interface)
    ├── EnsureSchema
    ├── EnsureExperiment / UpsertExperiment
    ├── InsertOptimizationResult
    ├── InsertWalkForwardResult
    ├── InsertMonteCarloResult
    ├── GetExperiment / ListExperiments
    ├── GetResearchBundle
    ├── InsertResearchReport
    └── CountEntries

PostgresRepository (implementation)
    └── uses pgxpool.Pool via postgres.Pool.Underlying()
```

The repository interface enables future storage backends without changing engine or export logic.

## 61. Result Versioning

Each report generation increments `version` per `experiment_id`:

```text
version ← LatestReportVersion(experiment_id) + 1
filename ← {experiment_id}_v{version}.{json|csv}
```

Prior report rows remain in `research_reports` for audit and comparison. Version is included in `UnifiedReport` and export filenames.

## 62. Experiment Metadata

`research_experiments` stores correlation metadata extracted from event payloads:

- **From `optimization.updated`**: `strategy`, `symbol`, `timeframe`, `parameters`, `experiment_id` from serialized parameters
- **From `walkforward.completed`**: `experiment_id`, `best_parameters` as JSONB
- **From `montecarlo.completed`**: `EnsureExperiment` creates placeholder row if missing (no overwrite of existing metadata)

Queries support filtering by `experiment_id`, `strategy`, `symbol`, and `timeframe`.

## 63. Report Generation

`ReportGenerator.Generate()` loads a `ResearchBundle` from PostgreSQL and builds a `UnifiedReport` with:

- All optimization, walk-forward, and Monte Carlo rows for the experiment
- `ReportSummary` with best score, latest validation, probability of profit, risk of ruin

Report generation is triggered on `montecarlo.completed` (end of research pipeline).

## 64. Export Pipeline

```text
UnifiedReport (from PostgreSQL)
        ↓
Exporter interface
    ├── JSONExporter → {experiment_id}_v{version}.json
    └── CSVExporter  → {experiment_id}_v{version}.csv
        ↓
research_reports row (paths + version)
        ↓
research.updated event
```

Exporter interfaces are format-specific; HTML/PDF exporters can be added without changing repository or engine logic.

### Configuration

```yaml
research:
  enabled: true
  export_directory: ./reports
  formats:
    - json
    - csv
```

## 65. Dashboard Integration

`research.updated` provides dashboard-ready payloads:

| Field | Use |
|-------|-----|
| `research_id` | Report identifier |
| `experiment_id` | Experiment correlation |
| `strategy` | Strategy filter/display |
| `metrics` | Summary headline cards (best score, validation, profit probability, risk of ruin) |
| `report_location` | Links to JSON/CSV files for download or API proxy |

Future HTTP endpoints can query `ListExperiments` and `GetResearchBundle` without accessing upstream engine caches.

## 66. Future ML Integration

Machine learning extensions (Phase 7+) will consume persisted research data:

| Extension | Integration point |
|-----------|-------------------|
| Feature extraction | `optimization_results.metrics` and `walkforward_results.performance_metrics` JSONB |
| Training datasets | `GetResearchBundle` cross-experiment queries |
| Model scoring | New event type (append-only) consuming `research.updated` |
| Hyperparameter feedback | `research_experiments.parameters` JSONB as feature vectors |

PostgreSQL JSONB columns and versioned reports provide a stable foundation without architectural changes to upstream engines.

## 67. Phase 6 — Research Repository & Reporting Engine

### Responsibility

- Subscribe to `optimization.updated`, `walkforward.completed`, `montecarlo.completed`.
- Persist all research artifacts to PostgreSQL.
- Generate unified JSON and CSV reports from stored data.
- Publish `research.updated` with report locations and summary metrics.

### Pipeline

```text
optimization.updated ──┐
walkforward.completed ─┼→ Research Engine → PostgreSQL → Report Generator → research.updated
montecarlo.completed ──┘
```

### Lifecycle

1. Research engine starts **before** Monte Carlo engine (subscribes first).
2. On `Start`, apply schema via `EnsureSchema()`.
3. On each input event, persist artifact to PostgreSQL.
4. On `montecarlo.completed`, generate report from DB, export files, publish `research.updated`.
5. On shutdown: cancel context, drain subscription, wait on WaitGroup.

### Event payload (`research.updated`)

| Field | Type | Description |
|-------|------|-------------|
| `research_id` | string | Generated report identifier |
| `experiment_id` | string | Source experiment batch |
| `strategy` | string | Strategy name |
| `metrics` | object | `ReportSummary` headline metrics |
| `report_location` | object | `{ json_path, csv_path }` |
| `timestamp` | time | Completion timestamp |

### Health

`GET /health/components` includes `research_engine` with `repository_entries`, `reports_generated`, `exports_completed`, `export_failures`, `postgres_writes`, `postgres_read_latency_ms`.

### Package layout

```text
internal/research/
├── engine.go       # Lifecycle, event consumption, orchestration
├── repository.go   # Repository interface
├── postgres.go     # PostgreSQL implementation
├── models.go       # Domain models and bundle types
├── reports.go      # ReportGenerator
├── exporter.go     # JSON/CSV exporters
├── config.go       # Engine configuration
├── cache.go        # Active report generation tracking
├── events.go       # research.updated payload
├── health.go       # Health reporter
├── errors.go       # Structured errors
└── schema.sql      # Embedded DDL
```

### Testing

| Test | Validates |
|------|-----------|
| Repository insert | Row stored in PostgreSQL |
| Repository read | Persisted bundle returned |
| JSON export | Valid JSON report file |
| CSV export | Correct CSV rows |
| `research.updated` | Event published with report locations |

## 68. Updated Stage 4 Roadmap

| Phase | Name | Status | Consumes | Produces |
|-------|------|--------|----------|----------|
| 1 | Backtest Replay | Complete | Historical data | `MarketDataReceived` |
| 2 | Strategy Optimization | Complete | `performance.updated` | `optimization.updated` |
| 3 | Experiment & Parameter Sweep | Complete | Config grid + replay | `experiment.completed` |
| 4 | Walk-Forward Analysis | Complete | Rolling replay windows | `walkforward.completed` |
| 5 | Monte Carlo Simulation | Complete | `walkforward.completed` | `montecarlo.completed` |
| 6 | Research Repository & Reporting | Complete | Pipeline events | `research.updated` + PostgreSQL + exports |

