# DiGo

Small dependency injection for Go 1.27+: singleton, request, and transient scopes with lifecycle hooks.

[![Go Tests](https://github.com/centraunit/digo/actions/workflows/tests.yml/badge.svg)](https://github.com/centraunit/digo/actions/workflows/tests.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

```bash
go get github.com/centraunit/digo@v1.0.0
```

## Performance

Measured with `go test ./services_test/ -bench=. -benchmem` on linux/amd64 (Intel i5-8600). Re-run locally; numbers move with hardware and Go version.

| Operation | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| Transient bind | 1013 | 704 | 13 |
| Request bind | 1014 | 704 | 13 |
| Singleton bind | 1309 | 976 | 15 |
| Transient resolve | 2071 | 1363 | 14 |
| Request resolve (cached) | 123 | 24 | 1 |
| Singleton resolve | 827 | 1016 | 12 |
| Deep dependency chain | 4047 | 3563 | 40 |
| Concurrent resolve | 22972 | 6687 | 75 |

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/centraunit/digo"
)

type DB interface {
	digo.Lifecycle
	Ping() error
}

type postgres struct{ digo.NoopLifecycle }

func (p *postgres) Ping() error { return nil }

func main() {
	_ = digo.ProvideSingleton[DB](func(ctx *digo.ContainerContext) (DB, error) {
		return &postgres{}, nil
	})
	_ = digo.ProvideRequest[DB](func(ctx *digo.ContainerContext) (DB, error) {
		return &postgres{}, nil
	})
	if err := digo.Boot(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		parent := r.Context()
		if id := r.Header.Get("X-Request-ID"); id != "" {
			parent = digo.WithRequestID(parent, id)
		}
		ctx, end, err := digo.RequestScope(parent)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer end()

		db, err := digo.ResolveRequest[DB](ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprintln(w, db.Ping())
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Scopes

- **Singleton** — one instance per container (`BindSingleton` / `ProvideSingleton`, `Boot`, `ResolveSingleton`)
- **Request** — one instance per `request_id` (`ProvideRequest`, `ResolveRequest(ctx)`, cleanup via `RequestScope`’s `end`)
- **Transient** — new instance every resolve (`ProvideTransient`, `ResolveTransient(ctx)`)

Use `NewContainer()` when you need an isolated container; package-level helpers use a process default.

### Lifecycle

`Boot()` starts singletons. `Shutdown(false)` tears down request/transient inventory. `Shutdown(true)` / `Reset()` clear everything. Hooks run outside the container lock so nested `Resolve*` is safe if you pass the boot `ctx`.

## v1.0.0 notes

Requires Go 1.27. Request scope is keyed by request id; transient means a new instance; resolve APIs that need request/transient take `context.Context`. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
