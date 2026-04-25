# Programmer checklist (reviewer-maintained)

**Source:** Derived from `refactoring_plan.md` (repo root). It is the **task queue for implementation work** in `golang/`.

**Who edits this file:** **Reviewer only** (in-repo updates). The reviewer **removes** stale or completed tasks, **adds** new ones when gaps appear, and keeps order aligned with the plan. Programmers follow the list and do not rewrite sections for policy—ask the reviewer to adjust the file if scope changed.

**Last review sync:** 2026-04-25 (`/reviewer-compare` share parity pass: plan clarified, add template-root task).

---

## How to use (programmer)

1. Read `refactoring_plan.md` for context, then work from **unchecked** items below (top to bottom unless a task names a dependency).
2. One PR-sized slice: implement + tests; run `go test ./...` and `go test -race` for affected packages.
3. In the PR or chat, reference the **checklist line** you completed. Do not delete or renumber rows; the reviewer updates the file when merging knowledge.

## Active tasks (unchecked = open)

- [ ] **Phase 2 – `room_info_dgl` / started views:** Close remaining gaps vs Perl TT for `room_info_dgl.cml`, `started_room_info*.cml` / `statcols` (use/extend `controller_roominfo_*_test.go`); add missing field coverage where Go flattens or omits.
- [ ] **Phase 2 – `reg_new_room`:** Exercises for game-version / `is_american_conquest` row fields and byte-level response var parity with Perl (extend `controller_reg_new_room_parity_test.go`).
- [ ] **Phase 4 – liveness edge cases:** Add or extend cases for `alive` refresh, `not_alive`, and disconnect so timer/map behavior matches Perl edge cases (see plan §D).
- [ ] **Optional / product-dependent:** If a real client needs behavior for `direct` / `direct_ping` / `direct_join` / `started_room_message`, define parity from traces or agreed spec, then implement + test (see plan §A—currently no Perl `sub` for these names).

## Completed (reviewer: move done items here and add date)

- **Build — `supporterAmountString`:** `json.Number` branch now returns on all paths (`controller.go`); `go test ./...` and `go test -race ./internal/server/commands/...` pass. (2026-04-25)
- **Phase 0 – contract:** Added full command contract doc (`golang/docs/parity_command_contract.md`), linked from `parity_matrix.md`, with args/state/response/side-effects and no-response vs empty semantics; added checked-in fixture catalog (`golang/testdata/parity_fixtures/README.md`). (2026-04-25)
- **Phase 0 – goldens:** Added/extended `LW_*` metadata snapshots in `internal/server/commands/testdata/golden` and `golden_lw_flows_test.go` covering enter→startup, create room→GETTBL, join→room info, start, leave/timeout-alive across `ver=2` and `ver=8`. (2026-04-25)
- **Phase 2 – `join_pl_cmd`:** Tightened no-response vs routed edge tests (`controller_join_player_parity_test.go`) and fixed `VE_PLAYER` trim/parsing parity in `controller.go`. (2026-04-25)
- **Phase 6 – template path resolution:** Added `commands.ConfigureTemplateRoots(cfg.Templates)` wiring in `cmd/cossacksd/main.go` and template-root builder logic in `templates.go` (`custom templates` first, hardcoded roots as fallback); added unit coverage in `templates_path_resolution_test.go`. (2026-04-25)
- **Phase 6 – templates:** Expanded high-risk template golden coverage (`golden_lw_flows_test.go` + `testdata/golden`) for `new_room_dgl`, `user_details`, and started-room views (`started_room_info.cml` + `started_room_info/statcols.cml`) across `ver=2` and `ver=8`; also added `/cossacks/SimpleCossacksServer/share` fallback root for docker-compose volume parity. (2026-04-25)

## Notes (reviewer freeform)

- **Concurrency:** `stateMu` + `go test -race ./...` is the current bar; if locking model changes, add a checklist line for new `Store` API or stress tests.
- **`tournaments` / `gg_cup_thanks_dgl`:** Success paths load data; if layout must match Perl exactly, add a specific task and link tests.
- **`gg_cup_thanks` overflow:** No test yet asserts `"and more..."` when supporter count exceeds the row cap (`buildGGCupThanksCML`); add when locking TT parity (fits Phase 6 / goldens).
