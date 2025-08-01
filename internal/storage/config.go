package storage

import (
	"os"
	"strconv"
	"time"
)

// BackupConfig holds backup-specific configuration
type BackupConfig struct {
	Enabled        bool
	BackupDir      string
	RetentionDays  int
	BackupInterval time.Duration
}

// Config holds all configuration options for the persistence system
type Config struct {
	Enabled       bool
	DBPath        string
	FlushInterval time.Duration
	BatchSize     int
	Backup        BackupConfig
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		// Set defaults
		Enabled:       false,
		DBPath:        "/tmp/health.db",
		FlushInterval: 60 * time.Second,
		BatchSize:     100,
		Backup: BackupConfig{
			Enabled:        false,
			BackupDir:      "/data/backups/health",
			RetentionDays:  30,
			BackupInterval: 24 * time.Hour,
		},
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

	// HEALTH_BACKUP_ENABLED - Enable/disable backup functionality
	if enabledStr := os.Getenv("HEALTH_BACKUP_ENABLED"); enabledStr != "" {
		config.Backup.Enabled = (enabledStr == "true")
	}

	// HEALTH_BACKUP_DIR - Backup directory path
	if backupDir := os.Getenv("HEALTH_BACKUP_DIR"); backupDir != "" {
		config.Backup.BackupDir = backupDir
	}

	// HEALTH_BACKUP_RETENTION_DAYS - How long to keep backups (in days)
	if retentionStr := os.Getenv("HEALTH_BACKUP_RETENTION_DAYS"); retentionStr != "" {
		if days, err := strconv.Atoi(retentionStr); err == nil && days >= 0 {
			config.Backup.RetentionDays = days
		}
	}

	// HEALTH_BACKUP_INTERVAL - How often to create backups  
	if intervalStr := os.Getenv("HEALTH_BACKUP_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			config.Backup.BackupInterval = interval
		}
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