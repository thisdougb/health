package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/thisdougb/health/internal/storage"
)

// AdminInterface defines the interface needed for administrative data extraction
type AdminInterface interface {
	GetStorageManager() *storage.Manager
}

// MetricTimeSeriesEntry represents a single metric point in time series format
type MetricTimeSeriesEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Name      string      `json:"name"`
	Value     interface{} `json:"value"`
	Type      string      `json:"type"`
}

// ComponentMetrics represents metrics for a specific component
type ComponentMetrics struct {
	Component string                  `json:"component"`
	Metrics   []MetricTimeSeriesEntry `json:"metrics"`
}

// AllMetricsExport represents the complete export structure
type AllMetricsExport struct {
	StartTime  time.Time          `json:"start_time"`
	EndTime    time.Time          `json:"end_time"`
	Components []ComponentMetrics `json:"components"`
	Summary    ExportSummary      `json:"summary"`
}

// ExportSummary provides aggregate information about the export
type ExportSummary struct {
	TotalComponents int `json:"total_components"`
	TotalMetrics    int `json:"total_metrics"`
	TimeSpanHours   int `json:"time_span_hours"`
}

// HealthSummary provides aggregated health information for a time period
type HealthSummary struct {
	StartTime      time.Time                  `json:"start_time"`
	EndTime        time.Time                  `json:"end_time"`
	Components     []ComponentHealthSummary   `json:"components"`
	SystemMetrics  *SystemMetricsSummary      `json:"system_metrics,omitempty"`
	OverallSummary OverallHealthSummary       `json:"overall_summary"`
}

// ComponentHealthSummary provides summary for a single component
type ComponentHealthSummary struct {
	Component   string                     `json:"component"`
	MetricCount int                        `json:"metric_count"`
	Values      map[string]ValueSummary    `json:"values,omitempty"`
}

