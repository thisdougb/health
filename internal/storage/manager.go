package storage

import (
	"fmt"
	"time"
)

// Manager coordinates persistence operations between the State system and storage backends
type Manager struct {
	backend Backend
	enabled bool
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
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if !config.Enabled {
		// Return manager with no backend (disabled)
		return &Manager{
			backend: nil,
			enabled: false,
		}, nil
	}

	// Create appropriate backend based on configuration
	var backend Backend
	if config.DBPath != "" {
		// Use SQLite backend
		sqliteConfig := SQLiteConfig{
			DBPath:        config.DBPath,
			FlushInterval: config.FlushInterval,
			BatchSize:     config.BatchSize,
		}
		backend, err = NewSQLiteBackend(sqliteConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite backend: %w", err)
		}
	} else {
		// Use memory backend (for testing/development)
		backend = NewMemoryBackend()
	}

	return &Manager{
		backend: backend,
		enabled: true,
	}, nil
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
	if !m.enabled || m.backend == nil {
		return nil // Nothing to close
	}

	return m.backend.Close()
}

// IsEnabled returns whether persistence is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}