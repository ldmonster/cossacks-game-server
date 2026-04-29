# Cossacks Game Server

Multiplayer backend for the GSC family of titles (Cossacks: European
Wars / The Art of War / Back to War; American Conquest).

The server exposes a TCP listener for the GSC protocol, a UDP STUN
responder for hole-punching, and an HTTP endpoint for metrics and
health probes. An IRC service (Ergo) for the in-game chat is bundled
in the Compose stack.

## Documentation map

| Topic | Link |
|-------|------|
| Container build & Docker Compose | [CONTAINER.md](CONTAINER.md) |
| Internal package layout          | [internal/README.md](internal/README.md) |
| GSC application                  | [internal/app/gsc/README.md](internal/app/gsc/README.md) |
| Templates                        | [templates/README.md](templates/README.md) |
| Template pipeline notes          | [docs/TEMPLATES.md](docs/TEMPLATES.md) |
| Runtime configuration            | [config/simple-cossacks-server.yaml](config/simple-cossacks-server.yaml) |

## Connecting a non-Steam client

These steps apply to the classic GSC client.

1. Open `Internet\ggwdc.ini`.
2. Replace `ggwdserver_addr gms.2gw.net` with your host or IP.
3. Keep `ggwdserver_port 34001`.
4. Save and start the game.

## Configuration

Runtime configuration is loaded from a YAML file by
`internal/platform/config`. Every key is documented inline in
[config/simple-cossacks-server.yaml](config/simple-cossacks-server.yaml).

Environment variables that override or supplement the config:

| Variable | Effect |
|----------|--------|
| `HOST_NAME` | Public hostname surfaced as `chat_server` and the STUN advertisement target. |
| `UDP_KEEP_ALIVE_INTERVAL` | Propagated to the GSC `hole_int` config and the STUN keep-alive TTL. |
| `METRICS_ADDR` | Address for `/metrics`, `/livez`, `/readyz`. |
| `PROBE_ADDR` | Override address for `/livez` + `/readyz` only. |
| `LOG_FORMAT` | `user` (console) or `json`. |
| `LOG_FILE` | Log file path with rotation. |

## Local development

From the repository root:

```bash
go test -race -cover ./...
make test
make lint
```

`make lint` invokes the project's pinned `golangci-lint` build under
`bin/`; no global toolchain is required.

## Quick start

```bash
git clone https://github.com/ldmonster/cossacks-game-server
cd cossacks-game-server
cat > .env <<'EOF'
HOST_NAME=YOUR_PUBLIC_HOST_OR_IP
UDP_KEEP_ALIVE_INTERVAL=300
EOF
docker compose up -d --build
```

For everything else container-related see [CONTAINER.md](CONTAINER.md).

## Copyright and license

This project is licensed under the Apache License 2.0. See
[LICENSE](LICENSE) for the full text.
