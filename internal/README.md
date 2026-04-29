# internal — package layout

Hexagonal / Ports-and-Adapters layout, organised by DDD bounded
contexts. Imports flow inward only; depguard rules in
`.golangci.yaml` enforce the boundaries.

```
internal/
    domain/      pure types per bounded context
        identity/  Account, AccountInfo, Nickname
        lobby/     Room, RoomID, RoomTitle, ControlSum
        match/     PlayerStat, EndgameEvent, StartedUsers, StartPayload
        player/    Player, PlayerID, ConnectionID
        session/   Session

    app/         use-case services, orchestrate domain via ports
        identity/    enter flow, post-account-action
        lobby/       room lifecycle: create/join/leave/start/info
        match/       in-game stats, endgame
        ranking/     CossacksLeague / CossacksCup readers + cache
        connectivity/  alive timers + by-id session registry
        gsc/         GSC application (controller, registry, dispatcher)
            commands/  one struct per top-level GSC command
            routes/    one struct per open/go route

    port/        small role-based interfaces shared across the app
                 layer (KVStore, RankingService, RoomRepository, ...)

    transport/   protocol adapters
        gsc/     wire codec (Stream/CommandSet/Command + I/O)
        tcp/     TCP server + per-connection read loop
        tconn/   per-connection runtime context (Connection struct)

    adapter/     outbound infrastructure adapters
        accountprovider/  HTTP-backed identity provider
        rankingfile/      file-backed ranking source
        kvmemory/         in-memory KV store
        stun/             STUN UDP responder
        rooms/            in-memory room/player aggregate store

    platform/    cross-cutting platform services
        config/  YAML loader
        logging/ zap setup
        metrics/ Prometheus storage + HTTP
        health/  /livez /readyz

    render/      LW_show template renderer (template loader, vars
                 builders, time-interval helpers)
```

## Dependency rules (enforced by depguard)

- `domain/*` — stdlib + sibling domain packages only.
- `app/*` — may depend on `domain`, `port`, `transport/gsc`,
  `transport/tconn`, `render`, `platform`. May not depend on
  `adapter` (use ports) or on `transport/tcp` (the inbound server).
  The GSC application's wire tests are exempted from the adapter
  ban so they can construct concrete stores directly.
- `adapter/*` — may not depend on `transport/*`.
- `transport/tcp` and `transport/tconn` — may depend on `port` and
  `transport/gsc`.

## Composition root

`cmd/cossacksd/main.go` wires everything: it constructs each adapter,
the GSC dispatcher (`gsc.NewController`), and the TCP server that
hands accepted connections to the dispatcher.
