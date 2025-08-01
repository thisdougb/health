package health

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestFullWorkflowMemoryBackend tests the complete workflow using memory backend
// This test is ideal for junior developers to understand the basic flow:
// 1. Create state
// 2. Add metrics 
// 3. Export data
// 4. Clean shutdown
func TestFullWorkflowMemoryBackend(t *testing.T) {
	// Step 1: Create a new health state instance with memory backend
	// Memory backend means data is only stored in RAM and lost when program stops
	// This is perfect for development and testing
	state := NewState() // Uses memory backend by default
	defer state.Close() // Always clean up resources when test finishes

	// Step 2: Configure the instance with a unique identifier
	// This identifier will appear in JSON output to identify which service
	// the metrics came from - useful in microservice architectures
	state.SetConfig("test-memory-workflow")

	// Step 3: Add different types of metrics
	// Counter metrics: These count how many times something happened
	state.IncrMetric("total_requests")           // Global counter
	state.IncrMetric("total_requests")           // Increment again (now = 2)
	state.IncrComponentMetric("web", "requests") // Component-specific counter
	state.IncrComponentMetric("api", "calls")    // Different component

	// Raw value metrics: These store actual measurement values for analysis
	// These are NOT shown in JSON output - they go to storage for historical analysis
	state.AddMetric("response_time", 145.7)                    // Global measurement
	state.AddComponentMetric("database", "query_time", 23.4)   // Component measurement
	state.AddComponentMetric("cache", "hit_rate", 0.87)        // Another measurement

	// Step 4: Export current state as JSON
	// This is what gets returned to monitoring systems and health checks
	jsonOutput := state.Dump()

	// Step 5: Verify the JSON structure
	// For junior developers: JSON structure is designed for easy consumption
	// by monitoring tools like Prometheus, Grafana, or custom dashboards
	if jsonOutput == "" {
		t.Fatal("Expected JSON output, got empty string")
	}

	// Parse JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify required fields exist
	if result["Identity"] != "test-memory-workflow" {
		t.Errorf("Expected Identity 'test-memory-workflow', got %v", result["Identity"])
	}

	if result["Started"] == nil {
		t.Error("Expected Started timestamp, got nil")
	}

	// Verify metrics structure - should be organized by component
	metrics, ok := result["Metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Metrics to be an object")
	}

	// Check Global metrics (counter metrics only)
	globalMetrics, ok := metrics["Global"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Global metrics section")
	}

	if globalMetrics["total_requests"] != float64(2) { // JSON numbers are float64
		t.Errorf("Expected total_requests = 2, got %v", globalMetrics["total_requests"])
	}

	// Check component-specific metrics
	webMetrics, ok := metrics["web"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected web component metrics")
	}

	if webMetrics["requests"] != float64(1) {
		t.Errorf("Expected web requests = 1, got %v", webMetrics["requests"])
	}

	// Important: Raw values (response_time, query_time, hit_rate) should NOT appear in JSON
	// They are stored in the backend for historical analysis, not real-time display
	if _, exists := globalMetrics["response_time"]; exists {
		t.Error("Raw values should not appear in JSON output - they go to storage backend")
	}
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// TestFullWorkflowSQLiteBackend tests the complete workflow with SQLite persistence
// This demonstrates how to use the package in production with data persistence
func TestFullWorkflowSQLiteBackend(t *testing.T) {
	// Step 1: Set up temporary SQLite database for testing
	// In production, this would be a permanent file path like /data/health.db
	tmpDir := t.TempDir() // Automatically cleaned up after test
	tmpFile := tmpDir + "/health_integration_test.db"
	backupDir := tmpDir + "/backups"

	// Step 2: Configure environment variables for SQLite backend
	// This is how you enable persistence in production - no code changes needed!
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_FLUSH_INTERVAL", "1s")  // Flush to database every 1 second
	os.Setenv("HEALTH_BATCH_SIZE", "50")      // Batch up to 50 metrics before writing
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")   // Enable automatic backups
	os.Setenv("HEALTH_BACKUP_DIR", backupDir)   // Where to store backup files  
	os.Setenv("HEALTH_BACKUP_RETENTION_DAYS", "7") // Keep backups for 7 days // Keep backups for 7 days

	// Clean up environment after test
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_FLUSH_INTERVAL")
		os.Unsetenv("HEALTH_BATCH_SIZE")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
		os.Unsetenv("HEALTH_BACKUP_RETENTION_DAYS")
	}()

	// Step 3: Create state - automatically detects SQLite configuration from environment
	state := NewState()
	defer state.Close() // This will trigger backup creation if enabled

	state.SetConfig("test-sqlite-workflow")

	// Step 4: Add various metrics over time
	// In a real application, these would be called throughout your application lifecycle
	for i := 0; i < 10; i++ {
		state.IncrMetric("requests")
		state.IncrComponentMetric("webserver", "http_requests")
		state.AddMetric("response_time", float64(100+i*10)) // Varying response times
		state.AddComponentMetric("database", "query_time", float64(20+i*2))

		// Small delay to simulate real application timing
		time.Sleep(10 * time.Millisecond)
	}

	// Step 5: Allow time for async persistence to write to database
	// The package uses background processing to avoid blocking your application
	time.Sleep(2 * time.Second)

	// Step 6: Verify SQLite database was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("SQLite database file should have been created")
	}

	// Step 7: Test data extraction functions (useful for monitoring and analysis)
	// These functions let you extract historical data for analysis
	start := time.Now().Add(-10 * time.Minute) // Last 10 minutes
	end := time.Now()

	// List all components that have data
	components, err := handlers.ListAvailableComponents(state)
	if err != nil {
		t.Fatalf("Failed to list components: %v", err)
	}

	if len(components) == 0 {
		t.Error("Expected at least one component, got none")
	}

	// Extract metrics for webserver component
	webData, err := handlers.ExtractMetricsByTimeRange(state, "webserver", start, end)
	if err != nil {
		t.Fatalf("Failed to extract webserver metrics: %v", err)
	}

	if webData == "" {
		t.Error("Expected webserver data, got empty string")
	}

	// Get overall health summary
	summary, err := handlers.GetHealthSummary(state, start, end)
	if err != nil {
		t.Fatalf("Failed to get health summary: %v", err)
	}

	if summary == "" {
		t.Error("Expected health summary, got empty string")
	}

	// Step 8: Verify JSON output still works (real-time counter data)
	jsonOutput := state.Dump()
	if jsonOutput == "" {
		t.Fatal("Expected JSON output, got empty string")
	}

	// Parse and verify counter values
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	metrics := result["Metrics"].(map[string]interface{})
	globalMetrics := metrics["Global"].(map[string]interface{})
	
	if globalMetrics["requests"] != float64(10) {
		t.Errorf("Expected 10 requests, got %v", globalMetrics["requests"])
	}
}
*/

