# internal/app — application layer

Application services orchestrate domain objects to fulfil use cases.
Each subdirectory is a single bounded context; packages depend only on
`internal/domain`, `internal/port`, and other application packages.

| Package | Responsibility |
|---------|----------------|
| `identity` | Authentication and account post-actions (enter flow, nick rules). |
| `lobby`    | Room lifecycle: create, join, leave, start, info. |
| `match`    | In-game state: per-player stats, endgame events, start payload. |
| `ranking`  | CossacksLeague and CossacksCup score readers with caching. |
| `connectivity` | Alive-timer driven session registry keyed by player id. |
| `gsc`      | Inbound GSC command/route dispatcher (the application surface that the TCP transport drives). |

## Conventions

- Each package exposes a `Service` (or comparable) struct constructed
  with explicit dependencies. No global state.
- Consumer ports — narrow interfaces describing what a service needs
  from outside — live next to the service that consumes them.
- Adapters never appear in application signatures; tests substitute
  in-memory implementations of the ports.

## Wire fidelity

The GSC application is bound by a strict byte-exact contract with the
historical reference server. Wire-level fixtures in
`internal/app/gsc/testdata/golden` and `testdata/template_fullbody`
guard that contract; any change to wording, ordering, or numeric
formatting that affects the bytes a client receives is a regression.
