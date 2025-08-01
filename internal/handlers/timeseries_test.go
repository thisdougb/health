package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thisdougb/health/internal/storage"
)

// TestState implements StateInterface using real storage
type TestState struct {
	manager *storage.Manager
}

func (t *TestState) Dump() string {
	return `{"test": "data"}`
}

func (t *TestState) GetStorageManager() *storage.Manager {
	return t.manager
}

func setupTestStorage(t *testing.T) (*TestState, func()) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "health_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create SQLite backend
	dbPath := filepath.Join(tmpDir, "test.db")
	config := storage.SQLiteConfig{
		DBPath:        dbPath,
		FlushInterval: 100 * time.Millisecond, // Fast flush for tests
		BatchSize:     10,
	}

	backend, err := storage.NewSQLiteBackend(config)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}

	manager := storage.NewManager(backend, true)
	state := &TestState{manager: manager}

	// Cleanup function
	cleanup := func() {
		manager.Close()
		os.RemoveAll(tmpDir)
	}

	return state, cleanup
}

func addTestMetrics(t *testing.T, state *TestState, component string, baseTime time.Time) {
	metrics := []storage.MetricEntry{
		{
			Timestamp: baseTime.Add(-30 * time.Minute),
			Component: component,
			Name:      "requests",
			Value:     100.0,
			Type:      "value",
		},
		{
			Timestamp: baseTime.Add(-20 * time.Minute),
			Component: component,
			Name:      "requests",
			Value:     150.0,
			Type:      "value",
		},
		{
			Timestamp: baseTime.Add(-10 * time.Minute),
			Component: component,
			Name:      "requests",
			Value:     200.0,
			Type:      "value",
		},
		{
			Timestamp: baseTime.Add(10 * time.Minute),
			Component: component,
			Name:      "requests",
			Value:     250.0,
			Type:      "value",
		},
		{
			Timestamp: baseTime.Add(20 * time.Minute),
			Component: component,
			Name:      "requests",
			Value:     300.0,
			Type:      "value",
		},
	}

	err := state.manager.PersistMetrics(metrics)
	if err != nil {
		t.Fatalf("Failed to write test metrics: %v", err)
	}

	// Force flush to ensure metrics are written to SQLite (deterministic)
	err = state.manager.ForceFlush()
	if err != nil {
		t.Fatalf("Failed to force flush: %v", err)
	}
}

