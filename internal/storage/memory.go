package storage

import (
	"sort"
	"sync"
	"time"
	
	"github.com/thisdougb/health/internal/config"
)

// MemoryBackend implements Backend interface using in-memory storage
type MemoryBackend struct {
	mu              sync.RWMutex
	metrics         []MetricEntry
	timeSeriesData  []TimeSeriesEntry
}

// NewMemoryBackend creates a new in-memory storage backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		metrics:        make([]MetricEntry, 0),
		timeSeriesData: make([]TimeSeriesEntry, 0),
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

// WriteTimeSeriesMetrics stores aggregated time series metrics in memory
func (m *MemoryBackend) WriteTimeSeriesMetrics(metrics []TimeSeriesEntry) error {
	if len(metrics) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.timeSeriesData = append(m.timeSeriesData, metrics...)
	return nil
}

// ReadMetrics retrieves aggregated time series metrics for a component within the time range
func (m *MemoryBackend) ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert time range to time window keys for filtering
	startKey := timeToWindowKey(start)
	endKey := timeToWindowKey(end)

	var result []MetricEntry
	for _, tsEntry := range m.timeSeriesData {
		// Filter by component (empty string matches all components)
		if component != "" && tsEntry.Component != component {
			continue
		}

		// Filter by time window key range
		if tsEntry.TimeWindowKey < startKey || tsEntry.TimeWindowKey > endKey {
			continue
		}

		// Convert time window key back to timestamp
		timestamp, err := windowKeyToTime(tsEntry.TimeWindowKey)
		if err != nil {
			continue // Skip invalid time keys
		}

		// Create MetricEntry with aggregated data
		var value interface{}
		var metricType string
		if tsEntry.MinValue == 1.0 && tsEntry.MaxValue == 1.0 && tsEntry.AvgValue == 1.0 {
			// Counter metric - use count
			value = tsEntry.Count
			metricType = "counter"
		} else {
			// Value metric - provide full statistics
			value = map[string]interface{}{
				"avg":   tsEntry.AvgValue,
				"min":   tsEntry.MinValue, 
				"max":   tsEntry.MaxValue,
				"count": tsEntry.Count,
			}
			metricType = "value"
		}

		entry := MetricEntry{
			Timestamp: timestamp,
			Component: tsEntry.Component,
			Name:      tsEntry.Metric,
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

// timeToWindowKey converts a timestamp to a time window key format
func timeToWindowKey(t time.Time) string {
	// Use HEALTH_SAMPLE_RATE config value for window duration (default 60 seconds)
	windowSeconds := config.IntValue("HEALTH_SAMPLE_RATE")
	windowDuration := time.Duration(windowSeconds) * time.Second
	
	// Truncate to the time window boundary
	truncated := t.Truncate(windowDuration)
	
	// Format as YYYYMMDDHHMMSS with trailing zeros
	return truncated.Format("20060102150400")
}

// windowKeyToTime converts a time window key back to a timestamp
func windowKeyToTime(key string) (time.Time, error) {
	return time.Parse("20060102150400", key)
}
