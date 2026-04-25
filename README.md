# Cossacks Game Server

Dockerized multiplayer backend for Cossacks.

The stack in `docker-compose.yml` runs:
- `cossacks` (Go game server, TCP `34001`)
- `stun` (Go UDP hole-punch helper, UDP `3708`)
- `irc` (Ergo IRC server, TCP `6667`)
- `redis` (state exchange, bound to `127.0.0.1:6379`)

## Repository Layout

- `cmd/` - service entry points (`cossacksd`, `stund`)
- `internal/` - server/config/protocol implementation
- `templates/` - LW show templates (`cs/`, `ac/`, `.tmpl`)
- `config/` - runtime config (`simple-cossacks-server.yaml`)
- `tools/` - Dockerfiles and helper scripts (`ergo-entrypoint.sh`)
- `docker-compose.yml` - runtime stack

## Configuration

Create `.env` in repository root:

```env
HOST_NAME=YOUR_PUBLIC_HOST_OR_IP
UDP_KEEP_ALIVE_INTERVAL=300
```

- `HOST_NAME` is used as IRC host/name and hole-punch host hint in game flows.
- `UDP_KEEP_ALIVE_INTERVAL` controls STUN keep-alive TTL and hole interval.

Go server config is YAML at `config/simple-cossacks-server.yaml` and keeps Perl-compatible keys (`host`, `port`, `hole_port`, `hole_int`, `templates`, etc.).

## Run

Start all services:

```bash
docker compose up -d --build
```

Stop and remove containers:

```bash
docker compose down
```

Run in foreground (for logs), stop with `Ctrl+C`:

```bash
docker compose up
```

If images/scripts changed, rebuild:

```bash
docker compose build --no-cache
docker compose up
```

## Exposed Ports

- `34001/tcp` - game server (`cossacks`)
- `3708/udp` - STUN (`stun`)
- `6667/tcp` - IRC (`irc`)

## Service Notes

### cossacks (Go)

- Built from `tools/Dockerfile.cossacks`
- Entrypoint: `/app/cossacksd -config /app/config/simple-cossacks-server.yaml`
- Mounts:
  - `./logs:/app/logs`
  - `./templates:/app/templates`
- Env overrides:
  - `HOST_NAME` -> `chat_server`
  - `UDP_KEEP_ALIVE_INTERVAL` -> `hole_int`

### stun (Go)

- Built from `tools/Dockerfile.stun`
- Listens on UDP `:3708`
- Writes NAT endpoint data into Redis with TTL (`keep_alive * 1.5`)

### irc (Ergo)

- Built from `tools/Dockerfile.ergo`
- Listens on `6667`
- Entrypoint script is idempotent for TLS cert generation across restarts.

### redis

- Default image, bound to `127.0.0.1:6379`
- Used by `cossacks` and `stun` for endpoint exchange

## Client: Custom Server (Non-Steam)

These steps apply to the classic GSC client.

1. Open `Internet\ggwdc.ini`.
2. Replace `ggwdserver_addr gms.2gw.net` with your host/IP.
3. Keep `ggwdserver_port 34001`.
4. Save file and start the game.

## Quick Start

```bash
git clone https://github.com/ldmonster/cossacks-game-server
cd cossacks-game-server
cat > .env <<'EOF'
HOST_NAME=YOUR_PUBLIC_HOST_OR_IP
UDP_KEEP_ALIVE_INTERVAL=300
EOF
docker compose up -d --build
```

## Copyright and License

This project is licensed under the Apache License 2.0.
See `LICENSE` for the full text.
