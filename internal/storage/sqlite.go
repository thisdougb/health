package storage

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteBackend implements Backend interface using SQLite database
type SQLiteBackend struct {
	db    *sql.DB
	queue *SQLiteWriteQueue
}

// SQLiteConfig holds configuration for SQLite backend
type SQLiteConfig struct {
	DBPath        string
	FlushInterval time.Duration
	BatchSize     int
}

// NewSQLiteBackend creates a new SQLite storage backend
func NewSQLiteBackend(config SQLiteConfig) (*SQLiteBackend, error) {
	// Open database connection
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runSQLiteMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create write queue
	queue := NewSQLiteWriteQueue(db, config.FlushInterval, config.BatchSize)

	backend := &SQLiteBackend{
		db:    db,
		queue: queue,
	}

	// Start background queue processing
	queue.Start()

	return backend, nil
}

// NewSQLiteBackendFromEnv creates SQLite backend using environment variables
func NewSQLiteBackendFromEnv() (*SQLiteBackend, error) {
	config := SQLiteConfig{
		DBPath:        getEnv("HEALTH_DB_PATH", "./health.db"),
		FlushInterval: parseDuration(getEnv("HEALTH_FLUSH_INTERVAL", "60s")),
		BatchSize:     parseInt(getEnv("HEALTH_BATCH_SIZE", "100")),
	}

	return NewSQLiteBackend(config)
}

// WriteMetrics queues metrics for async writing to SQLite
func (s *SQLiteBackend) WriteMetrics(metrics []MetricEntry) error {
	if len(metrics) == 0 {
		return nil
	}

	return s.queue.Enqueue(metrics)
}

// WriteTimeSeriesMetrics writes aggregated time series metrics directly to SQLite
// This bypasses the queue since these are already aggregated and ready for storage
func (s *SQLiteBackend) WriteTimeSeriesMetrics(metrics []TimeSeriesEntry) error {
	if len(metrics) == 0 {
		return nil
	}

	// Prepare the insert statement
	query := `INSERT OR REPLACE INTO time_series_metrics 
		(time_window_key, component, metric, min_value, max_value, avg_value, count) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	// Begin transaction for batch insert
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert all metrics in the batch
	for _, metric := range metrics {
		_, err := stmt.Exec(
			metric.TimeWindowKey,
			metric.Component,
			metric.Metric,
			metric.MinValue,
			metric.MaxValue,
			metric.AvgValue,
			metric.Count,
		)
		if err != nil {
			return fmt.Errorf("failed to insert time series metric: %w", err)
		}
	}

	return tx.Commit()
}

// ReadMetrics retrieves aggregated time series metrics for a component within the time range
func (s *SQLiteBackend) ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error) {
	// Convert time range to time window keys for efficient querying
	startKey := timeToWindowKey(start)
	endKey := timeToWindowKey(end)
	
	var query string
	var args []interface{}

	if component == "" {
		// Query all components from time_series_metrics
		query = `SELECT time_window_key, component, metric, min_value, max_value, avg_value, count
				FROM time_series_metrics 
				WHERE time_window_key >= ? AND time_window_key <= ? 
				ORDER BY time_window_key ASC, component ASC, metric ASC`
		args = []interface{}{startKey, endKey}
	} else {
		// Query specific component from time_series_metrics  
		query = `SELECT time_window_key, component, metric, min_value, max_value, avg_value, count
				FROM time_series_metrics 
				WHERE component = ? AND time_window_key >= ? AND time_window_key <= ? 
				ORDER BY time_window_key ASC, metric ASC`
		args = []interface{}{component, startKey, endKey}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series metrics: %w", err)
	}
	defer rows.Close()

	var metrics []MetricEntry
	for rows.Next() {
		var timeWindowKey, component, metric string
		var minValue, maxValue, avgValue float64
		var count int

		err := rows.Scan(&timeWindowKey, &component, &metric, &minValue, &maxValue, &avgValue, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time series metric: %w", err)
		}

		// Convert time window key back to timestamp
		timestamp, err := windowKeyToTime(timeWindowKey)
		if err != nil {
			continue // Skip invalid time keys
		}

		// Create MetricEntry with aggregated data - use avg as the primary value
		// For counters (where min=max=avg=1.0), use count as value
		var value interface{}
		var metricType string
		if minValue == 1.0 && maxValue == 1.0 && avgValue == 1.0 {
			// Counter metric - use count
			value = count
			metricType = "counter"
		} else {
			// Value metric - use average, but include stats in metadata
			value = map[string]interface{}{
				"avg":   avgValue,
				"min":   minValue, 
				"max":   maxValue,
				"count": count,
			}
			metricType = "value"
		}

		entry := MetricEntry{
			Timestamp: timestamp,
			Component: component,
			Name:      metric,
			Value:     value,
			Type:      metricType,
		}
		metrics = append(metrics, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return metrics, nil
}

// ListComponents returns all unique component names
func (s *SQLiteBackend) ListComponents() ([]string, error) {
	query := `SELECT DISTINCT component FROM metrics ORDER BY component`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query components: %w", err)
	}
	defer rows.Close()

	var components []string
	for rows.Next() {
		var component string
		if err := rows.Scan(&component); err != nil {
			return nil, fmt.Errorf("failed to scan component: %w", err)
		}
		components = append(components, component)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return components, nil
}

// Close gracefully shuts down the SQLite backend
func (s *SQLiteBackend) Close() error {
	// Stop queue processing and flush remaining items
	if s.queue != nil {
		s.queue.Stop()
	}

	// Close database connection
	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

// CreateBackup creates a backup of the SQLite database using the existing connection
// This avoids file locking issues by using the same database connection
func (s *SQLiteBackend) CreateBackup(config *BackupConfig) error {
	if s.db == nil {
		return fmt.Errorf("no database connection available")
	}

	return BackupHealthDatabase(s.db, config)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return 60 * time.Second // Default to 60 seconds
	}
	return duration
}

func parseInt(s string) int {
	value, err := strconv.Atoi(s)
	if err != nil {
		return 100 // Default batch size
	}
	return value
}

