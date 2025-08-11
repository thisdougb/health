package health

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/thisdougb/health/internal/config"
)

// TestTimeWindowedMetricsPhase1 tests the basic time-windowed metrics collection
func TestTimeWindowedMetricsPhase1(t *testing.T) {
	// Test with default 60-second windows
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-time-windowed")

	// Test counter metrics (should show count in current window)
	state.IncrMetric("web_requests")
	state.IncrMetric("web_requests")
	state.IncrMetric("web_requests")
	
	state.IncrComponentMetric("database", "queries")
	state.IncrComponentMetric("database", "queries")

	// Test value metrics (should show min/max/avg statistics)
	state.AddMetric("response_time", 145.2)
	state.AddMetric("response_time", 162.8)
	state.AddMetric("response_time", 134.1)
	
	state.AddComponentMetric("database", "query_time", 23.1)
	state.AddComponentMetric("database", "query_time", 45.7)

	// Get JSON output
	jsonOutput := state.Dump()
	
	// Parse JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify basic structure
	if result["Identity"] != "test-time-windowed" {
		t.Errorf("Expected Identity 'test-time-windowed', got %v", result["Identity"])
	}

	metrics, ok := result["Metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("Metrics field is not a map")
	}

	// Verify Global metrics
	globalMetrics, ok := metrics["Global"].(map[string]interface{})
	if !ok {
		t.Fatal("Global metrics not found")
	}

	// Counter metric should show count
	if globalMetrics["web_requests"] != float64(3) {
		t.Errorf("Expected web_requests count of 3, got %v", globalMetrics["web_requests"])
	}

	// Value metric should show statistics
	responseTimeStats, ok := globalMetrics["response_time"].(map[string]interface{})
	if !ok {
		t.Fatal("response_time should be a statistics object")
	}
	if responseTimeStats["count"] != float64(3) {
		t.Errorf("Expected response_time count of 3, got %v", responseTimeStats["count"])
	}
	if responseTimeStats["min"] != 134.1 {
		t.Errorf("Expected response_time min of 134.1, got %v", responseTimeStats["min"])
	}
	if responseTimeStats["max"] != 162.8 {
		t.Errorf("Expected response_time max of 162.8, got %v", responseTimeStats["max"])
	}

	// Verify database component metrics
	dbMetrics, ok := metrics["database"].(map[string]interface{})
	if !ok {
		t.Fatal("Database metrics not found")
	}

	if dbMetrics["queries"] != float64(2) {
		t.Errorf("Expected queries count of 2, got %v", dbMetrics["queries"])
	}

	queryTimeStats, ok := dbMetrics["query_time"].(map[string]interface{})
	if !ok {
		t.Fatal("query_time should be a statistics object")
	}
	if queryTimeStats["count"] != float64(2) {
		t.Errorf("Expected query_time count of 2, got %v", queryTimeStats["count"])
	}

	t.Logf("JSON Output:\n%s", jsonOutput)
}

// TestTimeWindowKeyGeneration tests the getCurrentTimeKey function with different sample rates
func TestTimeWindowKeyGeneration(t *testing.T) {
	// Test with default 60-second sample rate
	originalSampleRate := os.Getenv("HEALTH_SAMPLE_RATE")
	defer func() {
		if originalSampleRate != "" {
			os.Setenv("HEALTH_SAMPLE_RATE", originalSampleRate)
		} else {
			os.Unsetenv("HEALTH_SAMPLE_RATE")
		}
	}()

	// Test 60-second windows (default)
	os.Setenv("HEALTH_SAMPLE_RATE", "60")
	
	state := NewState()
	defer state.Close()
	
	state.IncrMetric("test_metric")
	
	// Add another metric immediately - both should be in same window
	state.IncrMetric("test_metric")
	
	jsonOutput := state.Dump()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	metrics := result["Metrics"].(map[string]interface{})
	globalMetrics := metrics["Global"].(map[string]interface{})
	
	// Both increments should be in the same window
	if globalMetrics["test_metric"] != float64(2) {
		t.Errorf("Expected test_metric count of 2 in same window, got %v", globalMetrics["test_metric"])
	}
}

// TestConfigurableTimeWindows tests that time windows respect the HEALTH_SAMPLE_RATE configuration
func TestConfigurableTimeWindows(t *testing.T) {
	originalSampleRate := os.Getenv("HEALTH_SAMPLE_RATE")
	defer func() {
		if originalSampleRate != "" {
			os.Setenv("HEALTH_SAMPLE_RATE", originalSampleRate)
		} else {
			os.Unsetenv("HEALTH_SAMPLE_RATE")
		}
	}()

	// Test with 30-second windows
	os.Setenv("HEALTH_SAMPLE_RATE", "30")
	
	// Verify config value is read correctly
	sampleRate := config.IntValue("HEALTH_SAMPLE_RATE")
	if sampleRate != 30 {
		t.Errorf("Expected HEALTH_SAMPLE_RATE of 30, got %d", sampleRate)
	}
	
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-30s-windows")
	state.IncrMetric("test_metric")
	
	jsonOutput := state.Dump()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify the metric was recorded
	metrics := result["Metrics"].(map[string]interface{})
	globalMetrics := metrics["Global"].(map[string]interface{})
	
	if globalMetrics["test_metric"] != float64(1) {
		t.Errorf("Expected test_metric count of 1, got %v", globalMetrics["test_metric"])
	}
}

// TestEmptyTimeWindows tests behavior when no metrics are recorded in current window
func TestEmptyTimeWindows(t *testing.T) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-empty")
	
	// Don't add any metrics
	jsonOutput := state.Dump()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify basic structure exists but no metrics
	if result["Identity"] != "test-empty" {
		t.Errorf("Expected Identity 'test-empty', got %v", result["Identity"])
	}

	metrics, ok := result["Metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("Metrics field should exist as empty map")
	}

	// Should have empty metrics map
	if len(metrics) != 0 {
		t.Errorf("Expected empty metrics map, got %d entries", len(metrics))
	}
}