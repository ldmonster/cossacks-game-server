# internal/transport — protocol adapters

Inbound transport. The TCP server accepts connections, the wire codec
parses frames into `gsc.Stream` values, and per-connection runtime
context lives in `tconn`.

| Package | Role |
|---------|------|
| `gsc/`   | Wire codec — `Stream`, `CommandSet`, `Command`, plus binary marshal/read helpers. No upward dependencies. |
| `tcp/`   | `Server` — accepts TCP connections, reads frames, dispatches each command via a `port.RequestHandler`. |
| `tconn/` | `Connection` — per-connection runtime context (id, IP, ctx/cancel, session). Imported by both `tcp` and the GSC application layer. |

## Why `tconn` lives separately from `tcp`

The `Connection` struct is consumed by the application layer
(`internal/app/gsc/...`) and by the TCP server. Putting it in the
`tcp` package would create an import cycle with `internal/port`
(which references `*Connection` through `port.RequestHandler`).
Splitting it into a sibling package keeps the cycle broken without
introducing an interface boundary that the codebase does not yet
need.
