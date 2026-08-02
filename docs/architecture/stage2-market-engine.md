# Stage 2 Market Engine Architecture

## 1. High-level architecture

```mermaid
flowchart LR
    A[Application] --> B[Container]
    B --> C[ProviderManager]
    B --> D[SubscriptionManager]
    B --> E[Gateway]
    B --> F[Normalizer]
    B --> G[Validator]
    B --> H[Cache]
    B --> I[EventBus]
    B --> J[Snapshot]

    C --> K[Provider]
    D --> K
    E --> F
    E --> G
    E --> H
    E --> I
    H --> J

    K -->|events| E
```

## 2. Runtime event flow

1. The application starts the container and connects the active provider.
2. The container subscribes the desired symbols through the subscription manager.
3. The gateway consumes provider events and routes them through normalization and validation.
4. Valid ticks update the cache, fan out to the event bus, and are reflected in snapshots.
5. On reconnect or provider switch, the subscription manager replays the desired subscriptions to the new provider instance.

## 3. Component responsibilities

- ProviderManager
  - Owns provider lifecycle and provider switching.
  - Connects, disconnects, and replays subscriptions after reconnect/switch.
  - Owns the active provider session and exposes it to the gateway as an event source.
  - Does not own desired subscription state.

- SubscriptionManager
  - Owns the desired symbol set.
  - Stores subscriptions independently of the active provider connection.
  - Replays desired subscriptions after reconnect or provider switch.

- Gateway
  - Is the only runtime consumer of provider events.
  - Depends on the provider event-source abstraction rather than a concrete provider instance.
  - Converts provider payloads into canonical market ticks.
  - Runs normalization, validation, cache updates, and event fan-out.

- Normalizer
  - Converts provider-specific payloads into canonical market ticks.
  - Is a pure transformation step for runtime data.

- Validator
  - Rejects malformed, stale, duplicate, or out-of-order ticks.
  - Protects the cache and downstream consumers from invalid state.

- Cache
  - Stores the current mutable market state.
  - Holds the latest ticks, chains, and depth snapshots.

- EventBus
  - Fan-outs canonical runtime events to subscribers.
  - Provides a non-blocking bounded distribution mechanism.

- Snapshot
  - Produces immutable read-model views from the cache.
  - Prevents downstream consumers from mutating the live runtime state.

## 4. Lifecycle

### Startup

```mermaid
sequenceDiagram
    participant App
    participant Container
    participant ProviderManager
    participant SubscriptionManager
    participant Gateway

    App->>Container: StartRuntime(ctx)
    Container->>ProviderManager: Connect(ctx)
    ProviderManager->>ProviderManager: connect provider
    ProviderManager->>SubscriptionManager: Recover(ctx, provider)
    Container->>SubscriptionManager: Subscribe(ctx, symbols)
    Container->>Gateway: Start(ctx)
```

### Tick processing

```mermaid
sequenceDiagram
    participant Provider
    participant Gateway
    participant Normalizer
    participant Validator
    participant Cache
    participant EventBus

    Provider->>Gateway: MarketDataReceived
    Gateway->>Normalizer: Tick(payload)
    Normalizer-->>Gateway: canonical tick
    Gateway->>Validator: Validate(tick)
    Validator-->>Gateway: accept/reject
    Gateway->>Cache: PutTick(tick)
    Gateway->>EventBus: Publish(event)
```

### Shutdown

```mermaid
sequenceDiagram
    participant App
    participant Gateway
    participant ProviderManager
    participant Provider

    App->>Gateway: Close()
    Gateway->>Gateway: cancel context
    Gateway->>Gateway: wait for goroutine exit
    Gateway->>ProviderManager: disconnect
    ProviderManager->>Provider: Disconnect(ctx)
    Provider-->>ProviderManager: stopped
```

### Reconnect

```mermaid
sequenceDiagram
    participant ProviderManager
    participant SubscriptionManager
    participant Provider

    ProviderManager->>Provider: Disconnect(ctx)
    ProviderManager->>Provider: Connect(ctx)
    ProviderManager->>SubscriptionManager: Recover(ctx, provider)
    SubscriptionManager->>Provider: Subscribe(ctx, desired symbols)
```

## 5. Ownership rules

- Provider ownership
  - The active provider instance is owned by ProviderManager.
  - The gateway only consumes events from the runtime event-source abstraction.
  - Provider lifecycle, reconnect, and switching are not exposed to the gateway.

- Cache ownership
  - The cache is the single mutable market-state store for runtime ticks and related snapshots.
  - No other component mutates runtime tick state directly.

- Snapshot ownership
  - Snapshots are derived from cache contents and are immutable from the caller perspective.
  - Snapshots must never be used as the source of truth for mutations.

- EventBus ownership
  - EventBus is a delivery mechanism for runtime events.
  - It should not own business state or provider subscriptions.

- SubscriptionManager ownership
  - SubscriptionManager owns the desired subscription set.
  - It is the only place where desired subscriptions are stored.

## 6. Thread-safety guarantees

- ProviderManager uses a mutex to guard the active provider reference and lifecycle state.
- SubscriptionManager uses a mutex to protect the desired symbol set.
- Cache uses a mutex to protect its maps and expose safe copies.
- EventBus uses a mutex to guard subscriptions and publish operations.
- The gateway processes provider events sequentially through its runtime loop and does not mutate shared runtime state outside the cache/event bus contracts.

