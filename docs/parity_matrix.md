# Strict Compatibility Parity Matrix

This document maps the Perl implementation to Go targets under `./golang`.

## Protocol Parity

- **Frame header layout** (`GSC::Stream`): `num:uint16`, `lang:uint8`, `ver:uint8`, `size:uint32`, `len:uint32`.
- **Frame payload**: zlib-compressed command-set bytes.
- **Command-set binary layout** (`GSC::CommandSet`):
  - `count:uint16`
  - each command: `name:C/a`, `argc:uint16`, args as `L/a`.
- **String form**: `GW|cmd&arg1&arg2|next...`.
- **Arg escaping** (`GSC::Command`):
  - `\` -> `\5C`
  - `&` -> `\26`
  - `|` -> `\7C`
  - `NUL` -> `\00`

## Server Dispatch Parity

- One TCP listener on `host:port` (default `localhost:34001`).
- One GSC request -> one handler invocation.
- If multiple commands in request: handle first, log warning (Perl behavior).
- Last two args in command are interpreted as `win`, `key`; remaining are command args.
- Response attaches `win` as trailing arg for every outbound command.

## Config Parity

Source config: `golang/config/simple-cossacks-server.yaml` (Perl-compatible keys).

Required semantic parity:
- Keep config keys as-is (`host`, `port`, `hole_port`, `hole_int`, `templates`, etc.).
- Default `table_timeout = 10000` when absent.
- Preserve runtime env overrides currently done by `entrypoint-dev.sh` (`HOST_NAME`, `UDP_KEEP_ALIVE_INTERVAL`).

## Command Parity (Runtime)

From `SimpleCossacksServer::CommandController` and `...::Open`.

- **Transport/session**: `proxy`, `login`, `echo`, `alive`, `leave`, `url`, `upfile`, `unsync`.
- **Routing**: `open` and `go` map to methods in Open controller.
- **Lobby/room**: `enter`, `try_enter`, `startup`, `games`, `new_room_dgl`, `reg_new_room`, `join_game`, `join_pl_cmd`, `room_info_dgl`.
- **State sync**: `GETTBL`, `stats`, `start`, `endgame`.
- **Info/UI**: `user_details`, `users_list`, `tournaments`, `lcn_registration_dgl`, `gg_cup_thanks_dgl`.

## Side-Effect Parity

- In-memory player and room state mutation and cleanup on disconnect.
- Redis usage:
  - STUN service writes endpoint info by `player_id`.
  - room join reads host endpoint for hole punching.
- Optional HTTP backend calls for LCN/WCL account actions.
- Access/error logging conventions and key event messages.

## Phase 0: Per-command contract and fixtures (Go)

- **Full command table (GSC + open routes, args, state, no-response):** [parity_command_contract.md](parity_command_contract.md) — use this for Phase 0/2/4/6 work; keep the matrix here for wire/config overview.
- **Fixture catalog and golden test paths:** [testdata/parity_fixtures/README.md](testdata/parity_fixtures/README.md) (`golang/testdata/parity_fixtures/README.md` from repo root).

## STUN Parity

From legacy STUN Python implementation:
- UDP port `3708`.
- Packet parse:
  - tag `<4s` (expect `CSHP`)
  - version `<c` (expect `1`)
  - player id `>L`
  - access key `16s`, NUL-trimmed.
- Store `{host, port, version, access_key}` in Redis with TTL `keep_alive_interval * 1.5`.
- Reply `ok` over UDP on successful save.
