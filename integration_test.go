//go:build dev

package health

import (
	"os"
	"testing"

	"github.com/thisdougb/health/internal/storage"
)

func TestIntegrationMemoryPersistence(t *testing.T) {
	// Create state with memory backend
	manager := storage.NewManager(storage.NewMemoryBackend(), true)
	state := NewStateWithPersistence(manager)
	defer state.Close()

	// Set config
	state.SetConfig("test-instance")

	// Test counter metrics
	state.IncrMetric("requests")
	state.IncrComponentMetric("webserver", "requests")

	// Test value metrics
	state.AddMetric("response_time", 100.5)
	state.AddComponentMetric("database", "query_time", 25.3)

	// Force flush any pending persistence operations
	if sm := state.GetStorageManager(); sm != nil {
		sm.ForceFlush()
	}

	// Verify metrics are in memory
	json := state.Dump()
	if json == "" {
		t.Fatal("Dump returned empty string")
	}

	// For memory backend, we can't easily verify persistence without accessing internals
	// but we can verify no errors occurred during the operations
}

func TestIntegrationSQLitePersistence(t *testing.T) {
	// Create temporary SQLite database
	tmpFile := "/tmp/health_test.db"
	defer os.Remove(tmpFile)

	// Set environment variables for SQLite backend
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_FLUSH_INTERVAL", "1s")
	os.Setenv("HEALTH_BATCH_SIZE", "10")
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_FLUSH_INTERVAL")
		os.Unsetenv("HEALTH_BATCH_SIZE")
	}()

	// Create state - should automatically use SQLite backend from env vars
	state := NewState()
	defer state.Close()

	// Set config
	state.SetConfig("test-sqlite-instance")

	// Test counter metrics
	state.IncrMetric("requests")
	state.IncrComponentMetric("webserver", "requests")
	state.IncrComponentMetric("database", "queries")

	// Test value metrics
	state.AddMetric("response_time", 100.5)
	state.AddComponentMetric("database", "query_time", 25.3)

	// Force flush persistence and batching
	if sm := state.GetStorageManager(); sm != nil {
		sm.ForceFlush()
	}

	// Verify the database file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("SQLite database file was not created")
	}

	// Verify metrics are still available in memory
	json := state.Dump()
	if json == "" {
		t.Fatal("Dump returned empty string")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that existing API still works without any persistence configuration
	state := NewState()
	defer state.Close()

	// All existing methods should work exactly as before
	state.SetConfig("test-compat")
	
	state.IncrMetric("requests")
	state.AddMetric("response_time", 123.45)
	
	state.IncrComponentMetric("webserver", "requests")
	state.AddComponentMetric("cache", "hit_rate", 0.85)

	json := state.Dump()
	if json == "" {
		t.Fatal("Dump returned empty string")
	}

	// Verify handler methods still work
	if state.HealthHandler() == nil {
		t.Fatal("HealthHandler returned nil")
	}
	
	if state.StatusHandler() == nil {
		t.Fatal("StatusHandler returned nil")
	}
}

func TestGracefulShutdown(t *testing.T) {
	// Test that Close() method works properly
	state := NewState()
	state.SetConfig("test-shutdown")
	
	// Add some metrics
	state.IncrMetric("test")
	state.AddMetric("test_value", 42.0)
	
	// Should close without error
	if err := state.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
	
	// Should be safe to call Close() multiple times
	if err := state.Close(); err != nil {
		t.Fatalf("Second Close() returned error: %v", err)
	}
}