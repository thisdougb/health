package storage

import (
	"fmt"
	"time"

	"github.com/thisdougb/health/internal/config"
)

// Manager coordinates persistence operations between the State system and storage backends
type Manager struct {
	backend      Backend
	enabled      bool
	backupConfig BackupConfig
}

// NewManager creates a new persistence manager
func NewManager(backend Backend, enabled bool) *Manager {
	return &Manager{
		backend: backend,
		enabled: enabled,
	}
}

// NewManagerFromConfig creates a manager using environment variable configuration
func NewManagerFromConfig() (*Manager, error) {
	if !config.BoolValue("HEALTH_PERSISTENCE_ENABLED") {
		// Return manager with no backend (disabled)
		return &Manager{
			backend: nil,
			enabled: false,
		}, nil
	}

	// Create appropriate backend based on configuration
	var backend Backend
	var err error
	if dbPath := config.StringValue("HEALTH_DB_PATH"); dbPath != "" {
		// Parse flush interval
		flushInterval := 60 * time.Second
		if flushStr := config.StringValue("HEALTH_FLUSH_INTERVAL"); flushStr != "" {
			if interval, parseErr := time.ParseDuration(flushStr); parseErr == nil {
				flushInterval = interval
			}
		}

		// Use SQLite backend
		sqliteConfig := SQLiteConfig{
			DBPath:        dbPath,
			FlushInterval: flushInterval,
			BatchSize:     config.IntValue("HEALTH_BATCH_SIZE"),
		}
		backend, err = NewSQLiteBackend(sqliteConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite backend: %w", err)
		}
	} else {
		// Use memory backend (for testing/development)
		backend = NewMemoryBackend()
	}

	manager := &Manager{
		backend: backend,
		enabled: true,
		backupConfig: BackupConfig{
			Enabled:       config.BoolValue("HEALTH_BACKUP_ENABLED"),
			BackupDir:     config.StringValue("HEALTH_BACKUP_DIR"),
			RetentionDays: config.IntValue("HEALTH_BACKUP_RETENTION_DAYS"),
			BackupInterval: 24 * time.Hour,
		},
	}

	return manager, nil
}

// PersistMetric persists a single metric if persistence is enabled
func (m *Manager) PersistMetric(component, name string, value interface{}, metricType string) error {
	if !m.enabled || m.backend == nil {
		return nil // Persistence disabled, no-op
	}

	entry := MetricEntry{
		Timestamp: time.Now(),
		Component: component,
		Name:      name,
		Value:     value,
		Type:      metricType,
	}

	return m.backend.WriteMetrics([]MetricEntry{entry})
}

// PersistMetrics persists multiple metrics in batch if persistence is enabled
func (m *Manager) PersistMetrics(entries []MetricEntry) error {
	if !m.enabled || m.backend == nil {
		return nil // Persistence disabled, no-op
	}

	return m.backend.WriteMetrics(entries)
}

// PersistTimeSeriesMetrics persists aggregated time series metrics if persistence is enabled
func (m *Manager) PersistTimeSeriesMetrics(entries []TimeSeriesEntry) error {
	if !m.enabled || m.backend == nil {
		return nil // Persistence disabled, no-op
	}

	return m.backend.WriteTimeSeriesMetrics(entries)
}

// ReadMetrics retrieves metrics from storage
func (m *Manager) ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error) {
	if !m.enabled || m.backend == nil {
		return nil, fmt.Errorf("persistence not enabled")
	}

	return m.backend.ReadMetrics(component, start, end)
}

// ListComponents lists all components that have stored metrics
func (m *Manager) ListComponents() ([]string, error) {
	if !m.enabled || m.backend == nil {
		return nil, fmt.Errorf("persistence not enabled")
	}

	return m.backend.ListComponents()
}

// Close gracefully shuts down the persistence manager and flushes any pending data
func (m *Manager) Close() error {
	// Event-driven backup on shutdown
	if m.backupConfig.Enabled && m.enabled && m.backend != nil {
		_ = m.createBackupInternal() // Best effort, don't fail on error during shutdown
	}

	if !m.enabled || m.backend == nil {
		return nil // Nothing to close
	}

	return m.backend.Close()
}

// IsEnabled returns whether persistence is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// CreateBackup creates a backup of the health database (event-driven)
// Following backup patterns with event-driven triggers
func (m *Manager) CreateBackup() error {
	return m.createBackupInternal()
}

// createBackupInternal handles the backup creation using the backend's database connection
func (m *Manager) createBackupInternal() error {
	if !m.backupConfig.Enabled || !m.enabled || m.backend == nil {
		return nil // Backup disabled or no backend
	}

	// Only SQLite backends support backup (memory backends are temporary)
	sqliteBackend, ok := m.backend.(*SQLiteBackend)
	if !ok {
		return nil // Memory backend, no backup needed
	}

	// Use the SQLite backend's internal backup method to avoid file locking
	return sqliteBackend.CreateBackup(&m.backupConfig)
}

// ListBackups returns available backup files
func (m *Manager) ListBackups() ([]string, error) {
	if !m.backupConfig.Enabled {
		return nil, fmt.Errorf("backup not enabled")
	}

	return ListHealthBackups(&m.backupConfig)
}

// RestoreFromBackup restores database from specified backup file
func (m *Manager) RestoreFromBackup(backupFileName string, targetDBPath string) error {
	if !m.backupConfig.Enabled {
		return fmt.Errorf("backup not enabled")
	}

	return RestoreHealthDatabase(backupFileName, targetDBPath, &m.backupConfig)
}

// GetBackupInfo returns backup configuration information
func (m *Manager) GetBackupInfo() map[string]interface{} {
	return GetBackupRetentionInfo(&m.backupConfig)
}

// ForceFlush immediately flushes any queued metrics to storage (for testing)
func (m *Manager) ForceFlush() error {
	if !m.enabled || m.backend == nil {
		return nil // Nothing to flush
	}

	// Only SQLite backend has a queue to flush
	sqliteBackend, ok := m.backend.(*SQLiteBackend)
	if !ok {
		return nil // Memory backend, no queue to flush
	}

	// Force flush the queue
	return sqliteBackend.queue.ForceFlush()
}
