Stage 2 is ARCHITECTURALLY FROZEN.

Do not modify:
- Provider
- ProviderManager
- EventSource
- Gateway
- Normalizer
- Validator
- Cache
- EventBus
- Snapshot
- SubscriptionManager

unless a critical bug is found.

Stage 3 must consume ONLY canonical events from EventBus.

All new components must be provider-independent.

Always compile before finishing a phase.

Do not change public APIs without explicit approval.

Ask me if you have any doubts before proceeding with implementation
