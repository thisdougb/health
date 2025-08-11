package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/thisdougb/health/internal/storage"
)

// MockAdminState implements AdminInterface for testing
type MockAdminState struct {
	manager *storage.Manager
}

func (m *MockAdminState) GetStorageManager() *storage.Manager {
	return m.manager
}

// setupTestData creates test data in the storage manager
func setupTestData(manager *storage.Manager) error {
	// Create test metrics with timestamps spread over a few hours
	baseTime := time.Now().Add(-2 * time.Hour)
	
	testMetrics := []storage.MetricEntry{
		// Global component metrics
		{Timestamp: baseTime, Component: "Global", Name: "requests", Value: 10, Type: "value"},
		{Timestamp: baseTime.Add(30 * time.Minute), Component: "Global", Name: "requests", Value: 25, Type: "value"},
		{Timestamp: baseTime.Add(60 * time.Minute), Component: "Global", Name: "requests", Value: 40, Type: "value"},
		
		// Web server component metrics
		{Timestamp: baseTime, Component: "webserver", Name: "http_requests", Value: 100, Type: "value"},
		{Timestamp: baseTime.Add(30 * time.Minute), Component: "webserver", Name: "response_time", Value: 145.2, Type: "value"},
		{Timestamp: baseTime.Add(60 * time.Minute), Component: "webserver", Name: "response_time", Value: 162.8, Type: "value"},
		
		// Database component metrics
		{Timestamp: baseTime, Component: "database", Name: "queries", Value: 50, Type: "value"},
		{Timestamp: baseTime.Add(45 * time.Minute), Component: "database", Name: "query_time", Value: 23.5, Type: "value"},
		{Timestamp: baseTime.Add(90 * time.Minute), Component: "database", Name: "connections", Value: 15, Type: "value"},
		
		// System component metrics
		{Timestamp: baseTime, Component: "system", Name: "cpu_percent", Value: 25.5, Type: "value"},
		{Timestamp: baseTime.Add(60 * time.Minute), Component: "system", Name: "cpu_percent", Value: 45.2, Type: "value"},
		{Timestamp: baseTime.Add(30 * time.Minute), Component: "system", Name: "memory_bytes", Value: 1024000, Type: "value"},
		{Timestamp: baseTime.Add(90 * time.Minute), Component: "system", Name: "goroutines", Value: 25, Type: "value"},
	}
	
	// Queue raw metrics for processing
	err := manager.PersistMetrics(testMetrics)
	if err != nil {
		return fmt.Errorf("PersistMetrics failed: %w", err)
	}
	
	// Force flush to ensure data is processed through universal queue
	// This is necessary because queue processes data asynchronously
	err = manager.ForceFlush()
	if err != nil {
		return fmt.Errorf("ForceFlush failed: %w", err)
	}
	
	return nil
}

