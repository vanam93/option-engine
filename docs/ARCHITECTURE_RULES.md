# Option Engine - Architecture Rules

These rules apply to every implementation phase.

Ask me if you have any doubts before implmenting

## General

- Follow Clean Architecture.
- Follow SOLID principles.
- Keep packages cohesive and loosely coupled.
- Avoid cyclic dependencies.
- Keep APIs minimal and stable.

## Stage Freeze

Completed stages are frozen.

Do not modify previous stages unless:

- critical bug
- security issue
- explicit approval

Never refactor frozen stages to implement new features.

## Ownership

Every runtime component must have one owner.

Examples:

- Provider lifecycle → ProviderManager
- Desired subscriptions → SubscriptionManager
- Runtime event consumption → Gateway
- Mutable market state → Cache
- Immutable read models → Snapshot
- Analytics state → Analytics Engine

No shared ownership.

## Event Flow

All downstream engines consume events.

Never bypass the event pipeline.

Provider
→ Gateway
→ EventBus
→ Analytics
→ Strategy
→ Decision
→ Execution

## Contracts

Public interfaces are append-only.

Avoid breaking changes.

Prefer adding functionality over modifying contracts.

## Concurrency

No goroutine leaks.

Every goroutine must terminate.

Every background worker must:

- accept Context
- support graceful shutdown
- use WaitGroup when required

Avoid unnecessary locking.

## State

One mutable source of truth.

Expose immutable read models.

Never expose internal mutable maps or slices.

## Configuration

No hardcoded runtime values.

Use configuration for:

- buffers
- retries
- intervals
- timeouts
- limits

## Errors

Return structured errors.

Do not panic for runtime failures.

## Logging

Structured logging only.

No debug prints.

## Performance

Prefer incremental algorithms.

Avoid full recomputation.

Avoid unnecessary allocations.

Avoid O(n²) paths.

## Testing

Run:

go build ./...

Run existing tests.

Only add tests for:

- algorithms
- concurrency
- recovery logic

Skip trivial unit tests.

## Completion

Before finishing:

- build passes
- existing tests pass
- summarize changes
- stop

Do not implement future phases.