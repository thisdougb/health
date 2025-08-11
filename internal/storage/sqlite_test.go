package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteBackend_NewBackend(t *testing.T) {
	// Create temp database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := SQLiteConfig{
		DBPath:        dbPath,
		FlushInterval: time.Second,
		BatchSize:     10,
	}

	backend, err := NewSQLiteBackend(config)
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}
	defer backend.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("Database file was not created")
	}

	// Verify migrations were applied
	version, err := GetSQLiteSchemaVersion(backend.db)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}
	if version == 0 {
		t.Fatalf("Expected schema version > 0, got %d", version)
	}
}

func TestSQLiteBackend_WriteAndReadMetrics(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Test time series metrics (aggregated format)
	now := time.Now().Truncate(time.Minute) // Use minute precision for time windows
	timeKey := now.Format("20060102150400")
	
	tsMetrics := []MetricsDataEntry{
		{
			TimeWindowKey: timeKey,
			Component:     "test",
			Metric:        "counter1",
			MinValue:      1.0,  // Counter: all values are 1.0
			MaxValue:      1.0,
			AvgValue:      1.0,
			Count:         42,   // Counter: count shows total increments
		},
		{
			TimeWindowKey: timeKey,
			Component:     "test", 
			Metric:        "value_metric",
			MinValue:      2.0,   // Value metric: actual min/max/avg
			MaxValue:      5.0,
			AvgValue:      3.14,
			Count:         10,    // Value metric: count shows number of samples
		},
	}

	// Write time series metrics
	err := backend.WriteMetricsData(tsMetrics)
	if err != nil {
		t.Fatalf("Failed to write time series metrics: %v", err)
	}

	// Read metrics (now returns time-series format)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	readMetrics, err := backend.ReadMetrics("test", start, end)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	// Verify metrics
	if len(readMetrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(readMetrics))
	}

	// Check first metric (counter) - should return count as value
	if readMetrics[0].Component != "test" ||
		readMetrics[0].Name != "counter1" ||
		readMetrics[0].Value != 42 ||
		readMetrics[0].Type != "counter" {
		t.Fatalf("First metric doesn't match: %+v", readMetrics[0])
	}

	// Check second metric (value) - should return stats map
	if readMetrics[1].Component != "test" || readMetrics[1].Name != "value_metric" {
		t.Fatalf("Second metric component/name doesn't match: %+v", readMetrics[1])
	}
	
	// Value metrics return a map with statistics
	statsMap, ok := readMetrics[1].Value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected value metric to return stats map, got %T", readMetrics[1].Value)
	}
	
	if statsMap["min"] != 2.0 || statsMap["max"] != 5.0 || 
		 statsMap["avg"] != 3.14 || statsMap["count"] != 10 {
		t.Fatalf("Second metric stats don't match: %+v", statsMap)
	}
}

func TestSQLiteBackend_ReadMetricsAllComponents(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Test time series metrics from different components
	now := time.Now().Truncate(time.Minute)
	timeKey := now.Format("20060102150400")
	
	tsMetrics := []MetricsDataEntry{
		{
			TimeWindowKey: timeKey,
			Component:     "comp1", 
			Metric:        "metric1",
			MinValue:      1.0,
			MaxValue:      1.0,
			AvgValue:      1.0,
			Count:         5,
		},
		{
			TimeWindowKey: timeKey,
			Component:     "comp2",
			Metric:        "metric2", 
			MinValue:      1.0,
			MaxValue:      1.0,
			AvgValue:      1.0,
			Count:         10,
		},
	}

	err := backend.WriteMetricsData(tsMetrics)
	if err != nil {
		t.Fatalf("Failed to write time series metrics: %v", err)
	}

	// Read all components (empty component string)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	readMetrics, err := backend.ReadMetrics("", start, end)
	if err != nil {
		t.Fatalf("Failed to read all metrics: %v", err)
	}

	if len(readMetrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(readMetrics))
	}

	// Verify both components are returned
	foundComp1 := false
	foundComp2 := false
	for _, metric := range readMetrics {
		if metric.Component == "comp1" && metric.Name == "metric1" && metric.Value == 5 {
			foundComp1 = true
		}
		if metric.Component == "comp2" && metric.Name == "metric2" && metric.Value == 10 {
			foundComp2 = true  
		}
	}
	
	if !foundComp1 || !foundComp2 {
		t.Fatalf("Missing expected components in results: %+v", readMetrics)
	}
}