// TestHTTPEndpointIntegration tests all HTTP endpoints with detailed explanations
// This shows junior developers how the package integrates with web servers
func TestHTTPEndpointIntegration(t *testing.T) {
	// Step 1: Create health state and add some test data
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-http-endpoints")
	
	// Add test metrics to have something to display
	state.IncrMetric("total_requests")
	state.IncrComponentMetric("webserver", "http_requests")
	state.IncrComponentMetric("database", "queries")
	state.AddMetric("response_time", 150.5)

	// Step 2: Test the main health endpoint
	// This endpoint returns JSON with all counter metrics
	req := httptest.NewRequest("GET", "/health/", nil)
	w := httptest.NewRecorder()
	
	// HandleHealthRequest automatically detects the URL pattern and responds appropriately
	state.HandleHealthRequest(w, req)
	
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	
	// Verify JSON structure
	if !strings.Contains(bodyStr, "test-http-endpoints") {
		t.Error("Response should contain instance identity")
	}
	
	if !strings.Contains(bodyStr, "total_requests") {
		t.Error("Response should contain global metrics")
	}

	// Step 3: Test component-specific endpoint
	// URL: /health/webserver returns only webserver component metrics
	req = httptest.NewRequest("GET", "/some/prefix/health/webserver", nil)
	w = httptest.NewRecorder()
	
	state.HandleHealthRequest(w, req)
	
	resp = w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for component endpoint, got %d", resp.StatusCode)
	}

	// Step 4: Test status endpoint
	// URL: /health/status returns simple OK/NOT OK status
	req = httptest.NewRequest("GET", "/health/status", nil)
	w = httptest.NewRecorder()
	
	state.HandleHealthRequest(w, req)
	
	resp = w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for status endpoint, got %d", resp.StatusCode)
	}

	// Step 5: Test component status endpoint
	req = httptest.NewRequest("GET", "/health/webserver/status", nil)
	w = httptest.NewRecorder()
	
	state.HandleHealthRequest(w, req)
	
	resp = w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for component status, got %d", resp.StatusCode)
	}

	// Step 6: Test with custom HTTP server (real-world usage pattern)
	// This shows how to integrate with your existing web server
	mux := http.NewServeMux()
	
	// Add authentication wrapper (common in production)
	mux.HandleFunc("/health/", func(w http.ResponseWriter, r *http.Request) {
		// In production, you might check API keys or other authentication here
		// For this test, we'll just pass through
		state.HandleHealthRequest(w, r)
	})
	
	// Test server
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Make real HTTP request
	resp, err := http.Get(server.URL + "/health/")
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 from test server, got %d", resp.StatusCode)
	}
}

