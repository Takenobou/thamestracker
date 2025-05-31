# ThamesTracker

ThamesTracker tracks Tower Bridge lift events and vessel movements on the River Thames. It exposes a JSON API, iCalendar feeds, and provides CLI tools.

## Features
- JSON API for bridge lifts, vessel movements, and location stats
- iCalendar feeds for bridge lifts and vessel events
- Query filtering (name, type/category, location, after, before, unique, etc.)
- Health check endpoint
- Prometheus metrics endpoint (enabled with `METRICS_PUBLIC=true`)
- OpenAPI spec served at `/docs`
- Per-IP rate limiting (configurable)
- Circuit breaker for external API calls
- Redis (or in-memory fallback) caching
- CLI for scraping and fetching data/feeds

## Quickstart
### Local development
```bash
# Clone and install deps
git clone https://github.com/Takenobou/thamestracker.git
cd thamestracker
go mod download

# Run server (defaults to port 8080)
go run ./cmd/server/main.go
```

### Docker Compose
```yaml
version: '3.8'
services:
  thamestracker:
    image: ghcr.io/takenobou/thamestracker:latest
    ports:
      - "8080:8080"
    environment:
      PORT: 8080
      PORT_OF_LONDON: https://pla.co.uk/api-proxy/api?_api_proxy_uri=/ships/lists
      TOWER_BRIDGE: https://www.towerbridge.org.uk/lift-times
      REDIS_ADDRESS: redis://redis:6379
      CB_MAX_FAILURES: 5
      CB_COOL_OFF: 60
      CACHE_MAX_ENTRIES: 1000
      CACHE_TTL_SECONDS: 3600
      REQUESTS_PER_MIN: 60
      METRICS_PUBLIC: false
    depends_on:
      - redis

  redis:
    image: redis:latest
    ports:
      - "6379:6379"
```

## Configuration
All settings via environment variables. Defaults shown:

| Variable                   | Default                                                         | Description                                    |
|----------------------------|-----------------------------------------------------------------|------------------------------------------------|
| `PORT`                     | `8080`                                                          | HTTP port for server                           |
| `PORT_OF_LONDON`           | `https://pla.co.uk/api-proxy/api?_api_proxy_uri=/ships/lists`   | Base URL for Port of London ship API           |
| `TOWER_BRIDGE`             | `https://www.towerbridge.org.uk/lift-times`                     | URL for Tower Bridge lift times page           |
| `REDIS_ADDRESS`            | `localhost:6379`                                                | Redis connection address                       |
| `CB_MAX_FAILURES`          | `5`                                                             | Circuit-breaker max consecutive failures       |
| `CB_COOL_OFF`              | `60`                                                            | Circuit-breaker open timeout (sec)             |
| `CACHE_MAX_ENTRIES`        | `1000`                                                          | Max entries in in-memory fallback cache        |
| `CACHE_TTL_SECONDS`        | `3600`                                                          | TTL for in-memory fallback cache (sec)         |
| `REQUESTS_PER_MIN`         | `60`                                                            | Per-IP rate-limit (requests per minute)        |
| `METRICS_PUBLIC`           | `false`                                                         | Expose /metrics endpoint if true               |
| `BRIDGE_FILTER_PERCENTILE` | `0.10`                                                          | Percentile threshold for filtering most frequent bridge lifts when unique=true |
| `BRIDGE_FILTER_MAX_COUNT`  | `8`                                                             | Max times a vessel can appear in bridge lifts when unique=true |

## API Reference

### GET /bridge-lifts
Returns upcoming Tower Bridge lift events in JSON.

**Query parameters**:
- `unique` (boolean, default `false`): remove duplicate lifts by vessel name
- `name` (string, optional): filter lifts by vessel name substring
- `after` (RFC3339, optional): only events after this timestamp
- `before` (RFC3339, optional): only events before this timestamp
- `location` (string, optional): filter by location

**Response Example**:
```json
[
  {
    "timestamp": "2025-04-05T17:45:00Z",
    "vessel_name": "Paddle Steamer Dixie Queen",
    "category": "bridge",
    "direction": "Up river",
    "location": "Tower Bridge Road, London"
  }
]
```

**Example**:
```bash
curl -s "http://localhost:8080/bridge-lifts?unique=true&name=queen" | jq .
```

### GET /vessels
Returns vessel movements in JSON.

