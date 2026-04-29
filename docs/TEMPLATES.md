# Templates: a comprehensive guide

This document is the reference for the on-disk **`.tmpl`** files under [templates/cs/](../templates/cs/) and [templates/ac/](../templates/ac/), and for the small Template Toolkit–style renderer that turns them into **`LW_show`** payloads sent to the legacy Cossacks / American Conquest client.

It covers:

1. [What templates are and how they fit into the server](#1-what-templates-are)
2. [File layout, lookup, and resolution](#2-file-layout-lookup-and-resolution)
3. [The TT-subset language understood by the Go renderer](#3-the-tt-subset-language)
4. [Variables: how they flow from Go into a template](#4-variables)
5. [Show-body output: directives, regions, and styles](#5-show-body-output)
6. [Client-side tokens: `GV_*`, `CG_*`, `LW_*`, `GW|`, layout ids](#6-client-side-tokens)
7. [Catalogues of every template and where it is rendered](#7-template-catalogue)
8. [Authoring workflow, testing, and maintenance](#8-authoring-and-testing)
9. [Cheat sheet and further reading](#9-cheat-sheet)

> **Source of truth.** Whenever this guide and the code disagree, the code wins. The renderer is implemented in [internal/render/templates.go](../internal/render/templates.go); template-driven dispatch lives in [internal/app/gsc/controller.go](../internal/app/gsc/controller.go) and its siblings.

---

## 1. What templates are

A **template** in this repository is a plain UTF-8 text file with a **`.tmpl`** extension that contains:

- **Show-body primitives** the game client understands at runtime (`#font`, `#txt`, `#ebox`, `#edit`, `#btn`, `GW|…`, `LW_*`, …). These are emitted **as-is**.
- **TT-style fragments** (`<? … ?>`, with `<? IF ?> / <? ELSE ?> / <? END ?>`) that the **Go server** evaluates while loading the file. After evaluation only the show-body primitives remain.

The renderer is intentionally minimal: it is **not** Go's `text/template`, **not** `html/template`, and **not** a full Perl Template Toolkit. It is a hand-written subset tailored to the original Cossacks server templates so existing CML/LW assets keep working.

### Pipeline at a glance

```
.tmpl file ──► LoadShowBodyFromRoots ──► RenderShowTemplate ──► LW_show body
                      │                          │
                      │                          ├─ renderInlineIfBlocks  (inline <? IF ?>…<? END ?>)
                      │                          ├─ line-based <? IF/ELSE/END ?>
                      │                          └─ <? expr ?> interpolation (evalExpr / evalCondition)
                      │
                      └─ search roots × cs|ac × name.tmpl  ── falls back to FallbackShowBody
```

Implemented in [internal/render/templates.go](../internal/render/templates.go) (`LoadShowBodyFromRoots`, `RenderShowTemplate`, `evalExpr`, `evalCondition`, `lookupVar`, `renderInlineIfBlocks`, `FallbackShowBody`).

### Where templates are rendered

- **`port.TemplateRenderer`** ([internal/port/renderer.go](../internal/port/renderer.go)) — the abstraction the controller depends on.
- **`render.TemplateRenderer`** ([internal/render/templates.go](../internal/render/templates.go#L46)) — the production implementation.
- **`render.Service`** ([internal/render/service.go](../internal/render/service.go)) — nil-safe wrapper used inside controllers.
- **`Controller.render`** ([internal/app/gsc/controller_render.go](../internal/app/gsc/controller_render.go)) — the dispatcher used by every GSC handler.

---

## 2. File layout, lookup, and resolution

### 2.1 `cs/` vs `ac/`

The client family is selected by the protocol version `ver`:

| Directory | When used | Versions |
|-----------|-----------|----------|
| [templates/cs/](../templates/cs/) | Cossacks family (non-AC) | 2, 5, 6, 7 (`IsAC(ver)` is false) |
| [templates/ac/](../templates/ac/) | American Conquest family | 3, 8, 10 (`IsAC(ver)` is true) |

The mapping is `IsAC` in [internal/render/templates.go](../internal/render/templates.go#L98):

```go
func IsAC(ver uint8) bool { return ver == 3 || ver == 8 || ver == 10 }
```

If a template only makes sense for one family, leave the other absent — `LoadShowBodyFromRoots` will fall back to `FallbackShowBody()`.

### 2.2 Filename normalization

`NormalizeShowTemplateName` accepts:

- A plain name: `enter` → `enter.tmpl`.
- A `.tmpl` suffix: passed through.
- A `.cml` suffix: rewritten to `.tmpl` (so legacy `.dcml`/`.cml` references work).
- Forward slashes for sub-paths: `started_room_info/statcols`.

Empty or whitespace-only names yield `""` and the fallback body is returned.

### 2.3 Search roots

`DefaultTemplateRoots` ([internal/render/templates.go](../internal/render/templates.go#L34)) is, in order:

1. `/app/templates`
2. `/cossacks/templates`
3. `templates`
4. `../templates`
5. `../../templates`
6. `/cossacks/SimpleCossacksServer/share`

`NewTemplateRenderer(customRoot)` puts `customRoot` (e.g. from configuration) **before** the defaults via `BuildTemplateRoots`, deduplicating empty entries. The resolver tries `{root}/{cs|ac}/{name}.tmpl` against each root, returning the first hit. If every root misses, the result is the minimal **fallback body**:

```
#font(WF,WF,WF)
#txt(%BOX[x:10,y:10,w:100%,h:24],{},"server response")
```

The fallback is intentionally ugly so missing templates are obvious in tests and on screen.

### 2.4 Sub-paths

Sub-directories are allowed (and used). The only example today is [templates/cs/started_room_info/statcols.tmpl](../templates/cs/started_room_info/statcols.tmpl), reached as `started_room_info/statcols.tmpl`.

---

## 3. The TT-subset language

`RenderShowTemplate(src, vars)` runs three passes over the source string:

1. **Inline `<? IF ?>…<? END ?>` blocks** are replaced first via `renderInlineIfBlocks`. This handles control flow embedded inside a single physical line (e.g. inside a `#apan` argument).
2. **Whole-line directives** (`<? IF ?>`, `<? ELSE ?>`, `<? END ?>`) are interpreted line-by-line and their lines are removed from the output.
3. **Remaining `<? expr ?>` fragments** are replaced by the string result of `evalExpr`.

After all three passes the result is `strings.TrimSpace`d and returned.

### 3.1 Two delimiter families

| Delimiter | Evaluated by | Purpose |
|-----------|--------------|---------|
| `<? … ?>` | **Server** (Go renderer) | TT-subset: control flow and expressions. |
| `<% … %>`, `{% … }`, `<%NAME>`, `{%NAME}` | **Client** (game binary) | Live bindings to client globals (`GV_*`, `CG_*`, `ASTATE`, `RL_ID`, …). |

The Go renderer **never** touches `<%…%>` or `{%…}`. It just passes them through into the show body for the client to interpret.

### 3.2 Whole-line vs inline `<? IF ?>`

A line is treated as a **whole-line directive** when, after trimming, it starts with `<?`, ends with `?>`, and contains **no** `<%`. Recognized forms (after stripping leading/trailing `~`, TT-style):

- `<? IF condition ?>` — opens a nested block.
- `<? ELSE ?>` — flips the current block.
- `<? END ?>` — closes the current block.

Anything else on a whole-line `<? … ?>` (e.g. `<? USE Date ?>`, `<? FOREACH … ?>`) **is silently dropped**. These constructs are **not** implemented; legacy TT directives in old templates serve as documentation only.

**Inline** blocks are the exact same syntax but on a single span:

```
#txt(%BOX[…], {}, "<? IF nick ?>Hello, <? nick ?>!<? ELSE ?>Hello.<? END ?>")
```

Inline blocks may be nested by repetition (the regex iterates until no match remains), but they cannot reference a label across lines once the inline pass has completed.

### 3.3 Conditions (`evalCondition`)

`evalCondition` walks the expression with the following precedence (top is **lowest**):

1. `||` (left-to-right, short-circuit).
2. `&&` (left-to-right, short-circuit).
3. Unary `!` (binds the whole right-hand side after trimming).
4. `!=`, `>=`, `<=`, `>`, `<`, `==` (string `==`/`!=`, numeric for the others via `compareNum`).
5. **Bare expression** — the value from `evalExpr` is **truthy** when it is non-empty, not `"0"`, and not `"false"` (case-insensitive).

Notes and gotchas:

- `>=`, `<=`, `>`, `<` parse both sides with `strconv.ParseFloat`. Non-numeric operands silently make the comparison **false**.
- The split-on-`||` / split-on-`&&` is **textual**: parentheses do not protect the operator. Avoid nested logical operators on a single line — split them across `<? IF ?>` blocks instead.
- `==` and `!=` compare **trimmed strings**. Quote literals: `<? type == 'LCN' ?>`.

### 3.4 Expressions (`<? expr ?>`)

`evalExpr` handles, in order:

1. Strip a `| filter` suffix — only the **left** side is evaluated. Filters like `| cmd`, `| arg`, `| date`, `| CMLStringArgFilter` exist in legacy assets but are **not** executed.
2. Try `tryEvalAddMul`: top-level `+` and `*` arithmetic, left-associative, integer-truncated result.
3. Otherwise call `evalExprLeaf`:
   - **Quoted literal** (`'…'` or `"…"`) — return inner text.
   - **Ternary** `cond ? a : b` — split at the first top-level `?` followed by a `:` further right.
   - `==` short-circuit — returns `"1"` or `""`.
   - **`.length`** suffix — UTF-8 rune count of the inner expression's value (empty → `"0"`).
   - **`POSIX.floor(x)`** — floor of the numeric value; non-numeric → `"0"`.
   - `h.req.ver` → `vars["ver"]`.
   - `server.config.X` → `lookupVar("X", vars)`.
   - `P.X` → `lookupVar("X", vars)` (param-style alias).
   - **Numeric literal** — returned verbatim.
   - ` _ ` (space-underscore-space) — string concatenation of evaluated parts.
   - Otherwise — `lookupVar(name, vars)`.

### 3.5 Things that look like TT but are **not** supported

| Construct | Status |
|-----------|--------|
| `FOREACH`, `WHILE`, `SWITCH`, `CASE`, `BLOCK`, `MACRO`, `INCLUDE`, `WRAPPER`, `PROCESS` | **Not implemented**. Whole-line forms are dropped silently; inline forms are passed through. |
| `USE Date`, `USE …` | **Not implemented**. Filter pipes are stripped without execution. |
| `${expr}` interpolation in keys | **Not implemented**. Build the key in Go first. |
| `expr.method(args)` calls (`date.format(…)`, list joins, …) | **Not implemented**. Pre-format in Go, set a flat key. |
| Arithmetic with `-`, `/`, `%`, `**` | **Not implemented**. Only `+` and `*` work. |
| Parenthesized sub-expressions | Tokenization respects parentheses for `+`/`*` only; ternaries inside arithmetic are unreliable. Keep complex math in Go. |

When in doubt: pre-compute the value in the call site and pass it as a flat string in `vars`.

### 3.6 Tilde trimming

`normalizeTT` strips a single leading and trailing `~` from a directive body, mirroring TT's whitespace-control idiom (`<?~ … ~?>`). The renderer does **not** otherwise alter whitespace.

---

## 4. Variables

All template data is a single **`map[string]string`**. Every value is a string; the template never sees structured data.

### 4.1 Call sites

Templates are loaded in two equivalent ways:

```go
// Through the controller, the canonical path:
body := c.render(req.Ver, "enter.tmpl", vars)

// Or directly:
body := renderer.Render(ver, "enter.tmpl", vars)
```

The controller wrapper lives in [internal/app/gsc/controller_render.go](../internal/app/gsc/controller_render.go); search the codebase for `c.render(` to find every dispatched template.

### 4.2 How a name is resolved

Inside a `<? … ?>` fragment, `lookupVar(name, vars)` resolves a bare identifier:

1. A small **switch** of well-known short keys returns the matching map entry. Most are pass-through (`id`, `nick`, `error_text`, `chat_server`, `logged_in`, `type`, `window_size`, `table_timeout`, `ver`, `header`, `text`, `ok_text`, `height`, `command`, `ip`, `port`, `max_pl`, `name`, `active_players`, `exited_players`, `has_exited_players`, `room_players_start`).
2. **Legacy alias**: `NICK` is normalized to `nick` (uppercase variant of the same value).
3. The default branch is `vars[name]` after trimming.

Outside the switch, the **dotted key is the literal string**: `room.title` reads `vars["room.title"]`. Builders in `internal/render/` (e.g. `MergeRoomDottedVars`, `BuildRoomInfoVars`) populate dotted keys explicitly so the templates can reference them naturally.

### 4.3 Special prefixes

| Source form | Resolved as |
|-------------|-------------|
| `h.req.ver` | `vars["ver"]` |
| `server.config.X` | `vars["X"]` (legacy) |
| `P.X` | `vars["X"]` (param-style alias) |

Use these forms where the original templates already do; otherwise pass the value flat.

### 4.4 Truthiness for `IF`

A bare value is **truthy** unless it is empty, `"0"`, or `"false"` (case-insensitive). This applies to both `<? IF flag ?>` and `<? IF flag && other ?>`.

### 4.5 Rendering helpers

[internal/render/](../internal/render/) ships builders that produce `map[string]string` payloads ready to pass to `Render`. Use them rather than constructing maps inline, so dotted keys stay consistent across CS and AC:

| File | Purpose |
|------|---------|
| [room_info.go](../internal/render/room_info.go) | `BuildRoomInfoVars`, `BuildStartedRoomInfoVars`, `RoomInfoBackto`. |
| [room_lifecycle_vars.go](../internal/render/room_lifecycle_vars.go) | `MergeRoomDottedVars`, `StartedPlayerNames`, `RoomTimeInterval`. |
| [room_vars.go](../internal/render/room_vars.go) | `BuildRegNewRoomVars`, `BuildJoinRoomVars`, `SetRoomPlayersColumn`. |
| [user_details.go](../internal/render/user_details.go) | `UserDetailsBody`, plus CML-safe helpers. |
| [ggcup.go](../internal/render/ggcup.go) | `GGCupThanksBody`, `GGCupThanksBoxHeight`. |
| [time_interval.go](../internal/render/time_interval.go) | `TimeIntervalFromElapsedSec`. |
| [cml.go](../internal/render/cml.go) | `CMLSafe` — quote/newline scrubbing for free-form data. |
| [builders.go](../internal/render/builders.go) | `Show`, `Echo`, `Time`, `Alert` response wrappers. |

---

## 5. Show-body output

After the Go renderer finishes, the result is the body of an **`LW_show`** command sent to the client. Everything below is interpreted by the **client**, not the server.

### 5.1 Drawing directives

Each directive is `#name(args)` on its own line (or chained inside a widget action). Common ones:

| Directive | Purpose |
|-----------|---------|
| `#font(A,B,C)` | Set the active font/colour triple for the next text directives. See [§5.4](#54-font-triples). |
| `#txt(box, style, "text")` | Static text (left-aligned). |
| `#ctxt(box, style, "text")` | Centred text. |
| `#rtxt(box, style, "text")` | Right-aligned text. |
| `#edit(box, {bind})` | Editable input bound to a `GV_*` global. |
| `#cbb(box, {bind}, "opt1", "opt2", …)` | Combo box / dropdown. |
| `#btn[%STYLE](box, {action}, "label")` | Button with style id and action pipeline. |
| `#sbtn[%STYLE](box, {action}, "label")` | Submit-style button (default footer button). |
| `#pan[%ID](box)` | Plain panel container. |
| `#apan[%ID](box, {action})` | **Action** panel — a clickable region. |
| `#ebox[%ID](box)` / `#box[%ID](box)` | Empty box / generic box. |
| `#exec(LW_…&args)` | Execute an `LW_*` opcode at show time. |
| `#resize(LW_cfile&…|LW_show&…)` | Read a client config file then run a nested show body. |
| `#DBTBL(…)` | Database-driven table (rooms list). |

The Go renderer treats each directive as text. If you need to compute a coordinate or label, do it in `vars` and reference it via `<? … ?>`.

### 5.2 Region descriptors

Rectangular regions are written `%BOX[x:…, y:…, w:…, h:…]`. Coordinates may be:

- Numeric literals (`x:10`).
- Percentages (`w:100%`).
- Anchor references with offsets: `y:%L_NAME+6` (six pixels below the line registered under `L_NAME`). The client maintains the registry; the Go server does not.
- Conditionally-chosen anchors via `<? … ?>` evaluated server-side, e.g. `y:%<? has_exited_players ? "T_EXPLAYERS" : "T_PLAYERS" ?>+6`.

Common region/identifier prefixes:

| Prefix | Typical role |
|--------|--------------|
| `%BOX`, `%LBX`, `%MPN` | Outer dialog regions: dialog box, lockbox, main panel. |
| `%TIT` | Title bar / region. |
| `%L_…` | Static label rows (`L_NAME`, `L_HOST`, `L_PASSWD`, …). |
| `%T_…` | Value rows paired with `L_…` (`T_NAME`, `T_PLAYERS`, …). |
| `%E_…`, `%P_…` | Edit / panel rows on form dialogs (`E_NAME`, `E_PASS`, `P_NICK`, …). |
| `%B_…` | Button **chrome** ids (`B_RGST` standard footer, `B_C` AC create, `B_J` AC join). |

### 5.3 Layout id catalogues

#### `L_*` (static label rows)

| Id | Used in | Role |
|----|---------|------|
| `L_NAME` | most dialog templates | First/title label. |
| `L_PASS`, `L_MAXPL`, `L_LEVEL` | new room dialogs | Form labels. |
| `L_TYPE` | AC new room dialog | Battle type label (AC only). |
| `L_CTIME` | user_details, room_info_dgl | "Connected at" / "Created at". |
| `L_ACCOUNT`, `L_ROOM` | user_details | Account and room labels. |
| `L_HOST`, `L_PLAYERS`, `L_EXPLAYERS`, `L_PASSWD` | room_info_dgl | Room info labels. |

#### `T_*` (value rows paired with `L_*`)

| Id | Role |
|----|------|
| `T_CTIME` | Time / date string. |
| `T_ACCOUNT` | Account type string. |
| `T_ROOM` | Room title; **Join**/**Info** buttons align to it. |
| `T_NAME`, `T_HOST`, `T_PLAYERS`, `T_EXPLAYERS`, `T_LEVEL`, `T_PASSWD` | Room info values. |

#### `B_*` (button chrome ids)

| Id | Role |
|----|------|
| `B_RGST` | Standard footer button (Enter / Cancel / OK across most dialogs). |
| `B_C` | AC startup **Create** room button. |
| `B_J` | AC startup **Join** room button. |

Reuse ids from the nearest existing dialog rather than inventing new ones; the client only knows the chrome resources for ids it ships with.

### 5.4 Font triples

`#font(A,B,C)` takes three comma-separated **font/colour slot names** that the client maps to skin resources. The Go server does not interpret them. Common patterns:

| Triple | Typical use |
|--------|-------------|
| `WF,WF,WF` | Default body / dialog text. |
| `YF,YF,YF` / `YF,YF,WF` | Emphasis, table cells, nick fields. |
| `YF,WF,BF` / `WF,WF,BF` | Logon / link rows. |
| `SYF,SWF,SWF` | Strong styling, often error rows. |
| `RF,RF,RF` | Red — exited players, warnings. |
| `B1F40`, `YF16` | Sized / themed variants on AC `startup.tmpl`. |

Letters are mnemonic in original assets (**Y**=yellow, **W**=white, **B**=blue, **F**=face/font). Match the nearest existing dialog when you add new lines so colours stay consistent.

---

## 6. Client-side tokens

These tokens flow **through** the Go renderer untouched and reach the client embedded in the `LW_show` body. The server does **not** know what they mean — knowing the catalogue below is enough to keep CS and AC templates in sync with the original client.

### 6.1 `GV_*` — client globals

`GV_*` names live in the client's gvar namespace. Templates use them in two shapes:

- **`{%GV_NAME}`** — the **binding target** of `#edit`, `#cbb`, etc.
- **`<%GV_NAME>`** — embedded inside a `GW|…` action so the **current** value is substituted on activation.

| Name | Templates | Role |
|------|-----------|------|
| `GV_LCN_NICK` | cs/enter, ac/enter, ac/new_room_dgl | Nickname; on AC create room also pushed as `VE_NICK` in `GW|open&reg_new_room`. |
| `GV_LCN_INFR` | cs/started_room_info/statcols | Resource tab selector for started-room stats. |
| `GV_LCN_PROF` | cs/ok_enter, ac/ok_enter (commented `LW_cfile`) | Intended for profile / cookie persistence. |
| `GV_VE_NICK` | ac/enter (commented) | AC cookie / nick file hook. |
| `GV_VE_TITLE` | cs/new_room_dgl, ac/new_room_dgl | New room title field. |
| `GV_VE_PASSWD` | same | New room password. |
| `GV_VE_MAX_PL` | same | Max players (`#cbb`, choices 2–7). |
| `GV_VE_LEVEL` | same | Difficulty (Easy / Normal / Hard). |
| `GV_VE_TYPE` | ac/new_room_dgl only | Battle type combobox (Ordinal / Battle). CS form omits this. |

When you add a `GV_*` field:

1. Pick a name consistent with neighbours (`GV_VE_*` for room form fields, `GV_LCN_*` for lobby). The client must already understand the name — invent only if you control the client too.
2. Bind with `{%GV_NEW}` on the widget; reference `<%GV_NEW>` in the corresponding `GW|…` action.
3. Wire the **server** route that handles the form to read the matching parameter (`parseOpenParams`, `handleGo`, `dispatchOpen` in [internal/app/gsc/controller.go](../internal/app/gsc/controller.go)).

### 6.2 `CG_*` — game-launch globals

`CG_*` are pushed in a single `#exec(LW_gvar&%CG_…&value&…\00)` line in `reg_new_room.tmpl` / `join_room.tmpl`. They tell the native game executable how to host or join a session.

| Name | Meaning | reg_new_room | join_room |
|------|---------|--------------|-----------|
| `CG_GAMEID` | Game / room id (sometimes prefixed `HB…` for AC battle type). | CS + AC | CS + AC |
| `CG_MAXPL` | Maximum players. | CS + AC | CS + AC |
| `CG_GAMENAME` | Room title. | CS + AC | CS + AC |
| `CG_IP` | Host IP for joins. | — | CS + AC |
| `CG_PORT` | Host port for joins. | — | **CS only** |
| `CG_HOLEHOST` | STUN / hole-punch host. | **CS only** | — |
| `CG_HOLEPORT` | Hole-punch port. | **CS only** | — |
| `CG_HOLEINT` | Hole-punch interval (seconds). | **CS only** | — |

The same `LW_gvar` line also sets **`%COMMAND`**:

- `CGAME` after `reg_new_room` succeeds (host).
- `JGAME` after `join_room` succeeds (joiner).

The TCP transport ([internal/transport/tcp/server.go](../internal/transport/tcp/server.go)) sniffs `CG_HOLEHOST`, `CG_HOLEPORT`, `CG_HOLEINT` from outgoing show bodies for STUN / hole-punch debugging via `extractGVar`.

### 6.3 `LW_*` — two layers

`LW_*` appears in **two distinct places**:

#### As a top-level GSC command name (the wire payload)

Built in Go, sent to the client. Templates do not produce these directly; they produce the **body** that goes into `LW_show`.

| Command | Role |
|---------|------|
| `LW_show` | The dominant command — carries the rendered template body. |
| `LW_echo` | Debug echo of arguments (response to `echo`). |
| `LW_time` | Delayed action; used by the `url` handler to open a browser after a timer. |
| `LW_dtbl` / `LW_tbl` | Table definition + row data; used for the rooms list and similar. |

#### As tokens **inside** an `LW_show` body

Interpreted by the **client** while running the show. Templates emit them via `#exec(LW_…)`, `#resize(LW_…|…)`, or chained inside `{…}` widget actions.

| Token | Pattern | Role |
|-------|---------|------|
| `LW_gvar` | `#exec(LW_gvar&%K1&v1&%K2&v2…\00)` | Push name/value pairs into client globals (`%PROF`, `%NICK`, `%CG_*`, `%COMMAND`, …). |
| `LW_lockbox` | `#exec(LW_lockbox&%LBX)` | Lock the dialog container as a modal. |
| `LW_lockall` | `…\|LW_lockall` after a `GW|` | Lock the entire UI until the next server update. |
| `LW_enb` | `#exec(LW_enb&0&%RMLST)` | Enable/configure a list/table control. |
| `LW_key` | `{LW_key&#CANCEL\|LW_lockall}` | Inject a logical key press (Cancel, etc.). |
| `LW_file` | `{LW_file&Internet/Cash/cancel.cml}` | Load a bundled `.cml` from game assets. |
| `LW_cfile` | `#resize(LW_cfile&<#WinH#>&height.dat\|LW_show&…)` | Read a client-side config file (e.g. window height). |
| `LW_show` (nested) | inside the same `#resize` pipeline | Run another show fragment after `LW_cfile`. |
| `LW_visbox` | (commented in legacy templates) | Toggle a box's visibility. |

### 6.4 `GW|` — client-side command pipelines

`GW|` is the legacy command-string prefix used inside widget actions (`{GW|…}`). The client parses the string and later issues GSC commands against the server.

**Wire shape**: `GW|verb&arg1&arg2|verb&arg1` — segments separated by `|`, arguments separated by `&`. Reserved characters are escaped (see [§6.6](#66-escaping)).

| Verb (first token) | Server route | Notes |
|--------------------|--------------|-------|
| `open&<resource>` | `dispatchOpen` in [controller.go](../internal/app/gsc/controller.go) | Open a dialog. The server strips a `.dcml` suffix. A second `&`-segment may carry `KEY=value` pairs separated by `^` (parsed by `parseOpenParams`). |
| `go&<method>&…` | `handleGo` | Form / action submission. Args after the method are `key=value`; the alternate `key:=` form takes the next argument as the value. |
| `url&<http…>` | `case "url"` in `HandleWithMeta` | Open an external URL in a browser, returned as `LW_time` with `open:` payload. |
| `login&…` | login route | Triggers the login flow. |

**Chaining**: `{GW|open&dialog.dcml|LW_lockall}` — the `LW_*` tail is a client-side helper executed after the server call.

**Embedding server values**: use `<? … ?>` to inject a flat `vars` value at render time. Use `<%…%>` (`<%ASTATE>`, `<%RL_ID>`, `<%PASSWORD>`) to defer to the client at submit time.

**Leading colon**: payloads sometimes begin with `:GW|…` inside a show body. The colon is a client convention for embedding a pipeline in show text; it is **not** part of the GSC `GW|` prefix.

### 6.5 Other `%…` placeholders to expect

Beyond `GV_*` and `CG_*`:

- `<%ASTATE>` — current application state, often passed through `open&page.dcml&ASTATE=<%ASTATE>`.
- `<%RL_ID>`, `<%RL_HOST>`, `<%RL_TITLE>` — room-list row columns picked from the `RMLST` table.
- `<%PASSWORD>`, `<%VE_PASSWD>` — password fields.
- `<@%HEIGHT>` — current window height (in `#resize` pipelines).
- `<#WinH#>` — built-in client expression for window height.

The Go server does not enumerate these symbols; copy from an adjacent template that already does what you need.

### 6.6 Escaping

When `&`, `|`, `\`, or NUL appear **inside** a string that is already an arg of `GW|…` or `LW_…`, they are escaped as backslash-hex byte sequences:

| Sequence | Byte |
|----------|------|
| `\26` | `&` |
| `\7C` | `\|` |
| `\5C` | `\` |
| `\00` | NUL |

The same escaping is implemented in Go for the wire protocol — see `CommandFromString` / `CommandSetFromString` in [internal/transport/gsc/](../internal/transport/gsc/). A token like `\52ESIZE` you may see in templates is **uppercase 'R'** + literal `ESIZE`, i.e. an obfuscated `RESIZE`.

For free-form user data inserted into a CML literal (room titles, nicks, …), use **`render.CMLSafe`** ([internal/render/cml.go](../internal/render/cml.go)) before embedding: it converts double quotes to apostrophes and collapses newlines/CR to spaces.

---

## 7. Template catalogue

Templates by family; each row also lists the primary call site that loads it. Search [internal/app/gsc/](../internal/app/gsc/) for the exact handler when you need the full `vars` map.

### 7.1 Cossacks ([templates/cs/](../templates/cs/))

| Template | Loaded by | What it shows |
|----------|-----------|---------------|
| [enter.tmpl](../templates/cs/enter.tmpl) | `renderEnter` | Login screen; branches on `type` (anonymous vs LCN/WCL) and `logged_in`. Sets `error`, height. |
| [ok_enter.tmpl](../templates/cs/ok_enter.tmpl) | enter / login post-auth | Confirmation: pushes `%PROF`, `%NICK`, etc. into client globals. |
| [error_enter.tmpl](../templates/cs/error_enter.tmpl) | login error path | Auth failure dialog. |
| [startup.tmpl](../templates/cs/startup.tmpl) | `go&startup` route | Lobby: rooms table, GG Cup marketing block, server start time. |
| [new_room_dgl.tmpl](../templates/cs/new_room_dgl.tmpl) | `open&new_room_dgl` | Create-room form (title, password, max players, level). |
| [reg_new_room.tmpl](../templates/cs/reg_new_room.tmpl) | `regNewRoom` | Host-side confirmation; pushes `%CG_*` + hole-punch settings + `%COMMAND=CGAME`. |
| [join_room.tmpl](../templates/cs/join_room.tmpl) | `joinGame` | Joiner-side confirmation; pushes `CG_IP`, `CG_PORT`, `%COMMAND=JGAME`. |
| [confirm_password_dgl.tmpl](../templates/cs/confirm_password_dgl.tmpl) | password-protected join | Password entry; submits to `go&join_game`. |
| [confirm_dgl.tmpl](../templates/cs/confirm_dgl.tmpl) | reusable | Generic OK/Cancel using `header`, `text`, `command`, `ok_text`, `height`. |
| [alert_dgl.tmpl](../templates/cs/alert_dgl.tmpl) | reusable | Single-button alert using `header`, `text`. |
| [room_info_dgl.tmpl](../templates/cs/room_info_dgl.tmpl) | `open&room_info_dgl` | Room details (title, host, players, level, password status). |
| [started_room_info.tmpl](../templates/cs/started_room_info.tmpl) | `go&room_info_dgl` for started rooms | In-game stats with pagination. |
| [started_room_info/statcols.tmpl](../templates/cs/started_room_info/statcols.tmpl) | partial (`part=statcols`) | Resource columns for the stats tab. |
| [user_details.tmpl](../templates/cs/user_details.tmpl) | `open&user_details` | Player card: nick, account, connect time, current room, Join/Info buttons. |
| [gg_cup_thanks_dgl.tmpl](../templates/cs/gg_cup_thanks_dgl.tmpl) | `open&gg_cup_thanks` | Sponsors / supporters dialog with dynamic height. |

### 7.2 American Conquest ([templates/ac/](../templates/ac/))

The AC family currently mirrors the CS subset minus the started-room and user-details views, plus an AC-only battle-type field on `new_room_dgl`:

| Template | Notes vs CS |
|----------|-------------|
| [enter.tmpl](../templates/ac/enter.tmpl) | Smaller layout; commented `LW_cfile` cookie path retained for reference. |
| [ok_enter.tmpl](../templates/ac/ok_enter.tmpl) | AC variant of `ok_enter`. |
| [error_enter.tmpl](../templates/ac/error_enter.tmpl) | AC variant of `error_enter`. |
| [startup.tmpl](../templates/ac/startup.tmpl) | AC lobby; uses `B_C` / `B_J` startup buttons and AC font sizes (`B1F40`, `YF16`). |
| [new_room_dgl.tmpl](../templates/ac/new_room_dgl.tmpl) | Adds `GV_VE_TYPE` / `L_TYPE`; passes `VE_NICK=<%GV_LCN_NICK>` on submit. |
| [reg_new_room.tmpl](../templates/ac/reg_new_room.tmpl) | Omits `CG_HOLE*` — AC does not use hole punching. |
| [join_room.tmpl](../templates/ac/join_room.tmpl) | Omits `CG_PORT`. |
| [confirm_password_dgl.tmpl](../templates/ac/confirm_password_dgl.tmpl) | AC variant. |
| [confirm_dgl.tmpl](../templates/ac/confirm_dgl.tmpl) | AC variant. |
| [alert_dgl.tmpl](../templates/ac/alert_dgl.tmpl) | AC variant. |

When you change a screen that exists in both families, edit **both** files unless you have a deliberate reason to diverge.

---

## 8. Authoring and testing

### 8.1 Adding a new template

1. **Create** `templates/cs/<name>.tmpl` (and `templates/ac/<name>.tmpl` if the screen exists in AC).
2. **Render it** from a controller handler:
   ```go
   body := c.render(req.Ver, "<name>.tmpl", map[string]string{
       "header": "…",
       "text":   "…",
       // every key the template references
   })
   ```
   Use a builder in `internal/render/` if a similar dialog already has one; otherwise keep the map flat.
3. **Pass every key the template reads.** Missing keys resolve to `""` and silently drop UI elements — easy to miss until a test fails.
4. **Register the template in the golden suite** (see below).
5. **Run the tests** and refresh goldens once the output looks right.
6. **Verify in the real client** when layout matters; goldens only lock text.

### 8.2 Golden tests

The wire test [internal/app/gsc/share_template_fullbody_test.go](../internal/app/gsc/share_template_fullbody_test.go) does two things:

- Asserts that **every** `templates/{cs,ac}/**/*.tmpl` is listed in `shareGoldenTemplateRels`.
- Renders each entry with `varsForShareTemplateGolden` and compares against `internal/app/gsc/testdata/template_fullbody/*.golden`.

To update goldens after an intentional change:

```bash
go test ./internal/app/gsc -golden -run TestShareTemplatesFullbodyGolden
```

If the default `vars` make your template pick the wrong branch, override per-template inside `varsForShareTemplateGolden` (e.g. force `logged_in=""` for the anonymous enter branch, `room_started="true"` for in-game stats, `gg_cup.started="0"` for marketing copy).

### 8.3 Renderer unit tests

Add or extend tests in [internal/render/](../internal/render/) when you touch the language semantics:

- [templates_renderer_test.go](../internal/render/templates_renderer_test.go) — end-to-end rendering and search-path behaviour.
- [templates_path_resolution_test.go](../internal/render/templates_path_resolution_test.go) — root precedence, name normalization.
- [templates_startup_expr_test.go](../internal/render/templates_startup_expr_test.go) — expression / arithmetic semantics.
- [time_interval_test.go](../internal/render/time_interval_test.go) — duration formatting helper.

### 8.4 Bypasses

Some screens build the show body in Go but keep a `.tmpl` on disk for drift detection (e.g. `buildUserDetailsBody` in [controller.go](../internal/app/gsc/controller.go) vs [user_details.tmpl](../templates/cs/user_details.tmpl)). When you change either side, change the other so the golden render via `loadShowBody` stays representative.

### 8.5 Authoring guidelines

- **Match neighbours.** Reuse `#font` triples, `%B_*` button ids, `L_*`/`T_*` row ids, and `GW|` shapes from the closest existing dialog.
- **Keep ASCII** unless you have a documented reason to do otherwise. Templates are protocol data.
- **Use `//`** for line comments — the renderer leaves them in the output, but the client ignores them.
- **Server logic stays in Go.** If a template needs branching beyond a couple of `IF` blocks, compute the value in Go and pass it flat.
- **CS and AC drift** — keep them in sync unless a difference is intentional and documented (e.g. AC battle-type field, hole-punch absence).

---

## 9. Cheat sheet

```text
Server-side TT (Go renderer)
  <? IF cond ?> … <? ELSE ?> … <? END ?>     # whole-line OR inline
  <? expr ?>                                  # interpolation
  expr → ternary | + | * | _ | .length | POSIX.floor() | var
  cond → || | && | ! | == | != | <,>,<=,>=  | bare-truthy
  Truthy: non-empty AND not "0" AND not "false"
  Special: h.req.ver, server.config.X, P.X
  NOT supported: FOREACH, USE, filters, '-', '/', methods, ${dynamic}

Client-side (passed through)
  <%NAME>            inside command strings
  {%NAME}            widget binding target
  %BOX[…], %L_…, %T_…, %B_…, %E_…, %P_…   layout ids
  GV_*  client globals (editable fields)
  CG_*  game-launch globals (CGAME / JGAME)
  GW|verb&arg|verb&arg          command pipelines
    open&page.dcml&K=V^K=V       open a dialog
    go&method&K=V                submit / dispatch
    url&http://…                 external URL
  LW_*  inside show body         lockbox, lockall, gvar, key, file, cfile, enb
  LW_*  on the wire              show, echo, time, dtbl, tbl
  Escapes: \26 = &, \7C = |, \5C = \, \00 = NUL

Files & code
  templates/cs/, templates/ac/                    on-disk bodies
  internal/render/templates.go                    renderer + lookup
  internal/render/{room_info,room_vars,…}.go      vars builders
  internal/app/gsc/controller*.go                 dispatch and call sites
  internal/app/gsc/share_template_fullbody_test.go  golden tests
  internal/app/gsc/testdata/template_fullbody/      golden bodies

Update goldens
  go test ./internal/app/gsc -golden -run TestShareTemplatesFullbodyGolden
```

### Further reading

- Renderer behaviour: `RenderShowTemplate`, `evalExpr`, `evalExprLeaf`, `evalCondition`, `lookupVar`, `renderInlineIfBlocks` in [internal/render/templates.go](../internal/render/templates.go).
- Search path & resolution: `LoadShowBodyFromRoots`, `BuildTemplateRoots`, `NormalizeShowTemplateName`, `IsAC`, `FallbackShowBody` in the same file.
- Vars builders: [internal/render/](../internal/render/) — `BuildRoomInfoVars`, `BuildStartedRoomInfoVars`, `BuildRegNewRoomVars`, `BuildJoinRoomVars`, `UserDetailsBody`, `GGCupThanksBody`, `MergeRoomDottedVars`, `TimeIntervalFromElapsedSec`, `CMLSafe`.
- GSC dispatch: `HandleWithMeta`, `dispatchOpen`, `handleGo`, `parseOpenParams`, `renderEnter` in [internal/app/gsc/controller.go](../internal/app/gsc/controller.go).
- Wire encoding: `CommandFromString`, `CommandSetFromString` in [internal/transport/gsc/](../internal/transport/gsc/).
- STUN / hole-punch debugging via show bodies: `extractGVar` in [internal/transport/tcp/server.go](../internal/transport/tcp/server.go).
