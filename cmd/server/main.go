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
	"github.com/Floha0/football-sim/internal/postgres"
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

	// 6. Build a minimal HTTP server for now.
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

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 7. Start the server in a goroutine so we can listen for shutdown signals.
	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	// 8. Wait for either a shutdown signal or a server error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	// 9. Gracefully shut down the HTTP server within the configured timeout.
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
