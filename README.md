# DiGo

Small dependency injection for Go 1.27+: singleton, request, and transient scopes with lifecycle hooks.

[![Go Tests](https://github.com/centraunit/digo/actions/workflows/tests.yml/badge.svg)](https://github.com/centraunit/digo/actions/workflows/tests.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

```bash
go get github.com/centraunit/digo@v1.0.3
```

## Performance

Measured with `go test ./services_test/ -bench=. -benchmem` on linux/amd64 (Intel i5-8600). Re-run locally; numbers move with hardware and Go version.

| Operation | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| Transient bind | 877 | 680 | 12 |
| Request bind | 696 | 680 | 12 |
| Singleton bind | 1064 | 952 | 14 |
| Transient resolve | 1664 | 1340 | 13 |
| Request resolve (cached) | 113 | 0 | 0 |
| Singleton resolve (warm) | 46 | 0 | 0 |
| Deep dependency chain | 4264 | 3456 | 37 |
| Concurrent resolve | 19734 | 6970 | 70 |

## Usage

Singleton pool → request session that depends on it → two transient tokens per hit. Hit `/` twice: `pool` stays the same, `session` and `token_*` change.

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/centraunit/digo"
)

var seq atomic.Uint64

func nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, seq.Add(1))
}

type Pool interface {
	digo.Lifecycle
	ID() string
}

type pool struct {
	digo.NoopLifecycle
	id string
}

func (p *pool) ID() string { return p.id }

type Session interface {
	digo.Lifecycle
	ID() string
	PoolID() string
}

type session struct {
	digo.NoopLifecycle
	id   string
	pool Pool
}

func (s *session) ID() string     { return s.id }
func (s *session) PoolID() string { return s.pool.ID() }

type Token interface {
	digo.Lifecycle
	ID() string
}

type token struct {
	digo.NoopLifecycle
	id string
}

func (t *token) ID() string { return t.id }

func main() {
	_ = digo.ProvideSingleton[Pool](func(ctx *digo.ContainerContext) (Pool, error) {
		return &pool{id: nextID("pool")}, nil
	})
	_ = digo.ProvideRequest[Session](func(ctx *digo.ContainerContext) (Session, error) {
		p, err := digo.ResolveSingleton[Pool]()
		if err != nil {
			return nil, err
		}
		return &session{id: nextID("session"), pool: p}, nil
	})
	_ = digo.ProvideTransient[Token](func(ctx *digo.ContainerContext) (Token, error) {
		return &token{id: nextID("token")}, nil
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

		pool, err := digo.ResolveSingleton[Pool]()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		sess, err := digo.ResolveRequest[Session](ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Same request_id → same Session instance.
		again, err := digo.ResolveRequest[Session](ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		t1, err := digo.ResolveTransient[Token](ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		t2, err := digo.ResolveTransient[Token](ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		fmt.Fprintf(w, "pool=%s\n", pool.ID())
		fmt.Fprintf(w, "session=%s pool_via_session=%s same_session=%v\n",
			sess.ID(), sess.PoolID(), sess == again)
		fmt.Fprintf(w, "token_a=%s token_b=%s\n", t1.ID(), t2.ID())
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Example output (two hits; `pool-1` unchanged):

```
--- hit 1 ---
pool=pool-1
session=session-2 pool_via_session=pool-1 same_session=true
token_a=token-3 token_b=token-4
--- hit 2 ---
pool=pool-1
session=session-5 pool_via_session=pool-1 same_session=true
token_a=token-6 token_b=token-7
```

### Scopes

- **Singleton** — one instance per container (`BindSingleton` / `ProvideSingleton`, `Boot`, `ResolveSingleton`)
- **Request** — one instance per `request_id` (`ProvideRequest`, `ResolveRequest(ctx)`, cleanup via `RequestScope`’s `end`)
- **Transient** — new instance every resolve (`ProvideTransient`, `ResolveTransient(ctx)`)

Use `NewContainer()` when you need an isolated container; package-level helpers use a process default.

### Lifecycle

`Boot()` starts singletons. `Shutdown(false)` tears down request/transient inventory. `Shutdown(true)` / `Reset()` clear everything. Hooks run outside the container lock so nested `Resolve*` is safe if you pass the boot `ctx`.

## v1.0 notes

Requires Go 1.27. Request scope is keyed by request id; transient means a new instance; resolve APIs that need request/transient take `context.Context`. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
