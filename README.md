# football-sim
Insider Long term internship backend case

## Environment Variables

The application configures itself using environment variables. You can create a `.env` file in the root directory based on the `.env.example` provided.

| Variable | Description | Default Value | Type |
| :--- | :--- | :--- | :--- |
| `PORT` | The port HTTP server will listen on | `8080` | `string` |
| `SHUTDOWN_TIMEOUT` | Max duration to wait for active requests during shutdown | `10s` | `duration` |
| `DB_HOST` | PostgreSQL server host address | `localhost` | `string` |
| `DB_PORT` | PostgreSQL server port | `5432` | `string` |
| `DB_USER` | Database username | `insider` | `string` |
| `DB_PASSWORD` | Database password | `changeme` | `string` |
| `DB_NAME` | Target database name | `insider_league` | `string` |
| `DB_SSLMODE` | PostgreSQL SSL mode (`disable`, `require`, etc.) | `disable` | `string` |
| `SIM_HOME_ADVANTAGE` | Power multiplier for the home team advantage | `1.15` | `float64` |
| `SIM_AVG_GOALS` | Average total goals expected per match (e.g., PL style) | `2.75` | `float64` |
| `SIM_MAX_GOALS` | Hard safety cap for goals scored by a single team | `10` | `int` |
| `PREDICTION_ITERATIONS`| Total runs for the Monte Carlo championship simulation | `10000` | `int` |
| `LOG_LEVEL` | Structured log verbosity (`debug`, `info`, `warn`, `error`) | `info` | `string` |

## Design Decisions & Trade-offs

### 1. Database Migrations on Application Startup
In this project, database migrations are automatically executed inside `main.go` during the application bootstrap phase. 
* **Pros:** Provides a seamless, zero-friction local development experience. The database auto-initializes itself on the very first `make run` or `go run` command.
* **Cons (Production Reality):** In a production environment with horizontal scaling (multiple replicas/pods), having every container attempt to run migrations simultaneously on boot can cause severe lock contention. Furthermore, rolling back schemas requires a full code deployment cycle.
* **Production Alternative:** In a real-world cloud environment, migrations should be decoupled from application startup and managed via a CI/CD deployment pipeline, a Kubernetes `initContainer`, or a dedicated one-off migration job.

## Known Limitations

### Special Characters in Database Password
The current database connection string is built manually from environment variables
* **Impact:** If the `DB_PASSWORD` contains URL-reserved special characters (such as `@`, `:`, `/`, `?`, `#`), the connection URL parsing will break.
* **Mitigation:** For this case project, standard alphanumeric values (like `1234` or `changeme`) are assumed. In a production-grade system, the password string must be sanitized using Go's `net/url.QueryEscape()` before building the connection string to prevent syntax errors.