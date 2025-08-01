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

// ReadMetrics retrieves metrics for a component within the time range
func (s *SQLiteBackend) ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error) {
	var query string
	var args []interface{}

	if component == "" {
		// Query all components
		query = `SELECT timestamp, component, name, value, type 
				FROM metrics 
				WHERE timestamp >= ? AND timestamp <= ? 
				ORDER BY timestamp ASC`
		args = []interface{}{start.Unix(), end.Unix()}
	} else {
		// Query specific component
		query = `SELECT timestamp, component, name, value, type 
				FROM metrics 
				WHERE component = ? AND timestamp >= ? AND timestamp <= ? 
				ORDER BY timestamp ASC`
		args = []interface{}{component, start.Unix(), end.Unix()}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []MetricEntry
	for rows.Next() {
		var metric MetricEntry
		var timestamp int64
		var value float64

		err := rows.Scan(&timestamp, &metric.Component, &metric.Name, &value, &metric.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		metric.Timestamp = time.Unix(timestamp, 0)

		// Convert value based on type
		if metric.Type == "counter" {
			metric.Value = int(value)
		} else {
			metric.Value = value
		}

		metrics = append(metrics, metric)
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
