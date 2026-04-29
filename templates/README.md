# templates/

LW_show templates rendered by the GSC application
([internal/render](../internal/render/README.md)). Two parallel sets,
one per game family:

- `cs/` — Cossacks: European Wars, The Art of War, Back to War.
- `ac/` — American Conquest.

The renderer picks the directory based on a per-connection language
flag carried in the `gsc.Stream`. Most templates have a counterpart
in the other directory; the few that exist only in `cs/` (e.g.
`gg_cup_thanks_dgl.tmpl`, `room_info_dgl.tmpl`,
`started_room_info.tmpl`, `user_details.tmpl`) are gated by the
relevant route.

## File naming

- `*_dgl.tmpl` — modal dialogs (`alert`, `confirm`, etc.).
- `enter.tmpl`, `ok_enter.tmpl`, `error_enter.tmpl` — the entry
  pipeline.
- `startup.tmpl`, `reg_new_room.tmpl`, `join_room.tmpl`,
  `new_room_dgl.tmpl` — primary lobby pages.
- `started_room_info.tmpl` (and the `started_room_info/` shards) —
  in-match status pages.

## Variable expansion

Templates contain placeholders of the form `%NAME&fallback&%`. The
renderer fills them from a per-request variable map; unfilled
placeholders fall through to the literal `fallback` text. Time
intervals (`%TIME&...&%`) are formatted by helpers in
[internal/render](../internal/render/README.md). The expansion is
byte-exact with the historical reference server — any change to
placeholder names, fallback text, or whitespace alters the wire
output and breaks wire tests in
[internal/app/gsc/testdata](../internal/app/gsc/testdata).

## Editing a template

1. Update the relevant file under `cs/` and the matching one under
   `ac/`.
2. Run the wire tests (`go test ./internal/app/gsc/...`) and
   inspect the diff if any goldens fail. Goldens may need
   regeneration with documented justification.
3. Verify the rendered byte length and ordering match the client's
   expectations; the LW_show wire frame embeds the byte length, so a
   single off-by-one breaks every client read.