// TestConcurrentAccess tests thread safety with multiple goroutines
// This is crucial for junior developers to understand - the package must handle
// multiple threads accessing it simultaneously without data corruption
func TestConcurrentAccess(t *testing.T) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-concurrent")
	
	// Number of goroutines to run simultaneously
	numGoroutines := 10
	// Number of operations each goroutine performs  
	opsPerGoroutine := 100
	
	// WaitGroup ensures we wait for all goroutines to complete
	var wg sync.WaitGroup
	
	// Launch multiple goroutines that increment metrics simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		
		// Each goroutine runs this function
		go func(goroutineID int) {
			defer wg.Done() // Mark this goroutine as complete when function exits
			
			// Each goroutine performs many metric operations
			for j := 0; j < opsPerGoroutine; j++ {
				// These operations happen simultaneously across all goroutines
				state.IncrMetric("concurrent_test")
				state.IncrComponentMetric("worker", fmt.Sprintf("goroutine_%d", goroutineID))
				state.AddMetric("operation_time", float64(j))
				
				// Occasionally export JSON (this tests read operations during writes)
				if j%10 == 0 {
					_ = state.Dump()
				}
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	// Verify final state - should have exact number of increments
	// If thread safety is working correctly, we should have:
	// - concurrent_test = numGoroutines * opsPerGoroutine
	jsonOutput := state.Dump()
	
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		t.Fatalf("Failed to parse JSON after concurrent access: %v", err)
	}
	
	metrics := result["Metrics"].(map[string]interface{})
	globalMetrics := metrics["Global"].(map[string]interface{})
	
	expectedCount := float64(numGoroutines * opsPerGoroutine)
	if globalMetrics["concurrent_test"] != expectedCount {
		t.Errorf("Expected concurrent_test = %v, got %v", expectedCount, globalMetrics["concurrent_test"])
		t.Error("This indicates a thread safety issue - operations were lost during concurrent access")
	}
	
	// Verify worker component has metrics from all goroutines
	workerMetrics := metrics["worker"].(map[string]interface{})
	if len(workerMetrics) != numGoroutines {
		t.Errorf("Expected %d worker metrics, got %d", numGoroutines, len(workerMetrics))
	}
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// TestSystemMetricsCollection tests automatic system metrics
// These metrics are collected automatically every minute to monitor application health
func TestSystemMetricsCollection(t *testing.T) {
	// Set up SQLite backend to actually store system metrics
	tmpDir := t.TempDir() // Automatically cleaned up after test
	tmpFile := tmpDir + "/health_system_test.db"
	
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_FLUSH_INTERVAL", "100ms") // Very fast for testing
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH") 
		os.Unsetenv("HEALTH_FLUSH_INTERVAL")
	}()
	
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-system-metrics")
	
	// System metrics are collected automatically in the background
	// We need to wait a bit for the first collection cycle
	time.Sleep(200 * time.Millisecond)
	
	// Add some application metrics to generate activity
	for i := 0; i < 50; i++ {
		state.IncrMetric("test_metric")
		state.AddMetric("test_value", float64(i))
	}
	
	// Wait for metrics to be persisted
	time.Sleep(500 * time.Millisecond)
	
	// System metrics don't appear in JSON (they go to storage backend)
	// But we can verify they were collected using the admin functions
	start := time.Now().Add(-1 * time.Minute)
	end := time.Now()
	
	// Extract system metrics
	systemData, err := handlers.ExtractMetricsByTimeRange(state, "system", start, end)
	if err != nil {
		t.Fatalf("Failed to extract system metrics: %v", err)
	}
	
	if systemData == "" {
		t.Error("Expected system metrics data, got empty string")
	}
	
	// Parse the system metrics to verify expected metrics are present
	var systemResult map[string]interface{}
	if err := json.Unmarshal([]byte(systemData), &systemResult); err != nil {
		t.Fatalf("Failed to parse system metrics JSON: %v", err)
	}
	
	// System metrics should include: cpu_percent, memory_bytes, goroutines, uptime_seconds
	// These are automatically collected every minute
	if systemResult["component"] != "system" {
		t.Error("Expected system component in extracted data")
	}
}
*/

