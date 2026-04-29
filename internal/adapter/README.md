# internal/adapter — outbound infrastructure adapters

Concrete implementations of application ports. Each adapter is a leaf
package: it depends only on stdlib, on `internal/domain` types it
needs to construct or return, and on `internal/port` for the
interfaces it satisfies. Adapters never depend on `internal/app` or
`internal/transport`.

| Package | Implements | Notes |
|---------|------------|-------|
| `accountprovider/` | `port.AuthProvider` | HTTP-backed identity provider. |
| `rankingfile/`     | ranking source     | Reads CossacksLeague / CossacksCup score files. |
| `kvmemory/`        | `port.KVStore`     | Process-local in-memory KV store. |
| `stun/`            | UDP STUN responder | Used by the GSC startup payload to discover the client's external (ip, port). |
| `rooms/`           | room aggregate store | In-memory `*Store` providing the room/player repositories with per-room locks. |

## Picking an adapter

Composition happens in `cmd/cossacksd/main.go`. Tests for application
services should always use in-memory test doubles; tests that exercise
the full GSC pipeline (wire tests in `internal/app/gsc/`) construct
the adapters directly so wire output matches the production wiring.
