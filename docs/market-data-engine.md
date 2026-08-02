# Market Data Engine

The provider manager owns the active provider lifecycle and capability discovery. A provider emits `events.Event` envelopes only; its broker payload is normalized at the adapter boundary into `market.Tick`.

Event flow is `provider -> gateway.Engine -> validator -> cache -> eventbus subscribers`. The cache is exclusively a latest-state store and snapshots copy its data before downstream engines consume it. This prevents strategy, indicator, and decision code from observing mutable provider state.

On reconnect, the subscription manager retains the desired symbol set and replays it to the new connected provider in configured batches. Event bus delivery is non-blocking: a slow subscriber drops only its own excess messages and exposes the drop count. `Engine.Close` disconnects the provider, waits for its forwarding goroutine, then closes subscriber channels.

Replay providers use `ReplayClock`; `Pause`, `Resume`, `Seek`, and `SetSpeed` are safe during an active session. Mock providers use a seeded PRNG and sorted subscriptions so the generated feed is repeatable.
