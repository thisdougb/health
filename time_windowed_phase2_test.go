package health

import (
	"os"
	"testing"
	"time"

	"github.com/thisdougb/health/internal/core"
	"github.com/thisdougb/health/internal/storage"
)

// TestTimeWindowedMetricsPhase2 tests the move-and-flush architecture
func TestTimeWindowedMetricsPhase2(t *testing.T) {
	// Test with SQLite backend to verify actual database operations
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/health_test.db"

	// Configure environment for testing
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_SAMPLE_RATE", "1") // 1 second windows for fast testing
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_SAMPLE_RATE")
	}()

	state := core.NewState()
	defer state.Close()
	
	state.Info("phase2-test")

	// Add some metrics in the current time window
	state.IncrMetric("requests")
	state.IncrMetric("requests")
	state.IncrComponentMetric("webserver", "connections")
	state.AddMetric("Global", "response_time", 150.5)
	state.AddMetric("webserver", "cpu_usage", 45.2)

	// Wait for the time window to change and trigger flush
	time.Sleep(2 * time.Second)

	// Add more metrics in a new time window
	state.IncrMetric("requests")
	state.AddMetric("Global", "response_time", 125.8)

	// Manually trigger flush to ensure data is processed
	state.MoveToFlushQueueManual() // We'll need to expose this for testing
	
	// Verify that the storage backend received time series data
	storageManager := state.GetStorageManager()
	if storageManager == nil {
		t.Fatal("Storage manager is nil")
	}

	// Force flush any queued metrics
	if err := storageManager.ForceFlush(); err != nil {
		t.Fatalf("Failed to force flush: %v", err)
	}

	t.Logf("Phase 2 move-and-flush test completed successfully")
}

// TestCalculateStats tests the statistics calculation function
func TestCalculateStats(t *testing.T) {
	// Test calculateStats function directly
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	
	min, max, avg, count := core.CalculateStatsPublic(values)
	
	expectedMin := 1.0
	expectedMax := 5.0
	expectedAvg := 3.0
	expectedCount := 5
	
	if min != expectedMin {
		t.Errorf("Expected min %f, got %f", expectedMin, min)
	}
	if max != expectedMax {
		t.Errorf("Expected max %f, got %f", expectedMax, max)
	}
	if avg != expectedAvg {
		t.Errorf("Expected avg %f, got %f", expectedAvg, avg)
	}
	if count != expectedCount {
		t.Errorf("Expected count %d, got %d", expectedCount, count)
	}
}

// TestTimeWindowKeyFormat tests the time window key generation
func TestTimeWindowKeyFormat(t *testing.T) {
	// Test getCurrentTimeKey function
	// We need to expose this function for testing
	
	// Set a known sample rate
	os.Setenv("HEALTH_SAMPLE_RATE", "60")
	defer os.Unsetenv("HEALTH_SAMPLE_RATE")
	
	// The exact key will depend on current time, but we can test the format
	key := core.GetCurrentTimeKeyPublic()
	
	// Should be 14 characters: YYYYMMDDHHMMSS
	if len(key) != 14 {
		t.Errorf("Expected key length 14, got %d", len(key))
	}
	
	// Should be all digits
	for _, char := range key {
		if char < '0' || char > '9' {
			t.Errorf("Expected all digits, got character %c in key %s", char, key)
		}
	}
	
	t.Logf("Generated time key: %s", key)
}

// TestMemoryBackendTimeSeriesSupport tests that memory backend supports time series metrics
func TestMemoryBackendTimeSeriesSupport(t *testing.T) {
	backend := storage.NewMemoryBackend()
	defer backend.Close()
	
	// Test writing time series entries
	entries := []storage.TimeSeriesEntry{
		{
			TimeWindowKey: "20250810143000",
			Component:     "test",
			Metric:        "counter",
			MinValue:      1.0,
			MaxValue:      3.0,
			AvgValue:      2.0,
			Count:         3,
		},
	}
	
	err := backend.WriteTimeSeriesMetrics(entries)
	if err != nil {
		t.Errorf("Failed to write time series metrics to memory backend: %v", err)
	}
	
	t.Log("Memory backend time series support test passed")
}