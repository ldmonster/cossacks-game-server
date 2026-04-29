# internal/app/gsc/routes — GSC open/go routes

One file per `open` / `go` route. Routes render `LW_show` payloads
(dialogs, lists, full pages) in response to the client navigating to a
named URL.

A route exposes an exported `<Name>Impl` method on the `Routes` struct
in `routes.go`, returning `([]gsc.Command, error)`. A thin wrapper in
`flow_routes.go` adapts it to the uniform signature used by the
dispatch table.

```go
func (r *Routes) StartupImpl(
    _ context.Context,
    conn *tconn.Connection,
    req *gsc.Stream,
    p map[string]string,
) ([]gsc.Command, error) { ... }
```

## Adding a new route

1. Add a new file `internal/app/gsc/routes/<name>.go` defining the
   `Impl` method on `*Routes` plus any typed errors.
2. Define narrow consumer ports the route needs (renderer, ranking,
   players, rooms, lobby, identity, sessions). The `Deps` struct in
   `routes.go` already exposes the common ones.
3. Add a wrapper in `flow_routes.go` and register the route in
   `internal/app/gsc/dispatch_routes.go`'s `openRoutes` map.

## Conventions

- Routes never reach into adapter packages directly. They use the
  consumer ports declared in `*_port.go` files.
- Mutable config is held by pointer in `Deps` (`*config.GameConfig`,
  `*config.ServerConfig`) so test fixtures can mutate it after
  construction and have the changes propagate.
- Templates and rendering go through `Renderer.Render`. Vars are
  built by the route via the renderer's helpers; the renderer itself
  is shared and never holds per-request state.
