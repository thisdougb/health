package health

import (
	"strings"
	"testing"
)

func TestInfoMethodSetters(t *testing.T) {
	// Test setting the identity.
	identity := "workerXYZ"

	s := NewState()
	s.SetConfig(identity)
	result := s.Dump()

	searchFor := "\"Identity\": \"" + identity + "\","
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Info failed to set Identity")
	}
}

func TestInfoMethodSetterDefaults(t *testing.T) {
	// Test setting the identity uses defaults when no values are supplied.
	identity := ""

	s := NewState()
	s.SetConfig(identity)
	result := s.Dump()

	searchFor := "\"Identity\": \"identity unset\","
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Info failed to set default Identity")
	}
}

func TestIncrMetric(t *testing.T) {
	// Test incrementing metrics.
	s := NewState()
	s.SetConfig("test-node")

	s.IncrMetric("requestsRecvd")
	s.IncrMetric("requestsRecvd")
	s.IncrMetric("requestsRecvd")

	result := s.Dump()

	searchFor := "\"requestsRecvd\": 3"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("IncrMetric failed to increment to expected value")
	}
}

func TestIncrMetricIgnoresEmptyName(t *testing.T) {
	// Test empty metric names are ignored
	s := NewState()
	s.SetConfig("test-node")

	s.IncrMetric("")
	result := s.Dump()

	// Should not contain any empty-named metrics
	if strings.Contains(result, "\"\": ") {
		t.Errorf("IncrMetric should ignore empty metric names")
	}
}

func TestAddMetric(t *testing.T) {
	// Test adding raw metric values
	s := NewState()
	s.SetConfig("test-node")

	// Add some values - they should be persisted to storage backend
	s.AddMetric("response_time", 125.5)
	s.AddMetric("cpu_usage", 45.2)

	// Verify state still works for in-memory data
	result := s.Dump()
	if result == "" {
		t.Errorf("Dump should return non-empty JSON")
	}

	// Values are persisted asynchronously to storage, not in the JSON dump
	// The Dump() now only shows counter metrics, not raw values
}

func TestComponentMetrics(t *testing.T) {
	// Test component-based metrics
	s := NewState()
	s.SetConfig("test-node")

	// Test component counters
	s.IncrComponentMetric("webserver", "requests")
	s.IncrComponentMetric("webserver", "requests")
	s.IncrComponentMetric("database", "queries")

	// Test component values
	s.AddComponentMetric("webserver", "response_time", 123.4)
	s.AddComponentMetric("database", "query_time", 56.7)

	result := s.Dump()

	// Check for component-based counter metrics
	if !strings.Contains(result, "\"webserver\"") {
		t.Errorf("Component metrics should include webserver component")
	}

	if !strings.Contains(result, "\"database\"") {
		t.Errorf("Component metrics should include database component")
	}
}

func TestJSONStructure(t *testing.T) {
	// Test the overall JSON structure
	s := NewState()
	s.SetConfig("test-structure")

	s.IncrMetric("global_counter")
	s.IncrComponentMetric("web", "requests")

	result := s.Dump()

	// Should contain basic fields
	expectedFields := []string{
		"\"Identity\":",
		"\"Started\":",
		"\"Metrics\":",
	}

	for _, field := range expectedFields {
		if !strings.Contains(result, field) {
			t.Errorf("JSON output missing expected field: %s", field)
		}
	}
}

func TestClose(t *testing.T) {
	// Test that Close method works
	s := NewState()
	s.SetConfig("test-close")

	err := s.Close()
	if err != nil {
		t.Errorf("Close() should not return error: %v", err)
	}

	// Should be safe to call multiple times
	err = s.Close()
	if err != nil {
		t.Errorf("Multiple Close() calls should not return error: %v", err)
	}
}