// TestBackupAndRestore tests the backup functionality
// This is important for production deployments to prevent data loss
func TestBackupAndRestore(t *testing.T) {
	// Set up SQLite with backup enabled
	tmpDir := t.TempDir() // Automatically cleaned up after test
	tmpFile := tmpDir + "/health_backup_test.db"
	backupDir := tmpDir + "/backups"
	
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")
	os.Setenv("HEALTH_BACKUP_DIR", backupDir)
	os.Setenv("HEALTH_BACKUP_RETENTION_DAYS", "7") // Keep backups for 7 days
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
		os.Unsetenv("HEALTH_BACKUP_RETENTION_DAYS")
	}()
	
	// Step 1: Create state and add some data
	state := NewState()
	state.SetConfig("test-backup")
	
	// Add metrics that will be backed up - focus on raw values that go to database
	for i := 0; i < 20; i++ {
		state.IncrMetric("backup_test")
		state.AddMetric("backup_value", float64(i*10))
		state.AddComponentMetric("webserver", "response_time", float64(100+i*5))
	}
	
	// Step 2: Force flush the queue to ensure data is written to database
	// This is the deterministic way to ensure data exists before backup
	storageManager := state.GetStorageManager()
	
	// Force flush any queued metrics to the database
	// This ensures all AddMetric() calls above are actually written to the database file
	if err := storageManager.ForceFlush(); err != nil {
		t.Fatalf("Failed to flush queued metrics: %v", err)
	}
	
	// Step 3: Create backup now that data is in database
	if err := storageManager.CreateBackup(); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	
	// Clean up
	defer state.Close()
	
	// Step 4: Verify backup was created
	// Backup files are named with today's date: health_YYYYMMDD.db
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Fatal("Backup directory was not created")
	}
	
	// Check that backup file exists (we can't predict exact filename due to date)
	backupFiles, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	
	if len(backupFiles) == 0 {
		t.Fatal("No backup files were created")
	}
	
	// Verify backup file has expected naming pattern
	foundBackup := false
	for _, file := range backupFiles {
		if strings.HasPrefix(file.Name(), "health_") && strings.HasSuffix(file.Name(), ".db") {
			foundBackup = true
			break
		}
	}
	
	if !foundBackup {
		t.Error("Backup file doesn't match expected naming pattern health_YYYYMMDD.db")
	}
}

// TestErrorConditions tests how the package handles error conditions
// Junior developers need to understand what happens when things go wrong
func TestErrorConditions(t *testing.T) {
	// Test 1: Invalid database path (permission denied)
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", "/root/invalid/path/health.db") // Should fail on most systems
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
	}()
	
	// Package should handle this gracefully - fall back to memory backend
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-error-conditions")
	
	// Should still work despite persistence failure
	state.IncrMetric("error_test")
	jsonOutput := state.Dump()  
	
	if jsonOutput == "" {
		t.Fatal("Package should continue working even with persistence errors")
	}
	
	// Test 2: Invalid metric names (should be ignored)
	state.IncrMetric("")           // Empty name - should be ignored
	state.IncrMetric("   ")        // Whitespace - should be ignored
	state.AddMetric("", 123.45)    // Empty name - should be ignored
	
	// These invalid operations should not cause crashes or corrupt data
	jsonOutput = state.Dump()
	if jsonOutput == "" {
		t.Fatal("Package should handle invalid inputs gracefully")
	}
	
	// Test 3: Extreme values
	state.AddMetric("extreme_value", 1e20)  // Very large number
	state.AddMetric("small_value", 1e-20)   // Very small number
	state.AddMetric("negative", -1000.0)    // Negative number
	
	// Should not cause issues
	jsonOutput = state.Dump()
	if jsonOutput == "" {
		t.Fatal("Package should handle extreme values")
	}
	
	// Test 4: Multiple Close() calls (should be safe)
	testState := NewState()
	testState.SetConfig("close-test")
	
	// First close should work
	if err := testState.Close(); err != nil {
		t.Errorf("First Close() failed: %v", err)
	}
	
	// Second close should be safe (idempotent)
	if err := testState.Close(); err != nil {
		t.Errorf("Second Close() should be safe, got error: %v", err)
	}
}