**Query parameters**:
| Name         | Type    | Default | Description                              |
|--------------|---------|---------|------------------------------------------|
| `type`       | string  | `all`   | one of `inport`, `arrivals`, `departures`, `forecast`, or `all` |
| `name`       | string  | —       | filter by vessel name                    |
| `location`   | string  | —       | filter by port/all location fields       |
| `nationality`| string  | —       | filter by vessel nationality             |
| `after`      | RFC3339 | —       | include events after this timestamp      |
| `before`     | RFC3339 | —       | include events before this timestamp     |
| `unique`     | boolean | `false` | remove duplicate vessel names            |

**Response Example**:
```json
[
  {
    "timestamp": "2025-01-25T20:33:47Z",
    "vessel_name": "SILVER STURGEON",
    "category": "inport",
    "voyage_number": "S7670",
    "location": "WOODS QUAY"
  }
]
```

**Example**:
```bash
curl -s "http://localhost:8080/vessels?type=arrivals&unique=true&after=2025-04-01T00:00:00Z" | jq .
```

### GET /bridge-lifts/calendar.ics
Returns an iCalendar feed for Tower Bridge lift events.

**Query parameters**:
- `unique` (boolean, default `false`): remove duplicate lifts by vessel name
- `name` (string, optional): filter lifts by vessel name substring
- `after` (RFC3339, optional): only events after this timestamp
- `before` (RFC3339, optional): only events before this timestamp
- `location` (string, optional): filter by location

**Example**:
```bash
curl -s "http://localhost:8080/bridge-lifts/calendar.ics?unique=true&after=2025-04-01T00:00:00Z" > bridge.ics
```

### GET /vessels/calendar.ics
Returns an iCalendar feed for vessel movements.

**Query parameters**:
- `type` (string, default `all`): one of `inport`, `arrivals`, `departures`, `forecast`, or `all`
- `name`, `location`, `nationality`, `after`, `before`, and `unique` (same as `/vessels`)

**Example**:
```bash
curl -s "http://localhost:8080/vessels/calendar.ics?type=arrivals&unique=true&after=2025-04-01T00:00:00Z" > vessels.ics
```

### GET /docs
Serves the OpenAPI JSON specification for the API.

**Example**:
```bash
curl -s "http://localhost:8080/docs" | jq .
```

### GET /metrics
Prometheus metrics endpoint, enabled only when the environment variable `METRICS_PUBLIC=true` is set.

**Response**: Prometheus metrics text

### GET /healthz
Liveness probe. Returns HTTP 200 if Redis and external API are healthy, HTTP 503 otherwise.

**Response**:
```json
{ "status": "ok" }
```
or
```json
{ "status": "fail", "error": "health check error" }
```

### GET /locations
Returns aggregated vessel counts per location.

**Query parameters**:
- `minTotal` (integer, default `0`): include only locations with `total` >= `minTotal`
- `q` (string, optional): case-insensitive substring filter on the location `name`

**Response Example**:
```json
[
  {
    "name": "PortA",
    "code": "",
    "inport": 1,
    "arrivals": 2,
    "departures": 3,
    "forecast": 0,
    "total": 6
  }
]
```

**Example**:
```bash
curl -s "http://localhost:8080/locations?minTotal=5&q=port" | jq .
```

## Error Handling
- 400 Bad Request: Invalid query parameters
- 503 Service Unavailable: Circuit breaker open or dependency unavailable
- 500 Internal Server Error: Unexpected errors

## Rate Limiting
Requests are rate-limited per IP (default: 60/minute, configurable via `REQUESTS_PER_MIN`).

## Caching
- Redis is used for caching if configured, otherwise an in-memory fallback cache is used.
- Circuit breaker protects external API calls.

## CLI Reference
The CLI replicates the service layer and fetches data from the APIs:
```bash
# Bridge lifts (JSON)
thamestracker bridge-lifts

# Vessels in port (JSON)
thamestracker vessels

# Vessel arrivals / departures / forecast
thamestracker arrivals
thamestracker departures
thamestracker forecast

# iCalendar feeds
thamestracker bridge-ics > bridge.ics
thamestracker vessels-ics > vessels.ics
```

## Bridge filter tuning
When using `unique=true` on bridge lift endpoints, the service will filter out vessels that are in the top `BRIDGE_FILTER_PERCENTILE` most frequent lifts, or that appear more than `BRIDGE_FILTER_MAX_COUNT` times. These thresholds can be tuned at runtime via environment variables, no code changes required.

## License
MIT