func TestExtractMetricsByTimeRange(t *testing.T) {
	// Create memory backend for testing
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	// Create mock admin state
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	// Setup test data
	if err := setupTestData(manager); err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}
	t.Logf("Debug: Setup test data completed")
	
	// Test extracting metrics for webserver component
	start := time.Now().Add(-3 * time.Hour)
	end := time.Now()
	
	t.Logf("Debug: Query range from %v to %v", start, end)
	
	// Debug: Check if we can read metrics directly from manager
	metrics, debugErr := manager.ReadMetrics("webserver", start, end)
	if debugErr != nil {
		t.Fatalf("Debug ReadMetrics failed: %v", debugErr)
	}
	t.Logf("Debug: Found %d metrics for webserver", len(metrics))
	for i, m := range metrics {
		t.Logf("Debug: webserver metric %d: name=%s, type=%s, value=%v", i, m.Name, m.Type, m.Value)
	}
	
	// Debug: Try reading all components
	allMetrics, debugErr2 := manager.ReadMetrics("", start, end)
	if debugErr2 != nil {
		t.Fatalf("Debug ReadMetrics (all) failed: %v", debugErr2)
	}
	t.Logf("Debug: Found %d total metrics", len(allMetrics))
	
	result, err := ExtractMetricsByTimeRange(admin, "webserver", start, end)
	if err != nil {
		t.Fatalf("ExtractMetricsByTimeRange failed: %v", err)
	}
	
	// Verify JSON structure
	var componentData ComponentMetrics
	if err := json.Unmarshal([]byte(result), &componentData); err != nil {
		t.Fatalf("Failed to unmarshal result JSON: %v", err)
	}
	
	// Verify component name
	if componentData.Component != "webserver" {
		t.Errorf("Expected component 'webserver', got '%s'", componentData.Component)
	}
	
	// Verify we have metrics (should have 3 webserver metrics from test data)
	if len(componentData.Metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(componentData.Metrics))
	}
	
	// Verify metrics are sorted by timestamp
	for i := 1; i < len(componentData.Metrics); i++ {
		if componentData.Metrics[i].Timestamp.Before(componentData.Metrics[i-1].Timestamp) {
			t.Error("Metrics are not sorted by timestamp")
		}
	}
	
	// Verify specific metric content (all metrics are now value metrics)
	foundHTTPRequests := false
	foundResponseTime := false
	for _, metric := range componentData.Metrics {
		if metric.Name == "http_requests" && metric.Type == "value" {
			foundHTTPRequests = true
		}
		if metric.Name == "response_time" && metric.Type == "value" {
			foundResponseTime = true
		}
	}
	
	if !foundHTTPRequests {
		t.Error("Expected to find 'http_requests' value metric")
	}
	if !foundResponseTime {
		t.Error("Expected to find 'response_time' value metric")
	}
}

func TestExtractMetricsByTimeRange_NonExistentComponent(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	
	result, err := ExtractMetricsByTimeRange(admin, "nonexistent", start, end)
	if err != nil {
		t.Fatalf("ExtractMetricsByTimeRange failed: %v", err)
	}
	
	// Should return empty metrics for non-existent component
	var componentData ComponentMetrics
	if err := json.Unmarshal([]byte(result), &componentData); err != nil {
		t.Fatalf("Failed to unmarshal result JSON: %v", err)
	}
	
	if len(componentData.Metrics) != 0 {
		t.Errorf("Expected 0 metrics for non-existent component, got %d", len(componentData.Metrics))
	}
}

func TestExportAllMetrics(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	if err := setupTestData(manager); err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}
	
	start := time.Now().Add(-3 * time.Hour)
	end := time.Now()
	
	result, err := ExportAllMetrics(admin, start, end, "json")
	if err != nil {
		t.Fatalf("ExportAllMetrics failed: %v", err)
	}
	
	// Verify JSON structure
	var exportData AllMetricsExport
	if err := json.Unmarshal([]byte(result), &exportData); err != nil {
		t.Fatalf("Failed to unmarshal result JSON: %v", err)
	}
	
	// Verify time range
	if exportData.StartTime.After(start.Add(time.Minute)) || exportData.EndTime.Before(end.Add(-time.Minute)) {
		t.Error("Time range in export doesn't match requested range")
	}
	
	// Verify we have multiple components (Global, webserver, database, system)
	if len(exportData.Components) < 4 {
		t.Errorf("Expected at least 4 components, got %d", len(exportData.Components))
	}
	
	// Verify summary information
	if exportData.Summary.TotalComponents != len(exportData.Components) {
		t.Errorf("Summary component count mismatch: expected %d, got %d", 
			len(exportData.Components), exportData.Summary.TotalComponents)
	}
	
	if exportData.Summary.TotalMetrics <= 0 {
		t.Error("Expected positive total metrics in summary")
	}
	
	// Verify component names
	componentNames := make(map[string]bool)
	for _, comp := range exportData.Components {
		componentNames[comp.Component] = true
	}
	
	expectedComponents := []string{"Global", "webserver", "database", "system"}
	for _, expected := range expectedComponents {
		if !componentNames[expected] {
			t.Errorf("Expected component '%s' not found in export", expected)
		}
	}
}

