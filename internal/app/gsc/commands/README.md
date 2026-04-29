# internal/app/gsc/commands — GSC commands

One file per top-level GSC command. Each command is a small struct
implementing `gsc.CommandHandler`:

```go
type Login struct{ /* deps */ }

func (Login) Name() string { return "login" }

func (l Login) Handle(
    ctx context.Context,
    conn *tconn.Connection,
    req *gsc.Stream,
    args []string,
) port.HandleResult { ... }
```

## Adding a new command

1. Add a new file `internal/app/gsc/commands/<name>.go` containing a
   struct, `Name()` method returning the wire name, and `Handle`.
2. Define any narrow consumer ports the handler needs in the same
   file (an interface that the wiring layer satisfies).
3. Register the struct in `defaultCommandRegistry` in
   `internal/app/gsc/controller.go`. Add a corresponding field to
   `commandDeps` for the dependencies the constructor needs.

The dispatcher core (`Registry.Lookup`) is unchanged. The command
itself owns its concurrency and validation; there is no shared
controller-level lock.

## Existing commands

| File | GSC name | Notes |
|------|----------|-------|
| `simple.go` | `login`, `echo`, `url` | Trivial handlers. |
| `proxy.go`  | `proxy` | Rewrites the connection's apparent (ip, port) after key check. |
| `leave.go`  | `leave` | Leaves the active room and stops the alive timer. |
| `alive.go`  | `alive` | Refreshes the alive timer for the current session. |
| `endgame.go`| `endgame` | Parses an endgame line and updates per-player stats. |
| `gettbl.go` | `gettbl` | Returns the rooms table for the current protocol version. |
| `start.go`  | `start` | Marks a host's room as started; posts an account action. |
| `stats.go`  | `stats` | Per-player in-game stat update; locks the room. |
| `route_open.go` | `open`, `go` | Dispatches to the route registry in `internal/app/gsc/routes`. |
