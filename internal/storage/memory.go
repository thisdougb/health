package storage

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryBackend implements Backend interface using in-memory storage.
// This is a simple CRUD backend that only handles storage operations.
// All processing logic is handled by the universal queue before calling this backend.
type MemoryBackend struct {
	mu      sync.RWMutex
	storage []MetricsDataEntry  // Primary in-memory storage for processed metrics data
}

// NewMemoryBackend creates a new in-memory storage backend.
// Returns a clean backend ready for CRUD operations.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		storage: make([]MetricsDataEntry, 0),
	}
}

// WriteMetrics is not used by memory backend - raw metrics are processed by queue.
// This method exists to satisfy the Backend interface but should not be called.
// The universal queue handles raw metrics and calls WriteMetricsData with processed data.
func (m *MemoryBackend) WriteMetrics(metrics []MetricEntry) error {
	// Memory backend only handles processed data via WriteMetricsData
	// Raw metrics should be handled by the universal queue
	return fmt.Errorf("memory backend only accepts processed data via WriteMetricsData")
}

// WriteMetricsData stores processed metrics data in memory storage.
// This is a simple CRUD operation - data has already been aggregated by the queue.
// Thread-safe storage of processed metrics for later retrieval.
func (m *MemoryBackend) WriteMetricsData(metrics []MetricsDataEntry) error {
	if len(metrics) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple append operation - queue has already done all processing
	m.storage = append(m.storage, metrics...)
	return nil
}

// ReadMetrics retrieves stored metrics for a component within the time range.
// This is a simple CRUD read operation that queries the processed storage.
// Converts stored MetricsDataEntry back to MetricEntry format for compatibility.
func (m *MemoryBackend) ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert time range to window keys for filtering
	startKey := timeToWindowKey(start)
	endKey := timeToWindowKey(end)

	var result []MetricEntry

	// Query processed storage data
	for _, storedEntry := range m.storage {
		// Filter by component (empty string matches all components)
		if component != "" && storedEntry.Component != component {
			continue
		}

		// Filter by time window key range
		if storedEntry.TimeWindowKey < startKey || storedEntry.TimeWindowKey > endKey {
			continue
		}

		// Convert time window key back to timestamp
		timestamp, err := windowKeyToTime(storedEntry.TimeWindowKey)
		if err != nil {
			continue // Skip invalid time keys
		}

		// Convert processed data back to MetricEntry format for compatibility
		var value interface{}
		var metricType string

		// All metrics are value metrics with statistical aggregation
		value = map[string]interface{}{
			"avg":   storedEntry.AvgValue,
			"min":   storedEntry.MinValue, 
			"max":   storedEntry.MaxValue,
			"count": storedEntry.Count,
		}
		metricType = "value"

		// Create MetricEntry from stored aggregated data
		entry := MetricEntry{
			Timestamp: timestamp,
			Component: storedEntry.Component,
			Name:      storedEntry.Metric,
			Value:     value,
			Type:      metricType,
		}
		result = append(result, entry)
	}

	// Sort by timestamp for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// ListComponents returns all unique component names from stored data.
// This is a simple CRUD read operation that queries processed storage for component names.
func (m *MemoryBackend) ListComponents() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Build unique set of component names from processed storage
	componentSet := make(map[string]bool)
	for _, entry := range m.storage {
		componentSet[entry.Component] = true
	}

	// Convert set to sorted slice
	components := make([]string, 0, len(componentSet))
	for component := range componentSet {
		components = append(components, component)
	}

	sort.Strings(components)
	return components, nil
}

// Close performs cleanup for memory backend
func (m *MemoryBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear storage
	m.storage = nil
	return nil
}

// Len returns the number of stored metrics entries (for testing)
func (m *MemoryBackend) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.storage)
}

// Clear removes all stored metrics (for testing)
func (m *MemoryBackend) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage = m.storage[:0]
}