func TestExportAllMetrics_UnsupportedFormat(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	
	_, err := ExportAllMetrics(admin, start, end, "xml")
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
	
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}

func TestListAvailableComponents(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	if err := setupTestData(manager); err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}
	
	components, err := ListAvailableComponents(admin)
	if err != nil {
		t.Fatalf("ListAvailableComponents failed: %v", err)
	}
	
	// Verify we have expected components
	expectedComponents := []string{"Global", "database", "system", "webserver"}
	if len(components) != len(expectedComponents) {
		t.Errorf("Expected %d components, got %d", len(expectedComponents), len(components))
	}
	
	// Verify components are sorted alphabetically
	for i := 1; i < len(components); i++ {
		if components[i] < components[i-1] {
			t.Error("Components are not sorted alphabetically")
		}
	}
	
	// Verify specific components exist
	componentSet := make(map[string]bool)
	for _, comp := range components {
		componentSet[comp] = true
	}
	
	for _, expected := range expectedComponents {
		if !componentSet[expected] {
			t.Errorf("Expected component '%s' not found in list", expected)
		}
	}
}

func TestGetHealthSummary(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	if err := setupTestData(manager); err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}
	
	start := time.Now().Add(-3 * time.Hour)
	end := time.Now()
	
	result, err := GetHealthSummary(admin, start, end)
	if err != nil {
		t.Fatalf("GetHealthSummary failed: %v", err)
	}
	
	// Verify JSON structure
	var healthSummary HealthSummary
	if err := json.Unmarshal([]byte(result), &healthSummary); err != nil {
		t.Fatalf("Failed to unmarshal result JSON: %v", err)
	}
	
	// Verify time range
	if healthSummary.StartTime.After(start.Add(time.Minute)) || healthSummary.EndTime.Before(end.Add(-time.Minute)) {
		t.Error("Time range in summary doesn't match requested range")
	}
	
	// Verify we have component summaries
	if len(healthSummary.Components) < 4 {
		t.Errorf("Expected at least 4 component summaries, got %d", len(healthSummary.Components))
	}
	
	// Verify overall summary
	if healthSummary.OverallSummary.TotalComponents != len(healthSummary.Components) {
		t.Error("Overall summary component count mismatch")
	}
	
	if healthSummary.OverallSummary.TotalMetrics <= 0 {
		t.Error("Expected positive total metrics in overall summary")
	}
	
	// Verify system metrics summary exists (since we have system component)
	if healthSummary.SystemMetrics == nil {
		t.Error("Expected system metrics summary but got nil")
	}
	
	// Verify system metrics contain expected metrics
	if healthSummary.SystemMetrics.CPUPercent == nil {
		t.Error("Expected CPU percent summary in system metrics")
	}
	
	if healthSummary.SystemMetrics.MemoryBytes == nil {
		t.Error("Expected memory bytes summary in system metrics")
	}
	
	// Verify component summaries have correct structure
	var foundWebserver bool
	for _, compSummary := range healthSummary.Components {
		if compSummary.Component == "webserver" {
			foundWebserver = true
			
			// Should have value metrics (all metrics are now values)
			if compSummary.Values == nil {
				t.Error("Expected webserver to have value metrics")
			}
			
			// Check specific metrics - all are now value metrics with statistical aggregation
			if _, exists := compSummary.Values["http_requests"]; !exists {
				t.Error("Expected 'http_requests' value in webserver summary")
			}
			
			if _, exists := compSummary.Values["response_time"]; !exists {
				t.Error("Expected 'response_time' value in webserver summary")
			}
		}
	}
	
	if !foundWebserver {
		t.Error("Expected to find webserver component in health summary")
	}
}

