# WingStation

A read-only Docker container dashboard for homelabbers. WingStation connects to your Docker daemon and presents an organized, label-driven web dashboard — no database, no state, just a single binary.

## Features

- **Read-only** — Only reads from Docker, never mutates containers
- **Label-driven organization** — Group, name, and customize containers via `wingstation.*` labels
- **Live updates** — Server-Sent Events push changes in real time
- **Container detail panel** — Inspect networking, volumes, security, hardware, and environment
- **Search & filter** — Filter by status, group, tags, or free-text search
- **Single binary** — Templates and static assets embedded; deploy one file
- **Tiny image** — Built on distroless, multi-arch (amd64 + arm64)
- **Optional basic auth** — Protect your dashboard with username/password

## Quick Start

```bash
docker run -d \
  --name wingstation \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --read-only \
  --security-opt no-new-privileges:true \
  ghcr.io/rbretschneider/wingstation:latest
```

Then visit [http://localhost:8080](http://localhost:8080).

### Docker Compose

```yaml
services:
  wingstation:
    image: ghcr.io/rbretschneider/wingstation:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    read_only: true
    security_opt:
      - no-new-privileges:true
```

### Examples

- **[`docker-compose.yml`](docker-compose.yml)** — Default compose file for WingStation itself
- **[`examples/docker-compose.minimal.yml`](examples/docker-compose.minimal.yml)** — Bare minimum, no labels needed
- **[`examples/docker-compose.homelab.yml`](examples/docker-compose.homelab.yml)** — Full homelab stack (Plex, Sonarr, Radarr, Pi-hole, Portainer, Home Assistant, etc.) showing all label features

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `WINGSTATION_PORT` | `8080` | HTTP listen port |
| `WINGSTATION_HOST` | `0.0.0.0` | HTTP listen host |
| `WINGSTATION_BASE_URL` | *(empty)* | Base URL prefix for reverse proxy setups |
| `WINGSTATION_DOCKER_SOCKET` | `/var/run/docker.sock` | Path to Docker socket |
| `WINGSTATION_AUTH_ENABLED` | `false` | Enable basic authentication |
| `WINGSTATION_AUTH_USER` | *(empty)* | Basic auth username |
| `WINGSTATION_AUTH_PASS` | *(empty)* | Basic auth password |
| `WINGSTATION_CACHE_TTL` | `5s` | Cache time-to-live (Go duration format) |
| `WINGSTATION_SSE_ENABLED` | `true` | Enable Server-Sent Events live updates |
| `WINGSTATION_SSE_RETRY_MS` | `3000` | SSE client reconnect interval (ms) |
| `WINGSTATION_STATS_INTERVAL_MS` | `5000` | Stats refresh interval (ms) |
| `WINGSTATION_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

## Container Labels

Customize how containers appear on the dashboard using `wingstation.*` labels. See [LABELS.md](LABELS.md) for the full schema.

```yaml
labels:
  - "wingstation.group=Media"
  - "wingstation.name=Plex Media Server"
  - "wingstation.icon=🎬"
  - "wingstation.description=Streaming media server"
  - "wingstation.url=http://plex.local:32400"
  - "wingstation.priority=10"
  - "wingstation.tags=media,streaming"
```

## API

WingStation exposes a JSON API alongside the HTML dashboard:

| Endpoint | Description |
|---|---|
| `GET /api/containers` | List all containers (supports `?q=`, `?status=`, `?group=`, `?tag=`, `?sort=`) |
| `GET /api/containers/{id}` | Full container detail |
| `GET /api/host` | Host and daemon information |
| `GET /healthz` | Health check (pings Docker daemon) |

## Development

```bash
# Run locally
make dev

# Build binary
make build

# Run tests
make test

# Lint
make lint

# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Requirements

- Go 1.23+
- Docker (for development testing)
- golangci-lint (for linting)

## Architecture

- **Safety boundary**: A `ReadOnlyClient` interface is the only Docker abstraction. No mutation methods are exposed.
- **Single binary**: All templates and static assets are embedded via `//go:embed`.
- **Hypermedia-driven**: The server returns HTML fragments for htmx. JSON API is secondary.
- **Zero external deps** beyond the Docker SDK: stdlib `net/http`, `html/template`, `embed`, `log/slog`.

## License

MIT — see [LICENSE](LICENSE).
