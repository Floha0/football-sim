# football-sim
A backend simulation of a football league, built in Go for the Insider internship case study.

Generates a round-robin fixture, plays matches using a Poisson-based simulator, and predicts championship probabilities via Monte Carlo simulation of remaining fixtures.

**Tech stack:** Go 1.22+, PostgreSQL, pgx, golang-migrate, log/slog

## Quick Start

```bash
# 1. Clone and enter the repo
git clone https://github.com/Floha0/football-sim
cd football-sim

# 2. Copy environment template
cp .env.example .env
# Edit .env to set DB_PASSWORD (required, no default)

# 3. Start PostgreSQL
make db-up

# 4. Apply migrations
make migrate-up

# 5. Run the server
make run
```

The server listens on `http://localhost:8000`. See [API Endpoints](#api-endpoints) below for available routes.

## Prerequisites

- **Go 1.22 or later**
- **Docker and Docker Compose** for the PostgreSQL container
- **`golang-migrate` CLI** for running migrations:
```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Environment Variables

The application configures itself using environment variables. You can create a `.env` file in the root directory based on the `.env.example` provided.

| Variable | Description | Default Value | Type |
| :--- | :--- | :--- | :--- |
| `PORT` | The port HTTP server will listen on | `8080` | `string` |
| `SHUTDOWN_TIMEOUT` | Max duration to wait for active requests during shutdown | `10s` | `duration` |
| `DB_HOST` | PostgreSQL server host address | `localhost` | `string` |
| `DB_PORT` | PostgreSQL server port | `5432` | `string` |
| `DB_USER` | Database username | `insider` | `string` |
| `DB_PASSWORD` | Database password | required | `string` |
| `DB_NAME` | Target database name | `insider_league` | `string` |
| `DB_SSLMODE` | PostgreSQL SSL mode (`disable`, `require`, etc.) | `disable` | `string` |
| `SIM_HOME_ADVANTAGE` | Power multiplier for the home team advantage | `1.15` | `float64` |
| `SIM_AVG_GOALS` | Average total goals expected per match (e.g., PL style) | `2.75` | `float64` |
| `SIM_MAX_GOALS` | Hard safety cap for goals scored by a single team | `10` | `int` |
| `PREDICTION_ITERATIONS`| Total runs for the Monte Carlo championship simulation | `10000` | `int` |
| `LOG_LEVEL` | Structured log verbosity (`debug`, `info`, `warn`, `error`) | `info` | `string` |

## API Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| GET    | `/health`           | Liveness check |
| GET    | `/api/standings`    | Current league table (sorted by Premier League tiebreakers) |
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

# View standings
curl http://localhost:8080/api/standings | jq

# View championship predictions at week 4
curl http://localhost:8080/api/predictions | jq

# Edit a match (provide a real match ID)
curl -X PUT http://localhost:8080/api/matches/5 \
    -H "Content-Type: application/json" \
    -d '{"homeGoals": 3, "awayGoals": 1}'

# Play remaining weeks
curl -X POST http://localhost:8080/api/play-all | jq
```

## Architecture
```
cmd/server/         entrypoint, dependency wiring, graceful shutdown
internal/
config/           env-based configuration with validation
domain/           pure types (Team, Match, Standing, Prediction)
league/           service orchestrating fixtures, simulation, standings
prediction/       Monte Carlo championship predictor
simulation/       Poisson-based match simulator
postgres/         pgx-backed repository implementations + migrations
handler/          HTTP transport: DTOs, routing, middleware, error mapping
migrations/         SQL migration files
```
Three principles:

1. **Interfaces are defined by consumers.** Repository and Predictor interfaces live in `internal/league`
2. **Dependencies flow inward.** `handler` → `league` → `domain`
3. **All wiring happens in `main.go`.** No DI framework, no reflection. The dependency graph is plain Go you can read top to bottom.

## Design Decisions

### Poisson simulator

Real football goals follow a Poisson distribution, so I sample from one instead 
of using uniform random ints. Team power and home advantage scale each side's 
expected goals, which feed the sampler. Result: mostly low-scoring games with 
the occasional blowout, which is what you want.

### Monte Carlo predictions

I run the remaining season N times (default 10,000), count how often each team 
finishes first, and that's the probability. Linear extrapolation from current 
points would ignore tiebreakers and remaining-opponent strength — Monte Carlo 
handles both for free since it reuses the same simulator and standings logic.

### Interfaces in `league/`, not `postgres/`

Repository and Predictor interfaces live where they're consumed. This is the 
Go convention (the consumer declares what it needs) and it means I could swap 
postgres for an in-memory store without touching the service.

### Error translation at the postgres boundary

The postgres package converts `pgconn.PgError` codes into domain errors before 
returning. The service never sees raw pg errors. Handlers can then map cleanly: 
`ErrNotFound` → 404, `ErrFixturesAlreadyExist` → 409.

### Migrations on startup

For local dev, migrations run at app startup. In production I'd run them as a 
separate job — running on every replica boot risks lock contention and 
couples deployments to schema changes.

### API errors are JSON-consistent

All errors return `{"error": "...", "code": "..."}`. Unknown paths and wrong 
methods both return 404 right now because of how I set up the catch-all route. 
A proper router like chi would split these into 404 and 405 — noted as a 
follow-up.

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
To run the domain package tests, use the following command:
```bash
go test -v ./internal/domain
```
To run the config tests, use the following command:
```bash
go test -v ./internal/config
```

## Known Limitations

### Special characters in `DB_PASSWORD`
I build the database URL manually from env vars rather than URL-escaping the 
password. If `DB_PASSWORD` contains `@`, `:`, `/`, `?`, or `#`, the connection 
string parses wrong. Use alphanumeric passwords for now. The fix is 
`net/url.QueryEscape()` on the password before building the URL.


## What I'd Add With More Time

- **Repository tests** with testcontainers-go. Currently only `config` and 
  `domain` have unit tests; the database layer is verified by running the 
  app end-to-end. Real CI would need integration tests.
- **Postman collection** committed to the repo so reviewers don't have to 
  craft curl commands.
- **A real router** like chi for proper 405 handling and cleaner middleware 
  composition.
- **Separate readiness probe** that pings the database, distinct from the 
  cheap liveness check at `/health`.
- **Migration runner as a separate job** instead of running on app startup. 
  Fine for this case, wrong for production.