func TestSQLiteBackend_ListComponents(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Test metrics from different components
	now := time.Now().Truncate(time.Second)
	metrics := []MetricEntry{
		{Timestamp: now, Component: "alpha", Name: "metric1", Value: 1, Type: "counter"},
		{Timestamp: now, Component: "beta", Name: "metric2", Value: 2, Type: "counter"},
		{Timestamp: now, Component: "alpha", Name: "metric3", Value: 3, Type: "counter"},
	}

	err := backend.WriteMetrics(metrics)
	if err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	err = backend.queue.ForceFlush()
	if err != nil {
		t.Fatalf("Failed to flush queue: %v", err)
	}

	// List components
	components, err := backend.ListComponents()
	if err != nil {
		t.Fatalf("Failed to list components: %v", err)
	}

	expectedComponents := []string{"alpha", "beta"}
	if len(components) != len(expectedComponents) {
		t.Fatalf("Expected %d components, got %d", len(expectedComponents), len(components))
	}

	// Components should be sorted
	for i, expected := range expectedComponents {
		if components[i] != expected {
			t.Fatalf("Expected component %s at index %d, got %s", expected, i, components[i])
		}
	}
}

func TestSQLiteWriteQueue_BatchProcessing(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Set small batch size for testing
	backend.queue.batchSize = 2

	// Add metrics one by one
	now := time.Now().Truncate(time.Second)
	metrics1 := []MetricEntry{
		{Timestamp: now, Component: "test", Name: "metric1", Value: 1, Type: "counter"},
	}
	metrics2 := []MetricEntry{
		{Timestamp: now, Component: "test", Name: "metric2", Value: 2, Type: "counter"},
	}

	// Queue first metric (should not flush yet)
	err := backend.WriteMetrics(metrics1)
	if err != nil {
		t.Fatalf("Failed to write first metric: %v", err)
	}

	// Check queue size
	if backend.queue.QueueSize() != 1 {
		t.Fatalf("Expected queue size 1, got %d", backend.queue.QueueSize())
	}

	// Queue second metric (should trigger flush)
	err = backend.WriteMetrics(metrics2)
	if err != nil {
		t.Fatalf("Failed to write second metric: %v", err)
	}

	// Queue should be empty after batch flush
	if backend.queue.QueueSize() != 0 {
		t.Fatalf("Expected queue size 0 after batch flush, got %d", backend.queue.QueueSize())
	}

	// Verify metrics were written to database
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	readMetrics, err := backend.ReadMetrics("test", start, end)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	if len(readMetrics) != 2 {
		t.Fatalf("Expected 2 metrics in database, got %d", len(readMetrics))
	}
}

func TestSQLiteBackend_EmptyMetrics(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Write empty metrics slice
	err := backend.WriteMetrics([]MetricEntry{})
	if err != nil {
		t.Fatalf("Failed to write empty metrics: %v", err)
	}

	// Read from empty database
	now := time.Now()
	metrics, err := backend.ReadMetrics("test", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to read from empty database: %v", err)
	}

	if len(metrics) != 0 {
		t.Fatalf("Expected 0 metrics from empty database, got %d", len(metrics))
	}

	// List components from empty database
	components, err := backend.ListComponents()
	if err != nil {
		t.Fatalf("Failed to list components from empty database: %v", err)
	}

	if len(components) != 0 {
		t.Fatalf("Expected 0 components from empty database, got %d", len(components))
	}
}

func TestSQLiteMigrations(t *testing.T) {
	// Create temp database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migrations_test.db")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = runSQLiteMigrations(db)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify schema version
	version, err := GetSQLiteSchemaVersion(db)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	expectedVersion := len(sqliteMigrations)
	if version != expectedVersion {
		t.Fatalf("Expected schema version %d, got %d", expectedVersion, version)
	}

	// Run migrations again (should be idempotent)
	err = runSQLiteMigrations(db)
	if err != nil {
		t.Fatalf("Failed to run migrations second time: %v", err)
	}

	// Version should remain the same
	version2, err := GetSQLiteSchemaVersion(db)
	if err != nil {
		t.Fatalf("Failed to get schema version after second run: %v", err)
	}

	if version2 != expectedVersion {
		t.Fatalf("Schema version changed after second migration run: %d -> %d", version, version2)
	}

	// Verify metrics table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='metrics'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Metrics table was not created: %v", err)
	}
}

// setupTestSQLiteBackend creates a SQLite backend for testing
func setupTestSQLiteBackend(t *testing.T) *SQLiteBackend {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := SQLiteConfig{
		DBPath:        dbPath,
		FlushInterval: time.Hour, // Long interval for testing
		BatchSize:     100,
	}

	backend, err := NewSQLiteBackend(config)
	if err != nil {
		t.Fatalf("Failed to create test SQLite backend: %v", err)
	}

	return backend
}
