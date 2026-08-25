# Changelog

## v1.0.2

- Fix README example: drop unused `context` import; print `ok` instead of `Ping()`'s nil error

## v1.0.1

- Warm singleton resolve skips cycle-stack / context merge (~46 ns/op, 0 allocs)
- `RWMutex` for provider lookups; cached type / provider keys

## v1.0.0

Requires Go 1.27.

- Real request isolation by `request_id` and `RequestScope`
- Transient/request via factories (`Provide*`); new instance per transient resolve
- `ResolveRequest` / `ResolveTransient` take `context.Context`
- `NewContainer` plus package-level helpers on a default container
- Lifecycle hooks never run under the container mutex
- Circular dependency detection via resolve stack in context
- `NoopLifecycle` for types with empty hooks

Breaking relative to earlier commits: predicates and shared-instance “transient”/global request slot are gone.
