# Changelog

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
