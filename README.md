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