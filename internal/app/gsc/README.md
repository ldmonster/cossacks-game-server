# internal/app/gsc — GSC application

Inbound GSC dispatcher: takes a parsed `transport/gsc.Stream` plus a
`transport/tconn.Connection` and produces zero or more response
commands.

## Files

- `controller.go` (was `dispatcher.go`) — the `Controller` type. Holds
  references to adapter stores, sub-services (identity, lobby, match,
  ranking, connectivity), the renderer, and the command/route
  registries. `NewController` is the GSC composition entry point.
- `registry.go` — table-driven `CommandHandler` registry. Adding a new
  command means adding a struct to `commands/` and registering it in
  `defaultCommandRegistry`. The dispatcher core does not change.
- `dispatch_routes.go` — open/go route table. Same pattern as the
  command registry: routes live in `routes/` and adapters in this
  file present a uniform signature to the dispatch table.
- `controller_*.go` — small per-feature handlers (alive timer
  refresh, stats lock, leave-room hook). They are thin wrappers that
  delegate to commands and routes; they exist so the routes/commands
  packages can satisfy a few cross-cutting Controller-level ports.
- `errors.go`, `unimplemented.go` — typed dispatch errors and the
  list of commands/routes that are accepted but intentionally
  produce no response.

## Sub-packages

- [`commands/`](commands/README.md) — one struct per top-level GSC
  command (`login`, `gettbl`, `start`, `endgame`, etc.).
- [`routes/`](routes/README.md) — one struct per `open` / `go`
  route (`enter`, `try_enter`, `startup`, `reg_new_room`, …).

## Tests

The package owns the wire fidelity suite. Tests construct
`Controller` directly, drive `dispatchOpen`, `HandleWithMeta`, or the
sub-handlers, and compare the resulting bytes to `testdata/golden`
(JSON descriptors) or `testdata/template_fullbody` (raw template
output). Any byte change in client-visible output requires updating
the corresponding golden file with a documented justification.
