package storage

import "time"

// Backend defines the interface for all storage implementations
type Backend interface {
	WriteMetrics(metrics []MetricEntry) error
	WriteMetricsData(metrics []MetricsDataEntry) error
	ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error)
	ListComponents() ([]string, error)
	Close() error
}

// MetricEntry represents a single metric data point
type MetricEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Component string      `json:"component"`
	Name      string      `json:"name"`
	Value     interface{} `json:"value"` // Metric value
	Type      string      `json:"type"`  // Metric type
}

// MetricsDataEntry represents aggregated metrics data for storage
type MetricsDataEntry struct {
	TimeWindowKey   string  `json:"time_window_key"`    // YYYYMMDDHHMMSS format
	Component       string  `json:"component"`
	Metric          string  `json:"metric"`
	MinValue        float64 `json:"min_value"`
	MaxValue        float64 `json:"max_value"`
	AvgValue        float64 `json:"avg_value"`
	Count           int     `json:"count"`
}
