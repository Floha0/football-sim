// Package config_test tests the public API of the config package.
package config_test

import (
	"testing"
	"time"

	"github.com/Floha0/football-sim/internal/config"
)

// TestLoad_ValidEnv ensures that valid env vars are parsed into the correct types and values.
func TestLoad_ValidEnv(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret123")
	t.Setenv("PORT", "9090")
	t.Setenv("SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("SIM_HOME_ADVANTAGE", "1.20")
	t.Setenv("PREDICTION_ITERATIONS", "5000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Expected Port to be 9090, got %s", cfg.Server.Port)
	}

	if cfg.Server.ShutdownTimeout != 5*time.Second {
		t.Errorf("Expected ShutdownTimeout to be 5s, got %v", cfg.Server.ShutdownTimeout)
	}

	if cfg.Simulation.HomeAdvantage != 1.20 {
		t.Errorf("Expected HomeAdvantage to be 1.20, got %f", cfg.Simulation.HomeAdvantage)
	}

	if cfg.Prediction.MonteCarloIterations != 5000 {
		t.Errorf("Expected MonteCarloIterations to be 5000, got %d", cfg.Prediction.MonteCarloIterations)
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")

	_, err := config.Load()
	if err == nil {
		t.Error("Expected error due to missing DB_PASSWORD, got nil")
	}
}

func TestLoad_InvalidValues(t *testing.T) {
	t.Setenv("DB_PASSWORD", "valid_pass")

	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"invalid duration", "SHUTDOWN_TIMEOUT", "not-a-duration"},
		{"invalid float", "SIM_HOME_ADVANTAGE", "abc"},
		{"invalid int", "SIM_MAX_GOALS", "xyz"},
		{"invalid iterations", "PREDICTION_ITERATIONS", "1.5"},
		{"negative float logic", "SIM_HOME_ADVANTAGE", "-1.15"},
		{"zero max goals logic", "SIM_MAX_GOALS", "0"},
		{"zero iterations logic", "PREDICTION_ITERATIONS", "0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.envKey, tc.envVal)
			if _, err := config.Load(); err == nil {
				t.Errorf("expected error for %s=%s, got nil", tc.envKey, tc.envVal)
			}
		})
	}
}

func TestLoad_IsIdempotent(t *testing.T) {
	t.Setenv("DB_PASSWORD", "idempotent_pass")

	// First call
	cfg1, err := config.Load()
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Second call
	cfg2, err := config.Load()
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	if cfg1.Database.Password != cfg2.Database.Password {
		t.Errorf("Load is not idempotent, values differ: %s vs %s", cfg1.Database.Password, cfg2.Database.Password)
	}
}
