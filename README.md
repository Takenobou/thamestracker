# ThamesTracker

**ThamesTracker** is a tool for tracking ship movements along the Thames. It provides both a scraper for live data and a web API to access the information.

## Features

- **Bridge Lifts:** Scrape upcoming Tower Bridge lift events.
- **Vessel Movements:** Scrape and display vessel information (in port, arrivals, departures, forecast).
- **Calendar Feed:** Generate an iCalendar feed for bridge lifts and vessel movements.
- **Caching:** Utilizes Redis for caching API responses.
- **Configuration:** Minimal configuration via environment variables with sensible defaults.

## Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/Takenobou/thamestracker.git
   cd thamestracker
   ```

2. **Set up your environment variables:**

   Create a `.env` file in the root directory to override defaults:

   ```env
   PORT=8080
   REDIS_ADDRESS=localhost:6379
   ```

3. **Install dependencies:**

   ```bash
   go mod download
   ```

4. **Build the project:**

   ```bash
   go build ./...
   ```

## Running the Application

### As a Server

To run the web API server:

```bash
go run ./cmd/server/main.go
```

The server will start on the port defined in your environment (default: `8080`).

### Command Line Tools

The project also provides command line utilities for scraping data:

- **Bridge Lifts:**

  ```bash
    go run ./cmd/scraper/main.go bridge-lifts
  ```

- **Vessels (Inport):**

  ```bash
    go run ./cmd/scraper/main.go vessels
  ```

- **Vessel Arrivals:**

  ```bash
    go run ./cmd/scraper/main.go arrivals
  ```

- **Vessel Departures:**

  ```bash
    go run ./cmd/scraper/main.go departures
  ```

- **Vessel Forecast:**

  ```bash
    go run ./cmd/scraper/main.go forecast
  ```

## Deployment with Docker

You can deploy ThamesTracker using Docker Compose. The `docker-compose.yml` file is included in the repository and defines both services.

### Example `docker-compose.yml`:

```yaml
services:
  thamestracker:
    container_name: thamestracker
    image: ghcr.io/takenobou/thamestracker:latest
    restart: always
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - REDIS_ADDRESS=redis:6379
    depends_on:
      - redis

  redis:
    container_name: thamestracker-redis
    image: redis:latest
    restart: always
    ports:
      - "6379:6379"
```

### Running the services

1. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```

2. Visit `http://localhost:8080` to access ThamesTracker.

## API Endpoints

Once the server is running, the following endpoints are available:

- **GET /bridge-lifts**  
  Retrieves upcoming Tower Bridge lift events in JSON format.
  
  **Query Parameters:**
  - `unique` (boolean, default: false): If set to `true`, duplicate events are filtered out.

- **GET /vessels**  
  Retrieves vessel movements in JSON format.
  
  **Query Parameters:**
  - `type` (string, default: "all"): Filter by vessel type. Valid values include: `inport`, `arrivals`, `departures`, `forecast`, or `all`.
  - `name` (string, optional): Filter by vessel name.
  - `location` (string, optional): Filter by vessel location.
  - `nationality` (string, optional): Filter by vessel nationality.

- **GET /calendar.ics**  
  Generates an iCalendar feed for bridge lifts and vessel events. The calendar feed adheres to the iCalendar format and can be imported into popular calendar applications as a subscription and new events will be added automatically.
  
  **Query Parameters:**
  - `eventType` (string, default: "all"): Determines which events to include. Valid values are:
    - "all": Include both bridge lifts and vessel events.
    - "bridge": Include only bridge lift events.
    - "vessel": Include only vessel events.
  - `unique` (boolean, default: false): If set to `true`, duplicate bridge lift events are filtered out.
  - `name` (string, optional): If provided, filters events by vessel name.
  - `location` (string, optional): If provided, filters events by vessel location.
