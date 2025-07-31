package storage

import (
	"sort"
	"sync"
	"time"
)

// MemoryBackend implements Backend interface using in-memory storage
type MemoryBackend struct {
	mu      sync.RWMutex
	metrics []MetricEntry
}

// NewMemoryBackend creates a new in-memory storage backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		metrics: make([]MetricEntry, 0),
	}
}

// WriteMetrics stores metrics in memory
func (m *MemoryBackend) WriteMetrics(metrics []MetricEntry) error {
	if len(metrics) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = append(m.metrics, metrics...)
	return nil
}

// ReadMetrics retrieves metrics for a component within the time range
func (m *MemoryBackend) ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []MetricEntry
	for _, metric := range m.metrics {
		// Filter by component (empty string matches all components)
		if component != "" && metric.Component != component {
			continue
		}

		// Filter by time range
		if (metric.Timestamp.Equal(start) || metric.Timestamp.After(start)) &&
			(metric.Timestamp.Equal(end) || metric.Timestamp.Before(end)) {
			result = append(result, metric)
		}
	}

	// Sort by timestamp for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// ListComponents returns all unique component names
func (m *MemoryBackend) ListComponents() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	componentSet := make(map[string]bool)
	for _, metric := range m.metrics {
		componentSet[metric.Component] = true
	}

	components := make([]string, 0, len(componentSet))
	for component := range componentSet {
		components = append(components, component)
	}

	sort.Strings(components)
	return components, nil
}

// Close performs cleanup (no-op for memory backend)
func (m *MemoryBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = nil
	return nil
}

// Len returns the number of stored metrics (for testing)
func (m *MemoryBackend) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.metrics)
}

// Clear removes all stored metrics (for testing)
func (m *MemoryBackend) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = m.metrics[:0]
}