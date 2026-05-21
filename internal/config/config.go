// Package config loads all runtime configurations from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Simulation SimulationConfig
	Prediction PredictionConfig
	LogLevel   string
}

type ServerConfig struct {
	Port            string
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type SimulationConfig struct {
	HomeAdvantage     float64
	AverageTotalGoals float64
	MaxGoalsPerTeam   int
}

type PredictionConfig struct {
	MonteCarloIterations int
}

// Load reads and validates all environment variables, returning a populated
// Config. It fails fast so misconfigured deployments surface errors at boot.
func Load() (*Config, error) {
	dbPassword, ok := os.LookupEnv("DB_PASSWORD")
	if !ok {
		return nil, errors.New("DB_PASSWORD is required")
	}

	// Server
	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err)
	}

	// Simulation
	homeAdv, err := strconv.ParseFloat(getEnv("SIM_HOME_ADVANTAGE", "1.15"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid SIM_HOME_ADVANTAGE: %w", err)
	}

	avgGoals, err := strconv.ParseFloat(getEnv("SIM_AVG_GOALS", "2.75"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid SIM_AVG_GOALS: %w", err)
	}

	maxGoals, err := strconv.Atoi(getEnv("SIM_MAX_GOALS", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid SIM_MAX_GOALS: %w", err)
	}

	// Prediction
	predIterations, err := strconv.Atoi(getEnv("PREDICTION_ITERATIONS", "10000"))
	if err != nil {
		return nil, fmt.Errorf("invalid PREDICTION_ITERATIONS: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "insider"),
			Password: dbPassword,
			Name:     getEnv("DB_NAME", "insider_league"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Simulation: SimulationConfig{
			HomeAdvantage:     homeAdv,
			AverageTotalGoals: avgGoals,
			MaxGoalsPerTeam:   maxGoals,
		},
		Prediction: PredictionConfig{
			MonteCarloIterations: predIterations,
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	// Validation
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Simulation.HomeAdvantage <= 0 {
		return fmt.Errorf("SIM_HOME_ADVANTAGE must be positive, got %f", c.Simulation.HomeAdvantage)
	}
	if c.Simulation.AverageTotalGoals <= 0 {
		return fmt.Errorf("SIM_AVG_GOALS must be positive, got %f", c.Simulation.AverageTotalGoals)
	}
	if c.Simulation.MaxGoalsPerTeam < 1 {
		return fmt.Errorf("SIM_MAX_GOALS must be at least 1, got %d", c.Simulation.MaxGoalsPerTeam)
	}
	if c.Prediction.MonteCarloIterations < 1 {
		return fmt.Errorf("PREDICTION_ITERATIONS must be at least 1, got %d", c.Prediction.MonteCarloIterations)
	}
	return nil
}

// Prevents password leakage to logs
func (c *Config) String() string {
	return fmt.Sprintf(
		"ServerPort: %s, DB_Host: %s, DB_Name: %s, DB_Password: [REDACTED], HomeAdv: %.2f, AvgGoals: %.2f, MonteCarlo: %d",
		c.Server.Port, c.Database.Host, c.Database.Name, c.Simulation.HomeAdvantage, c.Simulation.AverageTotalGoals, c.Prediction.MonteCarloIterations,
	)
}

// getEnv returns the env variable value for key, or fallback if unset.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
