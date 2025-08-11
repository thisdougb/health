package storage

import (
	"fmt"
	"time"

	"github.com/thisdougb/health/internal/config"
)

// Manager coordinates persistence operations between the State system and storage backends.
// Uses a universal queue for consistent processing across all backend types.
type Manager struct {
	backend      Backend         // The configured storage backend (memory, SQLite, etc.)
	queue        *MetricsQueue   // Universal queue that handles all processing
	enabled      bool           // Whether persistence is enabled
	backupConfig BackupConfig   // Backup configuration settings
}

// NewManager creates a new persistence manager with universal queue.
// The queue ensures consistent processing regardless of backend type (memory, SQLite, etc.).
// Automatically starts the queue for background processing.
func NewManager(backend Backend, enabled bool) *Manager {
	var queue *MetricsQueue
	
	if enabled && backend != nil {
		// Create universal queue with default settings
		// These should match SQLite queue settings for consistency
		flushInterval := 60 * time.Second
		batchSize := 100
		
		queue = NewMetricsQueue(backend, flushInterval, batchSize)
		// Start background processing
		queue.Start()
	}
	
	return &Manager{
		backend: backend,
		queue:   queue,
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

	// Parse universal queue configuration
	flushInterval := 60 * time.Second
	if flushStr := config.StringValue("HEALTH_FLUSH_INTERVAL"); flushStr != "" {
		if interval, parseErr := time.ParseDuration(flushStr); parseErr == nil {
			flushInterval = interval
		}
	}
	batchSize := config.IntValue("HEALTH_BATCH_SIZE")
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// Create appropriate backend based on configuration
	var backend Backend
	var err error
	if dbPath := config.StringValue("HEALTH_DB_PATH"); dbPath != "" {
		// Use SQLite backend (CRUD-only)
		sqliteConfig := SQLiteConfig{
			DBPath: dbPath,
		}
		backend, err = NewSQLiteBackend(sqliteConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite backend: %w", err)
		}
	} else {
		// Use memory backend (CRUD-only, for testing/development)
		backend = NewMemoryBackend()
	}

	// Create universal queue for consistent processing across all backends
	queue := NewMetricsQueue(backend, flushInterval, batchSize)
	queue.Start()

	manager := &Manager{
		backend: backend,
		queue:   queue,
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

// PersistMetric queues a single raw metric for processing and storage.
// The universal queue handles aggregation and backend operations.
func (m *Manager) PersistMetric(component, name string, value interface{}, metricType string) error {
	if !m.enabled || m.queue == nil {
		return nil // Persistence disabled, no-op
	}

	entry := MetricEntry{
		Timestamp: time.Now(),
		Component: component,
		Name:      name,
		Value:     value,
		Type:      metricType,
	}

	// Queue handles all processing - no direct backend calls
	return m.queue.Enqueue([]MetricEntry{entry})
}

// PersistMetrics queues raw metrics for processing and storage.
// The universal queue handles aggregation and backend storage operations.
// This ensures consistent behavior across all backend types.
func (m *Manager) PersistMetrics(entries []MetricEntry) error {
	if !m.enabled || m.queue == nil {
		return nil // Persistence disabled, no-op
	}

	// Queue handles all processing - no direct backend calls
	return m.queue.Enqueue(entries)
}

// PersistMetricsData persists aggregated metrics data if persistence is enabled
func (m *Manager) PersistMetricsData(entries []MetricsDataEntry) error {
	if !m.enabled || m.backend == nil {
		return nil // Persistence disabled, no-op
	}

	return m.backend.WriteMetricsData(entries)
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

// Close gracefully shuts down the persistence manager.
// Stops the universal queue, flushes pending data, creates backups, and closes backend.
func (m *Manager) Close() error {
	// Stop universal queue first to ensure all data is processed
	if m.queue != nil {
		m.queue.Stop() // This flushes pending data
	}
	
	// Event-driven backup on shutdown
	if m.backupConfig.Enabled && m.enabled && m.backend != nil {
		_ = m.createBackupInternal() // Best effort, don't fail on error during shutdown
	}

	// Close backend connection
	if m.enabled && m.backend != nil {
		return m.backend.Close()
	}
	
	return nil
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

// ForceFlush immediately processes any queued metrics to storage.
// This works consistently across all backend types via the universal queue.
// Useful for testing to ensure metrics are immediately available.
func (m *Manager) ForceFlush() error {
	if !m.enabled || m.queue == nil {
		return nil // Nothing to flush
	}

	// Universal queue works with all backends consistently
	return m.queue.ForceFlush()
}
