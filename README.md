# ThamesTracker

ThamesTracker tracks Tower Bridge lift events and vessel movements on the River Thames. It exposes a JSON API, an iCalendar feed, and provides CLI tools.

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
      TIMEZONE: Europe/London
    depends_on:
      - redis

  redis:
    image: redis:latest
    ports:
      - "6379:6379"
```

## Configuration
All settings via environment variables. Defaults shown:

| Variable           | Default                                                         | Description                                    |
|--------------------|-----------------------------------------------------------------|------------------------------------------------|
| `PORT`             | `8080`                                                          | HTTP port for server                           |
| `PORT_OF_LONDON`   | `https://pla.co.uk/api-proxy/api?_api_proxy_uri=/ships/lists`   | Base URL for Port of London ship API           |
| `TOWER_BRIDGE`     | `https://www.towerbridge.org.uk/lift-times`                     | URL for Tower Bridge lift times page           |
| `REDIS_ADDRESS`    | `localhost:6379`                                                | Redis connection address                       |
| `CB_MAX_FAILURES`  | `5`                                                             | Circuit‑breaker max consecutive failures       |
| `CB_COOL_OFF`      | `60`                                                            | Circuit‑breaker open timeout (sec)             |
| `CACHE_MAX_ENTRIES`| `1000`                                                          | Max entries in in‑memory fallback cache        |
| `CACHE_TTL_SECONDS`| `3600`                                                          | TTL for in‑memory fallback cache (sec)         |
| `REQUESTS_PER_MIN` | `60`                                                            | Per‑IP rate‑limit (requests per minute)        |
| `TIMEZONE`         | `Europe/London`                                                 | Timezone for timestamp conversion (IANA TZ)   |

## API Reference

### GET /bridge-lifts
Returns upcoming Tower Bridge lift events in JSON.

**Query parameters**:
- `unique` (boolean, default `false`): remove duplicate lifts by vessel name.
- `name` (string, optional): filter lifts by vessel name substring.

**Response**:
```json
[{
  "date": "2025-04-05",
  "time": "17:45",
  "vessel": "HMS Example",
  "direction": "Up river"
}]
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

**Response**:
```json
[{
  "time": "20:33",
  "date": "25/01/2025",
  "location_name": "Port Example",
  "name": "Example Vessel",
  "voyage_number": "E1234",
  "type": "inport"
}]
```

**Example**:
```bash
curl -s "http://localhost:8080/vessels?type=arrivals&unique=true&after=2025-04-01T00:00:00Z" | jq .
```

### GET /calendar.ics
Returns an iCalendar feed combining bridge lifts and vessel events.

**Query parameters**:
- `eventType` (string, default `all`): choose `all`, `bridge`, or `vessel` events.
- `from` (YYYY-MM-DD, optional): include only events starting on or after this date.
- `to` (YYYY-MM-DD, optional): include only events starting on or before this date.
- All `/vessels` and `/bridge-lifts` filters apply when `eventType` includes that type.

**Example**:
```bash
curl -s "http://localhost:8080/calendar.ics?type=departures&unique=true&from=2025-04-01&to=2025-04-07" > feed.ics
```

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

### GET /metrics
Prometheus scrape endpoint. Exposes metrics:
- `thamestracker_external_api_scrapes_total{api="bridge|vessels"}`
- `thamestracker_external_api_duration_seconds{api="bridge|vessels"}`
- `thamestracker_cache_hits_total`
- `thamestracker_cache_misses_total`

### GET /locations
Returns aggregated vessel counts per location.

**Query parameters**:
- `minTotal` (integer, default `0`): include only locations with `total` >= `minTotal`.
- `q` (string, optional): case-insensitive substring filter on the location `name`.

**Response**:
```json
[
  {"name":"PortA","code":"","inport":1,"arrivals":2,"departures":3,"forecast":0,"total":6},
  {"name":"PortB","code":"","inport":0,"arrivals":1,"departures":0,"forecast":0,"total":1}
]
```

**Example**:
```bash
curl -s "http://localhost:8080/locations?minTotal=5&q=port" | jq .
```

## CLI Reference
Built‑in CLI replicates service layer:
```bash
# Bridge lifts (JSON)
thamestracker bridge-lifts

# Vessels in port (JSON)
thamestracker vessels

# Vessel arrivals / departures / forecast
thamestracker arrivals
thamestracker departures
thamestracker forecast

# iCalendar feed
thamestracker ics > feed.ics
```

## License
MIT
