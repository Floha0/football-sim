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
	"strings"
	"syscall"
	"time"

	"github.com/Floha0/football-sim/internal/config"
	"github.com/Floha0/football-sim/internal/handler"
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

	// 7. Construct HTTP handler and wire routes.
	apiHandler := handler.New(leagueService)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	// 8. Apply middleware in explicit order. The first listed is outermost.
	//    Outermost catches panics; inner middleware then runs.
	rootHandler := handler.Chain(mux,
		handler.WithRequestID,
		handler.WithLogging,
		handler.WithRecovery,
	)

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 9. Start the server in a goroutine so we can listen for shutdown signals.
	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	// 10. Wait for either a shutdown signal or a server error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	// 11. Gracefully shut down the HTTP server within the configured timeout.
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
