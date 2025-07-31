package storage

import "time"

// Backend defines the interface for all storage implementations
type Backend interface {
	WriteMetrics(metrics []MetricEntry) error
	ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error)
	ListComponents() ([]string, error)
	Close() error
}

// MetricEntry represents a single metric data point
type MetricEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Component string      `json:"component"`
	Name      string      `json:"name"`
	Value     interface{} `json:"value"` // int for counters, float64 for rolling
	Type      string      `json:"type"`  // "counter" or "rolling"
}