func TestGetHealthSummary_SystemHealthIndicator(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	// Create system metrics with high values to trigger unhealthy status
	baseTime := time.Now().Add(-1 * time.Hour)
	highResourceMetrics := []storage.MetricEntry{
		{Timestamp: baseTime, Component: "system", Name: "cpu_percent", Value: 95.0, Type: "value"},
		{Timestamp: baseTime.Add(30 * time.Minute), Component: "system", Name: "memory_bytes", Value: 2000000000, Type: "value"}, // 2GB
	}
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	// Use manager to queue and process the metrics properly
	if err := manager.PersistMetrics(highResourceMetrics); err != nil {
		t.Fatalf("Failed to persist high resource test data: %v", err)
	}
	
	// Force flush to ensure data is processed
	if err := manager.ForceFlush(); err != nil {
		t.Fatalf("Failed to flush high resource test data: %v", err)
	}
	
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()
	
	result, err := GetHealthSummary(admin, start, end)
	if err != nil {
		t.Fatalf("GetHealthSummary failed: %v", err)
	}
	
	var healthSummary HealthSummary
	if err := json.Unmarshal([]byte(result), &healthSummary); err != nil {
		t.Fatalf("Failed to unmarshal result JSON: %v", err)
	}
	
	// System should be marked as unhealthy due to high resource usage
	if healthSummary.OverallSummary.SystemHealthy {
		t.Error("Expected system to be marked unhealthy due to high CPU/memory usage")
	}
}

func TestPersistenceDisabled(t *testing.T) {
	// Test behavior when persistence is disabled
	manager := storage.NewManager(nil, false)
	admin := &MockAdminState{manager: manager}
	
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	
	// All functions should return appropriate errors when persistence is disabled
	_, err := ExtractMetricsByTimeRange(admin, "test", start, end)
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("Expected 'persistence not enabled' error for ExtractMetricsByTimeRange")
	}
	
	_, err = ExportAllMetrics(admin, start, end, "json")
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("Expected 'persistence not enabled' error for ExportAllMetrics")
	}
	
	_, err = ListAvailableComponents(admin)
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("Expected 'persistence not enabled' error for ListAvailableComponents")
	}
	
	_, err = GetHealthSummary(admin, start, end)
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("Expected 'persistence not enabled' error for GetHealthSummary")
	}
}

// Benchmark tests to ensure good performance
func BenchmarkExtractMetricsByTimeRange(b *testing.B) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	setupTestData(manager)
	
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractMetricsByTimeRange(admin, "webserver", start, end)
		if err != nil {
			b.Fatalf("ExtractMetricsByTimeRange failed: %v", err)
		}
	}
}

func BenchmarkGetHealthSummary(b *testing.B) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	setupTestData(manager)
	
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetHealthSummary(admin, start, end)
		if err != nil {
			b.Fatalf("GetHealthSummary failed: %v", err)
		}
	}
}

// Test JSON output structure and field validation
func TestJSONOutputStructure(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	manager := storage.NewManager(backend, true)
	admin := &MockAdminState{manager: manager}
	
	if err := setupTestData(manager); err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}
	
	start := time.Now().Add(-3 * time.Hour)
	end := time.Now()
	
	// Test health summary JSON structure
	summaryJSON, err := GetHealthSummary(admin, start, end)
	if err != nil {
		t.Fatalf("GetHealthSummary failed: %v", err)
	}
	
	// Verify JSON is valid and properly formatted
	if !json.Valid([]byte(summaryJSON)) {
		t.Error("Health summary JSON is not valid")
	}
	
	// Check that JSON is pretty-printed (contains newlines and indentation)  
	if !strings.Contains(summaryJSON, "\n") || !strings.Contains(summaryJSON, "  ") {
		t.Error("JSON output should be pretty-printed with indentation")
	}
	
	// Verify all required fields are present
	requiredFields := []string{
		"start_time", "end_time", "components", "overall_summary",
		"total_components", "total_metrics", "system_healthy",
	}
	
	for _, field := range requiredFields {
		if !strings.Contains(summaryJSON, fmt.Sprintf("\"%s\"", field)) {
			t.Errorf("Required field '%s' not found in JSON output", field)
		}
	}
}