## 7. Extension points for new providers

New providers should implement the provider interface in [internal/providers/api/provider.go](../providers/api/provider.go) and register themselves in the provider registry. They must support:

- Connect/Disconnect lifecycle
- Subscribe/Unsubscribe for the desired symbol set
- Event emission through the provider event channel
- Health reporting
- Capabilities reporting

The reconnect path works automatically as long as the provider can be reconnected and accepts subscriptions again.

## 8. Public interfaces

### Provider interface

- Connect(ctx)
- Disconnect(ctx)
- Subscribe(ctx, symbols)
- Unsubscribe(ctx, symbols)
- Events()
- Health()
- Capabilities()

### Runtime gateway interface

- Start(ctx)
- Close()

### Subscription manager interface

- Subscribe(ctx, symbols)
- Unsubscribe(ctx, symbols)
- Recover(ctx, provider)
- Symbols()

## 9. Configuration

Relevant runtime configuration is sourced from the container config and provider config:

- Provider selection
- Reconnect interval and retry count
- Subscription batch size
- Tick validation max age
- Logging level

The container wires those values into the provider manager, gateway, validator, and subscription manager.

## 10. Known limitations

- Reconnect subscription recovery is now handled by the subscription manager after provider reconnect and provider switch, but providers still need to implement stable connect/subscribe semantics.
- The current runtime path is focused on market data ticks; richer market features such as deeper option-chain fan-out and historical replay remain future work.
- Snapshot generation is derived from cache state and should be treated as a read model rather than a write path.

## 11. Historical Groww Provider

### Purpose

The Groww provider integrates official Groww Historical Backtesting APIs as a market data source. It behaves like the replay provider: historical candles are downloaded, converted into the existing normalizer payload, and streamed sequentially as `MarketDataReceived` events through the standard provider pipeline.

No analytics, research, recommendation, or AI code is aware that data originates from Groww.

### Architecture

```
Groww REST API
    ↓
GrowwProvider (internal/providers/groww)
    ↓
ProviderManager
    ↓
Gateway → Normalizer → Validator → Cache → EventBus
    ↓
Entire platform
```

All Groww-specific logic is confined to `internal/providers/groww/`. Future providers (Zerodha, Fyers, Angel One, Upstox, Dhan, CSV, database) should follow the same package layout and register with the provider registry.

### Authentication

Credentials are loaded from configuration only:

- `api_key` + `api_secret` (checksum-based daily token flow)
- or pre-generated `access_token`

Tokens are sent using standard Groww headers (`Authorization: Bearer`, `X-API-VERSION: 1.0`). No credentials are hardcoded.

### Configuration

Enable by switching the active market provider:

```yaml
market:
  provider: groww
  groww:
    enabled: true
    api_key: ${GROWW_API_KEY}
    api_secret: ${GROWW_API_SECRET}
    access_token: ${GROWW_ACCESS_TOKEN}
    base_url: https://api.groww.in
    requests_per_second: 5
    retry_attempts: 3
    retry_backoff_ms: 500
    replay_speed: realtime
    candle_interval: 5minute
    timeframe: 5m
    exchange: NSE
    segment: CASH
    start_time: "2026-08-01T09:15:00+05:30"
    end_time: "2026-08-27T15:30:00+05:30"
```

`replay_speed` supports `instant`, `realtime`, `1x`, `2x`, `5x`, `10x`, or a numeric multiplier.

### Replay streaming

Historical candles are fetched in API-compliant date chunks and streamed one candle at a time. The provider does not load full histories into memory. Multi-symbol subscriptions are merged in timestamp order before emission.

Each candle is mapped to the existing `normalizer.Payload` (timestamp, open, high, low, close, volume) and published as `MarketDataReceived`.

### Provider lifecycle

GrowwProvider implements the standard provider interface:

- `Connect` authenticates and starts the streaming goroutine
- `Subscribe` / `Unsubscribe` manage the desired symbol set
- `Events` exposes the provider event channel consumed by the gateway session
- `Disconnect` stops streaming and closes HTTP resources
- `Health` reports connection, authentication, request, retry, and streaming metrics
- `Capabilities` reports historical replay support (no live ticks)

### Health

Health reports include:

- `connected`
- `authenticated`
- `requests`
- `errors`
- `latency_ms`
- `candles_streamed`
- `retries`
- `last_request`
- `last_response`

### Retry

The HTTP client retries transient failures with exponential backoff for HTTP `429`, `500`, `502`, `503`, and `504`, plus network timeouts. Retry count and backoff are configurable.

### Rate limiting

Outbound requests are throttled with a token-bucket limiter (`requests_per_second`). This protects against Groww API rate limits during long historical replays.

### Future provider support

Groww is the reference implementation for external historical providers. Adding Zerodha, Fyers, or other brokers requires:

1. A new package under `internal/providers/<broker>/`
2. Implementation of the existing `api.Provider` interface
3. Registration in `providers.DefaultRegistry()`
4. A `market.<broker>` configuration block

No changes are required in gateway, analytics, research, or AI packages.