// ValueSummary provides statistical summary for value metrics
type ValueSummary struct {
	Count int     `json:"count"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
}

// SystemMetricsSummary provides summary for system metrics
type SystemMetricsSummary struct {
	CPUPercent      *ValueSummary `json:"cpu_percent,omitempty"`
	MemoryBytes     *ValueSummary `json:"memory_bytes,omitempty"`
	HealthDataSize  *ValueSummary `json:"health_data_size,omitempty"`
	Goroutines      *ValueSummary `json:"goroutines,omitempty"`
	UptimeSeconds   *ValueSummary `json:"uptime_seconds,omitempty"`
}

// OverallHealthSummary provides overall system health indicators
type OverallHealthSummary struct {
	TimeSpanHours   int `json:"time_span_hours"`
	TotalComponents int `json:"total_components"`
	TotalMetrics    int `json:"total_metrics"`
	SystemHealthy   bool `json:"system_healthy"`
}

// ExtractMetricsByTimeRange extracts metrics for a specific component and time range
// Returns JSON-formatted metrics optimized for Claude analysis
func ExtractMetricsByTimeRange(admin AdminInterface, component string, start, end time.Time) (string, error) {
	manager := admin.GetStorageManager()
	if manager == nil || !manager.IsEnabled() {
		return "", fmt.Errorf("persistence not enabled")
	}

	// Read metrics from storage
	metrics, err := manager.ReadMetrics(component, start, end)
	if err != nil {
		return "", fmt.Errorf("failed to read metrics: %w", err)
	}

	// Convert to time series format
	var timeSeriesData []MetricTimeSeriesEntry
	for _, metric := range metrics {
		timeSeriesData = append(timeSeriesData, MetricTimeSeriesEntry{
			Timestamp: metric.Timestamp,
			Name:      metric.Name,
			Value:     metric.Value,
			Type:      metric.Type,
		})
	}

	// Sort by timestamp for chronological order
	sort.Slice(timeSeriesData, func(i, j int) bool {
		return timeSeriesData[i].Timestamp.Before(timeSeriesData[j].Timestamp)
	})

	result := ComponentMetrics{
		Component: component,
		Metrics:   timeSeriesData,
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(output), nil
}

// ExportAllMetrics exports all metrics within a time range in JSON format
// Returns comprehensive data optimized for Claude processing
func ExportAllMetrics(admin AdminInterface, start, end time.Time, format string) (string, error) {
	if format != "json" {
		return "", fmt.Errorf("unsupported format: %s (only 'json' supported)", format)
	}

	manager := admin.GetStorageManager()
	if manager == nil || !manager.IsEnabled() {
		return "", fmt.Errorf("persistence not enabled")
	}

	// Get list of all components
	components, err := manager.ListComponents()
	if err != nil {
		return "", fmt.Errorf("failed to list components: %w", err)
	}

	var allComponents []ComponentMetrics
	totalMetrics := 0

	// Collect metrics for each component
	for _, component := range components {
		metrics, err := manager.ReadMetrics(component, start, end)
		if err != nil {
			continue // Skip components with errors
		}

		var timeSeriesData []MetricTimeSeriesEntry
		for _, metric := range metrics {
			timeSeriesData = append(timeSeriesData, MetricTimeSeriesEntry{
				Timestamp: metric.Timestamp,
				Name:      metric.Name,
				Value:     metric.Value,
				Type:      metric.Type,
			})
		}

		// Sort by timestamp
		sort.Slice(timeSeriesData, func(i, j int) bool {
			return timeSeriesData[i].Timestamp.Before(timeSeriesData[j].Timestamp)
		})

		allComponents = append(allComponents, ComponentMetrics{
			Component: component,
			Metrics:   timeSeriesData,
		})

		totalMetrics += len(timeSeriesData)
	}

	// Calculate time span in hours
	timeSpan := int(end.Sub(start).Hours())

	export := AllMetricsExport{
		StartTime:  start,
		EndTime:    end,
		Components: allComponents,
		Summary: ExportSummary{
			TotalComponents: len(allComponents),
			TotalMetrics:    totalMetrics,
			TimeSpanHours:   timeSpan,
		},
	}

	output, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(output), nil
}

// ListAvailableComponents returns a list of all components that have stored metrics
func ListAvailableComponents(admin AdminInterface) ([]string, error) {
	manager := admin.GetStorageManager()
	if manager == nil || !manager.IsEnabled() {
		return nil, fmt.Errorf("persistence not enabled")
	}

	components, err := manager.ListComponents()
	if err != nil {
		return nil, fmt.Errorf("failed to list components: %w", err)
	}

	// Sort alphabetically for consistent output
	sort.Strings(components)

	return components, nil
}

// GetHealthSummary provides an aggregated health summary for a time period
// Returns JSON with statistical analysis optimized for Claude interpretation
func GetHealthSummary(admin AdminInterface, start, end time.Time) (string, error) {
	manager := admin.GetStorageManager()
	if manager == nil || !manager.IsEnabled() {
		return "", fmt.Errorf("persistence not enabled")
	}

	// Get all components
	components, err := manager.ListComponents()
	if err != nil {
		return "", fmt.Errorf("failed to list components: %w", err)
	}

	var componentSummaries []ComponentHealthSummary
	totalMetrics := 0
	var systemSummary *SystemMetricsSummary

	// Process each component
	for _, component := range components {
		metrics, err := manager.ReadMetrics(component, start, end)
		if err != nil {
			continue // Skip components with errors
		}

		summary := generateComponentSummary(component, metrics)
		componentSummaries = append(componentSummaries, summary)
		totalMetrics += summary.MetricCount

		// Special handling for system metrics
		if component == "system" {
			systemSummary = generateSystemMetricsSummary(metrics)
		}
	}

	// Determine overall system health
	systemHealthy := true
	if systemSummary != nil {
		// Simple heuristic: system is unhealthy if memory or CPU are extremely high
		if systemSummary.MemoryBytes != nil && systemSummary.MemoryBytes.Max > 1e9 { // > 1GB
			systemHealthy = false
		}
		if systemSummary.CPUPercent != nil && systemSummary.CPUPercent.Max > 90 {
			systemHealthy = false
		}
	}

	timeSpan := int(end.Sub(start).Hours())

	healthSummary := HealthSummary{
		StartTime:     start,
		EndTime:       end,
		Components:    componentSummaries,
		SystemMetrics: systemSummary,
		OverallSummary: OverallHealthSummary{
			TimeSpanHours:   timeSpan,
			TotalComponents: len(componentSummaries),
			TotalMetrics:    totalMetrics,
			SystemHealthy:   systemHealthy,
		},
	}

	output, err := json.MarshalIndent(healthSummary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(output), nil
}

// generateComponentSummary creates a summary for a single component's metrics
func generateComponentSummary(component string, metrics []storage.MetricEntry) ComponentHealthSummary {
	values := make(map[string]ValueSummary)

	for _, metric := range metrics {
		// All metrics are now value metrics with statistical data
		// Handle aggregated value data from storage backend
		switch v := metric.Value.(type) {
		case map[string]interface{}:
			// Aggregated data from storage backend (avg, min, max, count)
			avg, _ := v["avg"].(float64)
			min, _ := v["min"].(float64)
			max, _ := v["max"].(float64)
			count, _ := v["count"].(int)
			
			values[metric.Name] = ValueSummary{
				Count: count,
				Min:   min,
				Max:   max,
				Avg:   avg,
			}
		case float64:
			// Direct float64 value
			if summary, exists := values[metric.Name]; exists {
				summary.Count++
				summary.Min = math.Min(summary.Min, v)
				summary.Max = math.Max(summary.Max, v)
				summary.Avg = ((summary.Avg * float64(summary.Count-1)) + v) / float64(summary.Count)
				values[metric.Name] = summary
			} else {
				values[metric.Name] = ValueSummary{Count: 1, Min: v, Max: v, Avg: v}
			}
		case float32:
			// Convert float32 to float64
			val := float64(v)
			if summary, exists := values[metric.Name]; exists {
				summary.Count++
				summary.Min = math.Min(summary.Min, val)
				summary.Max = math.Max(summary.Max, val)
				summary.Avg = ((summary.Avg * float64(summary.Count-1)) + val) / float64(summary.Count)
				values[metric.Name] = summary
			} else {
				values[metric.Name] = ValueSummary{Count: 1, Min: val, Max: val, Avg: val}
			}
		case int, int64:
			// Convert integer to float64
			var val float64
			switch i := v.(type) {
			case int:
				val = float64(i)
			case int64:
				val = float64(i)
			}
			if summary, exists := values[metric.Name]; exists {
				summary.Count++
				summary.Min = math.Min(summary.Min, val)
				summary.Max = math.Max(summary.Max, val)
				summary.Avg = ((summary.Avg * float64(summary.Count-1)) + val) / float64(summary.Count)
				values[metric.Name] = summary
			} else {
				values[metric.Name] = ValueSummary{Count: 1, Min: val, Max: val, Avg: val}
			}
		}
	}

	// Only include non-empty maps
	var valuesPtr map[string]ValueSummary
	if len(values) > 0 {
		valuesPtr = values
	}

	return ComponentHealthSummary{
		Component:   component,
		MetricCount: len(metrics),
		Values:      valuesPtr,
	}
}

// generateSystemMetricsSummary creates a specialized summary for system metrics
func generateSystemMetricsSummary(metrics []storage.MetricEntry) *SystemMetricsSummary {
	summary := &SystemMetricsSummary{}

	// Process metrics similar to generateComponentSummary to handle aggregated data
	values := make(map[string]ValueSummary)

	for _, metric := range metrics {
		// Handle aggregated value data from storage backend (same as generateComponentSummary)
		switch v := metric.Value.(type) {
		case map[string]interface{}:
			// Aggregated data from storage backend (avg, min, max, count)
			avg, _ := v["avg"].(float64)
			min, _ := v["min"].(float64)
			max, _ := v["max"].(float64)
			count, _ := v["count"].(int)
			
			values[metric.Name] = ValueSummary{
				Count: count,
				Min:   min,
				Max:   max,
				Avg:   avg,
			}
		case float64:
			// Direct float64 value
			if existing, exists := values[metric.Name]; exists {
				existing.Count++
				existing.Min = math.Min(existing.Min, v)
				existing.Max = math.Max(existing.Max, v)
				existing.Avg = ((existing.Avg * float64(existing.Count-1)) + v) / float64(existing.Count)
				values[metric.Name] = existing
			} else {
				values[metric.Name] = ValueSummary{Count: 1, Min: v, Max: v, Avg: v}
			}
		case float32:
			// Convert float32 to float64
			val := float64(v)
			if existing, exists := values[metric.Name]; exists {
				existing.Count++
				existing.Min = math.Min(existing.Min, val)
				existing.Max = math.Max(existing.Max, val)
				existing.Avg = ((existing.Avg * float64(existing.Count-1)) + val) / float64(existing.Count)
				values[metric.Name] = existing
			} else {
				values[metric.Name] = ValueSummary{Count: 1, Min: val, Max: val, Avg: val}
			}
		case int, int64:
			// Convert integer to float64
			var val float64
			switch i := v.(type) {
			case int:
				val = float64(i)
			case int64:
				val = float64(i)
			}
			if existing, exists := values[metric.Name]; exists {
				existing.Count++
				existing.Min = math.Min(existing.Min, val)
				existing.Max = math.Max(existing.Max, val)
				existing.Avg = ((existing.Avg * float64(existing.Count-1)) + val) / float64(existing.Count)
				values[metric.Name] = existing
			} else {
				values[metric.Name] = ValueSummary{Count: 1, Min: val, Max: val, Avg: val}
			}
		}
	}

	// Map processed values to system metrics fields
	if cpuSummary, exists := values["cpu_percent"]; exists {
		summary.CPUPercent = &cpuSummary
	}
	if memSummary, exists := values["memory_bytes"]; exists {
		summary.MemoryBytes = &memSummary
	}
	if sizeSummary, exists := values["health_data_size"]; exists {
		summary.HealthDataSize = &sizeSummary
	}
	if goroutinesSummary, exists := values["goroutines"]; exists {
		summary.Goroutines = &goroutinesSummary
	}
	if uptimeSummary, exists := values["uptime_seconds"]; exists {
		summary.UptimeSeconds = &uptimeSummary
	}

	return summary
}

// calculateValueSummary computes min, max, and average for a slice of values
func calculateValueSummary(values []float64) *ValueSummary {
	if len(values) == 0 {
		return nil
	}

	min := values[0]
	max := values[0]
	sum := 0.0

	for _, val := range values {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
		sum += val
	}

	return &ValueSummary{
		Count: len(values),
		Min:   min,
		Max:   max,
		Avg:   sum / float64(len(values)),
	}
}