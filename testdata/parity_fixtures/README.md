# Parity test fixtures (catalog)

Checked-in and planned paths for **contract tests, goldens, and static inputs** under `golang/`. (See also `golang/docs/parity_matrix.md` and `golang/docs/parity_command_contract.md`.)

## Parity and snapshot tests

| Location | Role |
|----------|------|
| `internal/server/commands/*_test.go` | In-memory `Controller` / `state.Store` / `model.Connection` scenarios; no external I/O by default. |
| `internal/server/commands/testdata/golden/*.json` | **Command metadata** snapshots: JSON array of `{ "name", "arg_lens" }` per `LW_*` command (not full CML / binary). |
| `internal/server/commands/golden_lw_flows_test.go` | Flows: open `enter` / `startup`, `GETTBL`, `reg_new_room`→`GETTBL`, `join_pl_cmd`→`room_info`, `join_game`, no-response `alive` / `leave` / `start`, across `ver=2` (cs) and `ver=8` (ac) where applicable. |
| `internal/server/state/state_test.go` | `Store` invariants, `NextRoomID`, etc. |
| `internal/protocol/gsc/*_test.go` | GSC command-set encode/decode, escaping. |

## Config and templates (read-only from repo root)

| Location | Role |
|----------|------|
| `config/simple-cossacks-server.yaml` | Go runtime config; keys stay parity-compatible (`host`, `port`, `hole_port`, `hole_int`, `templates`, etc.). |
| `templates/cs/*.tmpl` | Templates for `ver` not in AC (e.g. 2, 5, 6, 7). |
| `templates/ac/*.tmpl` | Templates for AC / `isAC(ver)` in `templates.go` (e.g. 3, 8, 10). |
| `internal/server/commands/templates.go` | Go CML subset renderer (not full Template Toolkit). |

## Optional runtime files (not checked in; Go tests set `c.Config.Raw` under `t.TempDir()`)

| Key / path | Used by |
|------------|---------|
| `lcn_ranking` | LCN place line in `user_details`, `tournaments` (when added). |
| `gg_cup_file` | `startup` GG Cup banner, `gg_cup_thanks_dgl`. |
| `supporters` JSON | `gg_cup` supporters block. |

## Updating golden JSON

When a **documented** behavior change (or template size change) updates outbound commands:

1. Run: `go test ./internal/server/commands/ -run TestGolden -count=1`
2. If a golden change is intended, run `go test ./internal/server/commands -golden` to rewrite `internal/server/commands/testdata/golden/*.json` and `testdata/template_fullbody/*.golden` (same outputs the tests compute).
3. Commit the JSON; avoid updating goldens to hide regressions.
