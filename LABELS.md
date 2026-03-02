# WingStation Label Schema

WingStation uses Docker container labels in the `wingstation.*` namespace to customize how containers appear on the dashboard. All labels are optional.

## Labels

| Label | Type | Default | Description |
|---|---|---|---|
| `wingstation.group` | string | `Ungrouped` | Group name for organizing containers |
| `wingstation.name` | string | container name | Display name override |
| `wingstation.icon` | string | *(none)* | Emoji or icon for the container card |
| `wingstation.description` | string | *(none)* | Short description shown on the card |
| `wingstation.url` | string | *(none)* | Link to the service (shown as "Open" link) |
| `wingstation.priority` | int | `50` | Sort priority within group (lower = higher priority) |
| `wingstation.hide` | bool | `false` | Hide container from dashboard (`true`, `1`, or `yes`) |
| `wingstation.tags` | string | *(none)* | Comma-separated tags for filtering |

## Examples

### Docker Compose

```yaml
services:
  plex:
    image: linuxserver/plex
    labels:
      wingstation.group: "Media"
      wingstation.name: "Plex"
      wingstation.icon: "🎬"
      wingstation.description: "Media streaming server"
      wingstation.url: "http://plex.local:32400/web"
      wingstation.priority: "10"
      wingstation.tags: "media,streaming"

  sonarr:
    image: linuxserver/sonarr
    labels:
      wingstation.group: "Media"
      wingstation.name: "Sonarr"
      wingstation.icon: "📺"
      wingstation.description: "TV show management"
      wingstation.url: "http://sonarr.local:8989"
      wingstation.priority: "20"
      wingstation.tags: "media,automation"

  watchtower:
    image: containrrr/watchtower
    labels:
      wingstation.group: "Infrastructure"
      wingstation.name: "Watchtower"
      wingstation.icon: "🗼"
      wingstation.description: "Automatic container updates"
      wingstation.priority: "30"
      wingstation.tags: "infra,automation"

  temp-debug:
    image: alpine
    labels:
      wingstation.hide: "true"
```

### Docker CLI

```bash
docker run -d \
  --name nginx \
  --label wingstation.group="Web" \
  --label wingstation.name="Nginx Proxy" \
  --label wingstation.icon="🌐" \
  --label wingstation.url="http://localhost:80" \
  --label wingstation.priority="5" \
  nginx:latest
```

## Priority

Priority determines sort order within a group. Lower numbers appear first:

- `1-10`: Critical infrastructure
- `11-30`: Primary services
- `31-50`: Standard services (default)
- `51-80`: Background/utility
- `81-100`: Low priority

## Groups

Containers with the same `wingstation.group` value are displayed together. Containers without a group label appear under "Ungrouped". Groups are sorted by the lowest priority value among their containers.

## Tags

Tags are comma-separated and can be used to filter containers in the search bar. Example: `wingstation.tags=media,streaming,entertainment`.
