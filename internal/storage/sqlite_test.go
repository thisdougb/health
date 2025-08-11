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
		DBPath: dbPath,
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

	// All metrics now return statistical aggregation (no more counter/value distinction)
	// Check first metric - should return aggregated data
	if readMetrics[0].Component != "test" ||
		readMetrics[0].Name != "counter1" ||
		readMetrics[0].Type != "value" {
		t.Fatalf("First metric doesn't match expected component/name/type: %+v", readMetrics[0])
	}
	
	// Verify first metric has aggregated data
	firstStats, ok := readMetrics[0].Value.(map[string]interface{})
	if !ok {
		t.Fatalf("First metric should have aggregated data map, got %T", readMetrics[0].Value)
	}
	if firstStats["count"] != 42 || firstStats["avg"] != 1.0 {
		t.Fatalf("First metric stats don't match expected values: %+v", firstStats)
	}

	// Check second metric - should also return aggregated data
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

	// Verify both components are returned with aggregated data
	foundComp1 := false
	foundComp2 := false
	for _, metric := range readMetrics {
		if metric.Component == "comp1" && metric.Name == "metric1" {
			// Check that metric has aggregated data
			if stats, ok := metric.Value.(map[string]interface{}); ok {
				if stats["count"] == 5 && stats["avg"] == 1.0 {
					foundComp1 = true
				}
			}
		}
		if metric.Component == "comp2" && metric.Name == "metric2" {
			// Check that metric has aggregated data
			if stats, ok := metric.Value.(map[string]interface{}); ok {
				if stats["count"] == 10 && stats["avg"] == 1.0 {
					foundComp2 = true
				}
			}
		}
	}
	
	if !foundComp1 || !foundComp2 {
		t.Fatalf("Missing expected components in results: %+v", readMetrics)
	}
}

func TestSQLiteBackend_ListComponents(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Create manager for proper queue processing
	manager := NewManager(backend, true)
	defer manager.Close()

	// Test metrics from different components
	now := time.Now().Truncate(time.Second)
	metrics := []MetricEntry{
		{Timestamp: now, Component: "alpha", Name: "metric1", Value: 1, Type: "value"},
		{Timestamp: now, Component: "beta", Name: "metric2", Value: 2, Type: "value"},
		{Timestamp: now, Component: "alpha", Name: "metric3", Value: 3, Type: "value"},
	}

	err := manager.PersistMetrics(metrics)
	if err != nil {
		t.Fatalf("Failed to persist metrics: %v", err)
	}

	err = manager.ForceFlush()
	if err != nil {
		t.Fatalf("Failed to flush queue: %v", err)
	}

	// List components through manager
	components, err := manager.ListComponents()
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

func TestSQLiteUniversalQueue_BatchProcessing(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Create manager with small batch size for testing
	manager := NewManager(backend, true)
	defer manager.Close()

	// Test that manager properly processes metrics through universal queue
	now := time.Now().Truncate(time.Second)
	testMetrics := []MetricEntry{
		{Timestamp: now, Component: "test", Name: "metric1", Value: 1, Type: "value"},
		{Timestamp: now.Add(time.Second), Component: "test", Name: "metric2", Value: 2, Type: "value"},
	}

	// Queue metrics through manager (uses universal queue)
	err := manager.PersistMetrics(testMetrics)
	if err != nil {
		t.Fatalf("Failed to persist metrics: %v", err)
	}

	// Force flush to ensure processing
	err = manager.ForceFlush()
	if err != nil {
		t.Fatalf("Failed to force flush: %v", err)
	}

	// Verify metrics were written to database
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	readMetrics, err := manager.ReadMetrics("test", start, end)
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	if len(readMetrics) != 2 {
		t.Fatalf("Expected 2 metrics in database, got %d", len(readMetrics))
	}

	// Verify all metrics are now value metrics with statistical aggregation
	for _, metric := range readMetrics {
		if metric.Type != "value" {
			t.Errorf("Expected metric %s to be type 'value', got '%s'", metric.Name, metric.Type)
		}
		if _, isMap := metric.Value.(map[string]interface{}); !isMap {
			t.Errorf("Expected metric %s to have aggregated value map, got %T", metric.Name, metric.Value)
		}
	}
}

func TestSQLiteBackend_WriteMetrics_ShouldFail(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// SQLite backend should reject raw metrics and require processed data
	now := time.Now()
	metrics := []MetricEntry{
		{Timestamp: now, Component: "test", Name: "metric", Value: 42, Type: "value"},
	}

	err := backend.WriteMetrics(metrics)
	if err == nil {
		t.Error("Expected WriteMetrics to fail for SQLite backend, but it succeeded")
	}
}

func TestSQLiteBackend_EmptyMetrics(t *testing.T) {
	backend := setupTestSQLiteBackend(t)
	defer backend.Close()

	// Write empty metrics data slice (should succeed)
	err := backend.WriteMetricsData([]MetricsDataEntry{})
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
		DBPath: dbPath,
	}

	backend, err := NewSQLiteBackend(config)
	if err != nil {
		t.Fatalf("Failed to create test SQLite backend: %v", err)
	}

	return backend
}
