# football-sim

A backend simulation of a football league, built in Go for the Insider internship case study.

Generates a round-robin fixture, plays matches using a Poisson-based simulator, and predicts championship probabilities via Monte Carlo simulation of remaining fixtures.

**Tech stack:** Go 1.26+, PostgreSQL, pgx, golang-migrate, log/slog

## Quick Start

### Option A — Full stack via Docker

```bash
git clone https://github.com/Floha0/football-sim
cd football-sim
cp .env.example .env       # set DB_PASSWORD
docker compose up --build
```

The app and Postgres come up together. Stop with Ctrl+C; clean up data with `docker compose down -v`.

### Option B — Local Go binary against Dockerized Postgres

```bash
git clone https://github.com/Floha0/football-sim
cd football-sim
cp .env.example .env       # set DB_PASSWORD
make db-up                 # Postgres in Docker
make migrate-up            # apply schema
make run                   # Go app on host
```

Either way, the server listens on `http://localhost:8080`. See [API Endpoints](#api-endpoints) for available routes.

## Prerequisites

- **Go 1.26 or later** (matches `go.mod`)
- **Docker and Docker Compose** for PostgreSQL
- **`golang-migrate` CLI** for local migrations:
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

## Environment Variables

| Variable | Description | Default | Type |
| :--- | :--- | :--- | :--- |
| `PORT` | HTTP server port | `8080` | `string` |
| `SHUTDOWN_TIMEOUT` | Max wait for active requests during shutdown | `10s` | `duration` |
| `DB_HOST` | PostgreSQL host | `localhost` | `string` |
| `DB_PORT` | PostgreSQL port | `5432` | `string` |
| `DB_USER` | Database username | `insider` | `string` |
| `DB_PASSWORD` | Database password | **required** | `string` |
| `DB_NAME` | Database name | `insider_league` | `string` |
| `DB_SSLMODE` | SSL mode (`disable`, `require`, etc.) | `disable` | `string` |
| `SIM_HOME_ADVANTAGE` | Home team power multiplier | `1.15` | `float64` |
| `SIM_AVG_GOALS` | Expected total goals per match | `2.75` | `float64` |
| `SIM_MAX_GOALS` | Safety cap on goals per team | `10` | `int` |
| `PREDICTION_ITERATIONS` | Monte Carlo iteration count | `10000` | `int` |
| `LOG_LEVEL` | Log verbosity (`debug`, `info`, `warn`, `error`) | `info` | `string` |

## API Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| GET    | `/health`           | Liveness check |
| GET    | `/api/standings`    | Current league table (Premier League tiebreakers) |
| GET    | `/api/matches?week=N` | All matches for a given week |
| POST   | `/api/play-week`    | Play the next unplayed week |
| POST   | `/api/play-all`     | Play every remaining week in one call |
| GET    | `/api/predictions`  | Monte Carlo championship probabilities |
| POST   | `/api/reset`        | Wipe matches and regenerate fixtures |
| PUT    | `/api/matches/{id}` | Edit a match result |

### Example: full simulation flow

```bash
# Reset to a clean state
curl -X POST http://localhost:8080/api/reset

# Play 4 weeks
for i in 1 2 3 4; do
    curl -X POST http://localhost:8080/api/play-week
done

# View standings and predictions at week 4
curl http://localhost:8080/api/standings | jq
curl http://localhost:8080/api/predictions | jq

# Edit a match (use a real ID from /api/matches?week=N)
curl -X PUT http://localhost:8080/api/matches/5 \
    -H "Content-Type: application/json" \
    -d '{"homeGoals": 3, "awayGoals": 1}'

# Finish the season
curl -X POST http://localhost:8080/api/play-all | jq
```

## Postman Collection

A ready-to-import Postman collection is included as [`postman_collection.json`](./postman_collection.json).

1. Open Postman → File → Import → select `postman_collection.json`
2. In the collection's Variables tab, set `baseUrl` (default `http://localhost:8080`)
3. Send any request

Recommended order for a full flow: **Reset League → Play All → Get Standings → Get Predictions**.

## Architecture

```
cmd/server/             entrypoint, dependency wiring, graceful shutdown
internal/
  config/               env-based configuration with validation
  domain/               pure types (Team, Match, Standing, Prediction)
  league/               service orchestrating fixtures, simulation, standings
  prediction/           Monte Carlo championship predictor
  simulation/           Poisson-based match simulator
  postgres/             pgx-backed repository implementations + migrations
  handler/              HTTP transport: DTOs, routing, middleware, error mapping
migrations/             SQL migration files
```

Three principles:

1. **Interfaces are defined by consumers.** Repository and Predictor interfaces live in `internal/league`.
2. **Dependencies flow inward.** `handler` → `league` → `domain`, never the reverse.
3. **All wiring happens in `main.go`.** No DI framework, no reflection. The dependency graph is plain Go you can read top to bottom.

## Design Decisions

### Poisson simulator

Real football goals follow a Poisson distribution, so I sample from one instead of using uniform random ints. Team power and home advantage scale each side's expected goals, which feed the sampler. Result: mostly low-scoring games with the occasional blowout, which is what you want.

### Monte Carlo predictions

I run the remaining season N times (default 10,000), count how often each team finishes first, and that's the probability. Linear extrapolation from current points would ignore tiebreakers and remaining-opponent strength — Monte Carlo handles both for free since it reuses the same simulator and standings logic.

### Interfaces in `league/`, not `postgres/`

Repository and Predictor interfaces live where they're consumed. This is the Go convention (the consumer declares what it needs) and it means I could swap postgres for an in-memory store without touching the service.

### Error translation at the postgres boundary

The postgres package converts `pgconn.PgError` codes into domain errors before returning. The service never sees raw pg errors. Handlers can then map cleanly: `ErrNotFound` → 404, `ErrFixturesAlreadyExist` → 409.

### Migrations on startup

For local dev, migrations run at app startup. In production I'd run them as a separate job — running on every replica boot risks lock contention and couples deployments to schema changes.

### API errors are JSON-consistent

All errors return `{"error": "...", "code": "..."}`. Unknown paths and wrong methods both return 404 right now because of how I set up the catch-all route. A proper router like chi would split these into 404 and 405 — noted as a follow-up.

## Make Targets

| Target | Description |
| :--- | :--- |
| `make help`          | List all targets |
| `make db-up`         | Start PostgreSQL via Docker Compose |
| `make db-down`       | Stop PostgreSQL |
| `make db-shell`      | Open a `psql` shell to the database |
| `make migrate-up`    | Apply all pending migrations |
| `make migrate-down`  | Roll back the most recent migration |
| `make run`           | Run the application |
| `make test`          | Run all tests with the race detector |
| `make build`         | Compile the binary to `bin/server` |

## Development

### Run Tests

```bash
go test -v ./internal/domain    # domain types
go test -v ./internal/config    # config loader
```

The repository layer is verified by running the app end-to-end against the Dockerized Postgres. See "What I'd Add With More Time" for the integration test plan.

## Known Limitations

### Special characters in `DB_PASSWORD`

I build the database URL manually from env vars rather than URL-escaping the password. If `DB_PASSWORD` contains `@`, `:`, `/`, `?`, or `#`, the connection string parses wrong. Use alphanumeric passwords for now. The fix is `net/url.QueryEscape()` on the password before building the URL.

## What I'd Add With More Time

- **Repository tests** with testcontainers-go. Currently only `config` and `domain` have unit tests; the database layer is verified by running the app end-to-end.
- **A real router** like chi for proper 405 handling and cleaner middleware composition.
- **Separate readiness probe** that pings the database, distinct from the cheap liveness check at `/health`.
- **Migration runner as a separate job** instead of running on app startup. Fine for this case, wrong for production.