func TestTimeSeriesHandler_Lookback(t *testing.T) {
	state, cleanup := setupTestStorage(t)
	defer cleanup()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	addTestMetrics(t, state, "webserver", baseTime)

	handler := TimeSeriesHandler(state, "webserver")

	// Test lookback query
	req := httptest.NewRequest("GET", "/health/webserver?window=10m&lookback=1h&date=2025-01-15&time=10:00:00", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response TimeSeriesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	if response.Component != "webserver" {
		t.Errorf("Expected component 'webserver', got %s", response.Component)
	}
	if response.Direction != "lookback" {
		t.Errorf("Expected direction 'lookback', got %s", response.Direction)
	}
	if response.Window != "10m0s" {
		t.Errorf("Expected window '10m0s', got %s", response.Window)
	}
	if response.Duration != "1h0m0s" {
		t.Errorf("Expected duration '1h0m0s', got %s", response.Duration)
	}

	// Verify we have metrics data
	if len(response.Metrics) == 0 {
		t.Error("Expected metrics data, got empty map")
	}

	// Should have 'requests' metric
	if _, exists := response.Metrics["requests"]; !exists {
		t.Error("Expected 'requests' metric in response")
	}
}

func TestTimeSeriesHandler_Lookahead(t *testing.T) {
	state, cleanup := setupTestStorage(t)
	defer cleanup()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	addTestMetrics(t, state, "database", baseTime)

	handler := TimeSeriesHandler(state, "database")

	// Test lookahead query
	req := httptest.NewRequest("GET", "/health/database?window=5m&lookahead=2h&date=2025-01-15&time=10:00:00", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response TimeSeriesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Direction != "lookahead" {
		t.Errorf("Expected direction 'lookahead', got %s", response.Direction)
	}

	if response.Component != "database" {
		t.Errorf("Expected component 'database', got %s", response.Component)
	}

	// Should have future data points
	if len(response.Metrics) == 0 {
		t.Error("Expected metrics data, got empty map")
	}
}

func TestTimeSeriesHandler_MutuallyExclusive(t *testing.T) {
	state, cleanup := setupTestStorage(t)
	defer cleanup()

	handler := TimeSeriesHandler(state, "test")

	// Test both lookback and lookahead (should fail)
	req := httptest.NewRequest("GET", "/health/test?window=5m&lookback=1h&lookahead=2h", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "mutually exclusive") {
		t.Error("Expected error message about mutual exclusivity")
	}
}

func TestTimeSeriesHandler_MissingDirection(t *testing.T) {
	state, cleanup := setupTestStorage(t)
	defer cleanup()

	handler := TimeSeriesHandler(state, "test")

	// Test neither lookback nor lookahead (should fail)
	req := httptest.NewRequest("GET", "/health/test?window=5m", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "either lookback or lookahead must be specified") {
		t.Error("Expected error message about missing direction")
	}
}

func TestTimeSeriesHandler_MissingWindow(t *testing.T) {
	state, cleanup := setupTestStorage(t)
	defer cleanup()

	handler := TimeSeriesHandler(state, "test")

	// Test missing window parameter (should fail)
	req := httptest.NewRequest("GET", "/health/test?lookback=1h", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "window parameter is required") {
		t.Error("Expected error message about missing window parameter")
	}
}

func TestTimeSeriesHandler_DisabledPersistence(t *testing.T) {
	// Create state with disabled persistence
	manager := storage.NewManager(nil, false)
	state := &TestState{manager: manager}

	handler := TimeSeriesHandler(state, "test")

	req := httptest.NewRequest("GET", "/health/test?window=5m&lookback=1h", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "persistence") {
		t.Error("Expected error message about persistence not enabled")
	}
}

func TestParseTimeSeriesParams(t *testing.T) {
	// Test valid parameters
	req := httptest.NewRequest("GET", "/test?window=5m&lookback=2h&date=2025-01-15&time=14:30:45", nil)
	params, err := parseTimeSeriesParams(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if params.Window != 5*time.Minute {
		t.Errorf("Expected window 5m, got %v", params.Window)
	}
	if params.Lookback == nil || *params.Lookback != 2*time.Hour {
		t.Errorf("Expected lookback 2h, got %v", params.Lookback)
	}
	if params.Date == nil {
		t.Error("Expected date to be set")
	}
	if params.Time == nil {
		t.Error("Expected time to be set")
	}

	// Test time without seconds
	req2 := httptest.NewRequest("GET", "/test?window=1m&lookahead=30m&time=09:15", nil)
	params2, err := parseTimeSeriesParams(req2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if params2.Time == nil {
		t.Error("Expected time to be parsed")
	}

	// Test invalid time format
	req3 := httptest.NewRequest("GET", "/test?window=1m&lookback=1h&time=25:00:00", nil)
	_, err = parseTimeSeriesParams(req3)
	if err == nil {
		t.Error("Expected error for invalid hour")
	}
}

func TestCalculateReferenceTime(t *testing.T) {
	testDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	testTime := time.Date(2000, 1, 1, 14, 30, 45, 0, time.UTC)

	params := &TimeSeriesParams{
		Date: &testDate,
		Time: &testTime,
	}

	result := calculateReferenceTime(params)
	expected := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)

	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestAggregateMetricsByWindow(t *testing.T) {
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	metrics := []storage.MetricEntry{
		{
			Timestamp: baseTime,
			Name:      "requests",
			Value:     100.0,
		},
		{
			Timestamp: baseTime.Add(2 * time.Minute),
			Name:      "requests",
			Value:     200.0,
		},
		{
			Timestamp: baseTime.Add(7 * time.Minute),
			Name:      "requests",
			Value:     150.0,
		},
	}

	result := aggregateMetricsByWindow(metrics, 5*time.Minute)

	if _, exists := result["requests"]; !exists {
		t.Error("Expected 'requests' metric in result")
	}

	// Should have two time windows: 10:00:00 and 10:05:00
	requestsData, ok := result["requests"].(map[string]float64)
	if !ok {
		t.Error("Expected requests data to be map[string]float64")
	}

	if len(requestsData) != 2 {
		t.Errorf("Expected 2 time windows, got %d", len(requestsData))
	}

	// First window should average 100 and 200
	if val, exists := requestsData["10:00:00"]; !exists || val != 150.0 {
		t.Errorf("Expected first window average 150.0, got %v", val)
	}

	// Second window should have 150
	if val, exists := requestsData["10:05:00"]; !exists || val != 150.0 {
		t.Errorf("Expected second window average 150.0, got %v", val)
	}
}