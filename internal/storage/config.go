package storage

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration options for the persistence system
type Config struct {
	Enabled       bool
	DBPath        string
	FlushInterval time.Duration
	BatchSize     int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		// Set defaults
		Enabled:       false,
		DBPath:        "/tmp/health.db",
		FlushInterval: 60 * time.Second,
		BatchSize:     100,
	}

	// HEALTH_PERSISTENCE_ENABLED - Enable/disable persistence
	if enabledStr := os.Getenv("HEALTH_PERSISTENCE_ENABLED"); enabledStr != "" {
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			// Invalid value, keep default (false)
		} else {
			config.Enabled = enabled
		}
	}

	// HEALTH_DB_PATH - Database file path
	if dbPath := os.Getenv("HEALTH_DB_PATH"); dbPath != "" {
		config.DBPath = dbPath
	}

	// HEALTH_FLUSH_INTERVAL - How often to flush metrics to storage
	if flushStr := os.Getenv("HEALTH_FLUSH_INTERVAL"); flushStr != "" {
		if interval, err := time.ParseDuration(flushStr); err == nil {
			config.FlushInterval = interval
		}
		// Invalid duration keeps default
	}

	// HEALTH_BATCH_SIZE - Number of metrics to batch before writing
	if batchStr := os.Getenv("HEALTH_BATCH_SIZE"); batchStr != "" {
		if size, err := strconv.Atoi(batchStr); err == nil && size > 0 {
			config.BatchSize = size
		}
		// Invalid size keeps default
	}

	return config, nil
}

// DefaultConfig returns a default configuration for testing
func DefaultConfig() *Config {
	return &Config{
		Enabled:       false,
		DBPath:        ":memory:",
		FlushInterval: 60 * time.Second,
		BatchSize:     100,
	}
}

// TestConfig returns a configuration suitable for testing
func TestConfig() *Config {
	return &Config{
		Enabled:       true,
		DBPath:        ":memory:",
		FlushInterval: 1 * time.Second,
		BatchSize:     10,
	}
}