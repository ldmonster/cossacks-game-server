# Per-command contract (Go vs Perl)

**Canonical Perl:** `SimpleCossacksServer::CommandController` (GSC commands) and `...::CommandController::Open` (open / `go` routes). **Go:** `golang/internal/server/commands/controller.go` via `HandleWithMeta` and helpers.

**Outbound rule:** GSC `win` is appended to every outbound arg list by the framing layer in `golang/internal/server/core` (unchanged vs Perl: last args are `win`, `key` on input; response repeats `win` on each output command). This document describes **command names and body semantics** only.

**No-response vs empty (Go `Handle` / `HandleWithMeta`):**
- `HasResponse: false` → `Handle` returns `nil` (no commands).
- `HasResponse: true` and `Commands` empty `[]` → `open` routes that choose “empty” behavior (e.g. `join_pl_cmd` early `push_empty` in Perl) — client receives an **empty** command set, not “no response”. Document each edge case in the `open` section.

---

## GSC commands (non-open)

| Command | Request args (summary) | State / side effects | Outbound (typical) | Go `Handle` | Notes |
|--------|------------------------|----------------------|--------------------|------------|--------|
| `login` | — | none | one `LW_show` → `open&enter.dcml` | response | — |
| `open` | `url`, `params^…` (Perl); Go: `args[0]` = url, optional params | `dispatchOpen` | 0+ `LW_*` (usually `LW_show` with CML) | response | Unknown **public** method in Perl still hits `_default` (Page Not Found). Go: same via `default` in `dispatchOpen` only when method is non-public; unknown GSC `open` target goes through `handleOpen` strip `.dcml` and dispatch. |
| `go` | `method`, `k=v` / `k:=` pairs | `dispatchOpen` with parsed map | 0+ `LW_*` | response | — |
| `echo` | passthrough | none | `LW_echo` with same args | response | — |
| `GETTBL` | table name, num, `rows_pack` (U32LE ctlsum list) | reads `dbtbl` / `RoomsBy*`, hide-started rules | `LW_dtbl` then `LW_tbl` | response | Iteration over `RoomsByID` in Go: map order; parity tests avoid multi-row order sensitivity or sort by id if needed. |
| `stats` | packed stat blob, room id | `alive` refresh, updates player/room stat aggregates | (none) | no response | Perl: no `push` after work. |
| `alive` | optional args (logged on Perl) | resets 150s timer, `armAliveTimer` / replace timer | (none) | no response | — |
| `leave` | — | `leaveRoom`, clears alive timer | (none) | no response | — |
| `not_alive` | (timer callback) | not a GSC command: `notAlive` from timer | (none) | n/a | Invoked on timer, not client `Handle`. |
| `proxy` | `ip`, `port`, `key` | may reject + close; else rewrites `conn` IP / port | (none) | no response | — |
| `start` | `sav`, `map`, `players_count`, player quads | marks room started, `armAliveTimer` all players, optional `postAccountAction` | (none) | no response | Perl also does not push a client-visible command in the shown `sub`. |
| `endgame` | `game_id`, `player_id`, `result` | logging only in Perl/Go | (none) | no response | — |
| `upfile` | file chunk args | _before logging only in Perl | (none) | no response | Go: no file handling. |
| `unsync` | — | _before | (none) | no response | — |
| `url` | url string | `LW_time` + `open:…` | one `LW_time` | response | If no args, Go: no response. |
| (unknown) | — | log line | `[]` (empty set) | response, empty | Perl `Open` `_default` is only for `open`/`go`; unknown GSC name in Go logs and returns empty commands. |

---

## `open` / `go` routes (Perl `Open.pm` + Go `dispatchOpen`)

**Public name list (Perl `\@PUBLIC` / `public()`):** `enter`, `try_enter`, `startup`, `resize`, `games`, `rooms_table_dgl`, `new_room_dgl`, `reg_new_room`, `join_game`, `join_pl_cmd`, `user_details`, `users_list`, `direct`, `direct_ping`, `direct_join`, `room_info_dgl`, `started_room_message`, `tournaments`, `lcn_registration_dgl`, `gg_cup_thanks_dgl`.

**Subroutines actually defined in `Open.pm`:** `enter`, `try_enter`, `startup`, `resize`, `games`, `new_room_dgl`, `reg_new_room`, `room_info_dgl`, `join_game`, `user_details`, `join_pl_cmd`, `tournaments`, `lcn_registration_dgl`, `gg_cup_thanks_dgl`, `users_list`, and helpers. Names in the public list **without** a `sub` would call inherited/UNIVERSAL (typically errors at runtime) unless another mechanism exists — **in Go** these are treated explicitly:

| Route | Params (high level) | Response | Edge / no-response |
|--------|--------------------|----------|--------------------|
| `enter` | `TYPE` optional | `LW_show` + `enter.cml` | Logged-in uses `account` on connection. |
| `try_enter` | `NICK`, `TYPE`, `PASSWORD`, `LOGGED_IN`, `RESET` | `enter` / `ok_enter` / errors | LCN/WCL: HTTP; may `postAccountAction(enter)` on `LOGGED_IN`. |
| `startup`, `games`, `rooms_table_dgl` | — | `LW_show` + `startup.cml` (Go merges `gg_cup` vars) | `rooms_table_dgl` in Go reuses the same `startup` branch. |
| `resize` | `height` | `LW_show` resize block | — |
| `new_room_dgl` | `ASTATE` | `new_room_dgl` or error `alert_dgl` | ASTATE 0/empty: cannot create. |
| `reg_new_room` | many `VE_*` + room params | new room, `GETTBL` refresh pattern via CML/Perl | Truncation / CS rules in tests. |
| `join_game` | `VE_RID`, `VE_PASSWD`, `ASTATE`, … | `join_room` CML, confirms, or errors | Calls `leaveRoomByID` before join (see plan / `join_game` in Go). |
| `join_pl_cmd` | `VE_PLAYER` | 0+ `LW_*` or none | In room: **empty**. No room: **no** outbound. Started: `alert_dgl` (typo: “alredy”). Else: `room_info_dgl` flow. |
| `room_info_dgl` | `VE_RID` + host/join | `room_info` / `started` CML | — |
| `user_details` | `ID` | `user_details.cml` or no-op + warn | — |
| `tournaments` | `option` | LCN-style rating CML in Perl; Go: `alert_dgl` with lines | On missing ranking: error. |
| `lcn_registration_dgl` | — | `confirm_dgl` | — |
| `gg_cup_thanks_dgl` | — | `gg_cup_thanks_dgl` CML (supporters) | — |
| `users_list` | — | `alert_dgl` “Not imlemented” | — |
| `direct`, `direct_ping`, `direct_join`, `started_room_message` | (ignored) | `[]` in Go | **No Perl `sub`**; undefined client behavior. |

---

## Concurrency and disconnect

- **State mutation** for requests: serialized by `stateMu` in `HandleWithMeta` (see `refactoring_plan.md`).
- **OnDisconnect** and timer **`notAlive`**: also take `stateMu`; leave room, map cleanup per tests (`controller_alive_timer_test.go`).

---

## Related

- **Wire and frame layout:** [parity_matrix.md](parity_matrix.md) (this folder).
- **Test fixtures and golden path index:** [testdata/parity_fixtures/README.md](../testdata/parity_fixtures/README.md) (from repo: `golang/testdata/parity_fixtures/README.md`).
- **Golden snapshots (command metadata):** `internal/server/commands/golden_lw_flows_test.go` and `internal/server/commands/testdata/golden/*.json` (per-command name + per-arg length; not full CML text). Refresh with `go test ./internal/server/commands -golden`.
