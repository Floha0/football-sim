# football-sim
Insider Long term internship backend case

## Prerequisites

- **Go 1.22 or later**
- **Docker and Docker Compose** for the PostgreSQL container
- **`golang-migrate` CLI** for running migrations:

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

## Design Decisions & Trade-offs

### Database Migrations

For local development, migrations are applied automatically at startup which keeps setup simple for the case project.

In production, normally I would run migrations separately through CI/CD or a one-off deployment job to avoid multiple application instances trying to migrate the database at the same time.

### API Conventions

- All errors return JSON in the form `{"error": "...", "code": "..."}`
- Unknown routes and method mismatches both return 404 to maintain JSON consistency
- A future improvement would use a router (e.g. chi) that supports custom 405 handlers

## Known Limitations

### Special Characters in Database Password
The current database connection string is built manually from environment variables
* **Impact:** If the `DB_PASSWORD` contains URL-reserved special characters (such as `@`, `:`, `/`, `?`, `#`), the connection URL parsing will break.
* **Mitigation:** For this case project, standard alphanumeric values (like `1234` or `changeme`) are assumed. In a production-grade system, the password string must be sanitized using Go's `net/url.QueryEscape()` before building the connection string to prevent syntax errors.

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
