# CONTAINER.md — Docker assets

This document describes the container build and Compose stack for the
Cossacks Game Server. The runtime configuration is documented in
[config/simple-cossacks-server.yaml](config/simple-cossacks-server.yaml).
The package layout is documented in [internal/README.md](internal/README.md).

## Stack

`docker-compose.yml` runs two services:

| Service | Image source | Ports | Purpose |
|---------|--------------|-------|---------|
| `cossacks` | `tools/Dockerfile.cossacks` | `34001/tcp` (GSC), `3708/udp` (STUN), `9100/tcp` (metrics + probes) | The Go game server. |
| `irc`      | `tools/Dockerfile.ergo`     | `6667/tcp` | Ergo IRC server for the in-game chat. |

The Go server uses an in-process key–value store with TTL
(`internal/adapter/kvmemory`) for hole-punch / STUN data exchange.
There is no external broker, no separate STUN container.

## Building and running

```bash
docker compose up -d --build       # build images and start the stack
docker compose down                # stop and remove containers
docker compose up                  # foreground; Ctrl+C to stop
docker compose build --no-cache    # rebuild from scratch
```

## Environment

`docker-compose.yml` reads `.env` at the repository root. Useful keys:

| Variable | Effect |
|----------|--------|
| `HOST_NAME` | Public hostname surfaced to clients as the IRC `chat_server` and the STUN advertisement target. |
| `UDP_KEEP_ALIVE_INTERVAL` | STUN keep-alive TTL; also propagated to the GSC `hole_int` config. |
| `METRICS_ADDR` | e.g. `:9100`. When set, the server publishes `/metrics`, `/livez`, and `/readyz` on this address. |
| `PROBE_ADDR` | Optional. If set, `/livez` and `/readyz` listen here instead of on `METRICS_ADDR`. |
| `LOG_FORMAT` | `user` (console) or `json`. Overrides the YAML config. |
| `LOG_FILE` | Log file path with rotation. Overrides the YAML config. |

## cossacks service

- Multi-stage build; the runtime image includes `wget` for the
  health check.
- Entrypoint: `/app/cossacksd --config /app/config/simple-cossacks-server.yaml`.
- Volumes: `./logs:/app/logs`, `./templates:/app/templates`.
- Health check: `wget -qO- http://127.0.0.1:9100/readyz`.

## irc service

- Built from `tools/Dockerfile.ergo`.
- Listens on `6667/tcp`.
- The entrypoint script is idempotent for TLS certificate generation
  across restarts.

## Listening sockets summary

| Port | Protocol | Origin |
|------|----------|--------|
| `34001` | TCP | GSC application (`internal/transport/tcp`). |
| `3708`  | UDP | STUN responder (`internal/adapter/stun`). |
| `9100`  | TCP | Metrics + probes (`internal/platform/metrics`, `internal/platform/health`). |
| `6667`  | TCP | Ergo IRC. |

If `host` in the YAML config is `localhost`, the Go process rewrites
the listen address to `0.0.0.0` so Docker port publishing works as
expected.
