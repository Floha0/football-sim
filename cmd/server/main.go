// Package main is the entry point for the football-sim server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Floha0/football-sim/internal/config"
	"github.com/Floha0/football-sim/internal/league"
	"github.com/Floha0/football-sim/internal/postgres"
	"github.com/Floha0/football-sim/internal/prediction"
	"github.com/Floha0/football-sim/internal/simulation"
)

func main() {
	// Exit through run() so deferred cleanups always execute.
	// os.Exit bypasses defers, so we never call it from main directly.
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Load configuration. Fail fast on bad config.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Initialize structured logger.
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)
	logger.Info("starting football-sim",
		"port", cfg.Server.Port,
		"log_level", cfg.LogLevel,
	)

	// 3. Set up signal-aware context for graceful shutdown.
	// SIGINT (Ctrl+C) and SIGTERM (Docker stop) both trigger cancellation.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Connect to the database. Ping is performed inside Connect().
	pool, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connection established")

	// 5. Run pending migrations. ErrNoChange is treated as success.
	dsn := postgres.DSN(cfg.Database)
	if err := postgres.RunMigrations(dsn, "./migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("migrations applied")

	// 6. Construct repositories.
	teamRepo := postgres.NewTeamRepo(pool)
	matchRepo := postgres.NewMatchRepo(pool)

	simulator := simulation.NewPoissonSimulator(cfg.Simulation, nil)
	predictor := prediction.NewMonteCarloPredictor(cfg.Prediction, simulator)
	leagueService := league.NewService(teamRepo, matchRepo, simulator, predictor)
	_ = leagueService // wired into HTTP handlers in the next commit

	// 7. Build a minimal HTTP server for now.
	// Real routes and handlers come in later commits.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			logger.Error("health check failed", "err", err)
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Temporary diagnostic endpoint to verify Repository layer & Seeding data.
	// This confirms pgx connection pooling, data scanning, and error translation work.
	mux.HandleFunc("GET /health/teams", func(w http.ResponseWriter, r *http.Request) {
		teams, err := teamRepo.GetAll(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "found %d teams\n", len(teams))
		for _, t := range teams {
			fmt.Fprintf(w, "  - %s (power=%d)\n", t.Name, t.Power)
		}
	})

	mux.HandleFunc("POST /debug/init", func(w http.ResponseWriter, r *http.Request) {
		if err := leagueService.GenerateFixtures(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "fixtures generated")
	})

	mux.HandleFunc("POST /debug/play-all", func(w http.ResponseWriter, r *http.Request) {
		result, err := leagueService.PlayAll(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "played %d weeks\n", len(result))
	})

	mux.HandleFunc("GET /debug/standings", func(w http.ResponseWriter, r *http.Request) {
		standings, err := leagueService.GetStandings(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i, s := range standings {
			fmt.Fprintf(w, "%d. team=%d P=%d W=%d D=%d L=%d GF=%d GA=%d GD=%d Pts=%d\n",
				i+1, s.TeamID, s.Played, s.Wins, s.Draws, s.Losses,
				s.GoalsFor, s.GoalsAgainst, s.GoalDiff(), s.Points)
		}
	})

	mux.HandleFunc("GET /debug/predictions", func(w http.ResponseWriter, r *http.Request) {
		preds, err := leagueService.Predict(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, p := range preds {
			fmt.Fprintf(w, "team=%d chance=%.2f%% (week %d)\n", p.TeamID, p.Chance*100, p.Week)
		}
	})

	mux.HandleFunc("POST /debug/play-n", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("weeks"))
		if n <= 0 {
			n = 4
		}
		for i := 0; i < n; i++ {
			_, _, err := leagueService.PlayWeek(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		fmt.Fprintf(w, "played %d weeks\n", n)
	})

	mux.HandleFunc("POST /debug/reset", func(w http.ResponseWriter, r *http.Request) {
		if err := leagueService.Reset(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "league reset; fixtures regenerated")
	})

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 8. Start the server in a goroutine so we can listen for shutdown signals.
	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	// 9. Wait for either a shutdown signal or a server error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	// 10. Gracefully shut down the HTTP server within the configured timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		return fmt.Errorf("server shutdown: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

// newLogger constructs a slog.Logger with JSON output and the configured level.
// Unknown levels fall back to info.
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
