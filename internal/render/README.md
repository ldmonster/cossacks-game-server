# internal/render — template renderer

The `LW_show` template renderer used by every GSC route that sends a
dialog or page back to the client. The renderer owns:

- A loader that reads `templates/{ac,cs}/*.tmpl` once at startup.
- Variable expansion (`%FOO&...&%` and dotted vars) with the same
  precedence the historical reference server uses.
- Time-interval and player-column helpers that several routes share.

The renderer is stateless across requests; routes pass it the variable
map and receive the rendered body. It is bound to a templates root
directory via `NewTemplateRenderer(dir)`.

## Why it is a top-level package

The renderer is shared between the GSC application
(`internal/app/gsc/...`) and the wire test suite, but does not
itself fit the application/adapter/transport split — it is a small
pure-render concern that depends only on stdlib and on the rendering
template files. Keeping it at the top of `internal/` avoids miscoding
it as a domain service.
