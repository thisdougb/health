package storage

import (
	"database/sql"
	"fmt"
)

// SQLiteMigration represents a database schema migration for SQLite
type SQLiteMigration struct {
	Version int
	Up      string
	Down    string // Optional rollback SQL
}

// sqliteMigrations contains all SQLite database migrations in chronological order
var sqliteMigrations = []SQLiteMigration{
	{
		Version: 1,
		Up: `CREATE TABLE metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			component TEXT NOT NULL,
			name TEXT NOT NULL,
			value REAL NOT NULL,
			type TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now'))
		);

		CREATE INDEX idx_metrics_time_component ON metrics(timestamp, component);
		CREATE INDEX idx_metrics_component_name ON metrics(component, name);`,
		Down: `DROP TABLE IF EXISTS metrics;`,
	},
	{
		Version: 2,
		Up: `CREATE TABLE time_series_metrics (
			time_window_key TEXT NOT NULL,
			component TEXT NOT NULL,
			metric TEXT NOT NULL,
			min_value REAL NOT NULL,
			max_value REAL NOT NULL,
			avg_value REAL NOT NULL,
			count INTEGER NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			PRIMARY KEY (time_window_key, component, metric)
		);

		CREATE INDEX idx_time_series_component ON time_series_metrics(component);
		CREATE INDEX idx_time_series_window ON time_series_metrics(time_window_key);`,
		Down: `DROP TABLE IF EXISTS time_series_metrics;`,
	},
}

// runSQLiteMigrations applies all pending SQLite migrations to the database
func runSQLiteMigrations(db *sql.DB) error {
	// Create schema_migrations table if it doesn't exist
	if err := createSQLiteMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current schema version
	currentVersion, err := getCurrentSQLiteVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Apply pending migrations
	for _, migration := range sqliteMigrations {
		if migration.Version <= currentVersion {
			continue // Migration already applied
		}

		if err := applySQLiteMigration(db, migration); err != nil {
			return fmt.Errorf("failed to apply migration version %d: %w", migration.Version, err)
		}
	}

	return nil
}

// createSQLiteMigrationsTable creates the schema_migrations table for tracking applied migrations
func createSQLiteMigrationsTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
	)`

	_, err := db.Exec(query)
	return err
}

// getCurrentSQLiteVersion returns the highest applied migration version
func getCurrentSQLiteVersion(db *sql.DB) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`

	var version int
	err := db.QueryRow(query).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

// applySQLiteMigration applies a single migration within a transaction
func applySQLiteMigration(db *sql.DB, migration SQLiteMigration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute the migration SQL
	if _, err := tx.Exec(migration.Up); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record the migration as applied
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// GetSQLiteSchemaVersion returns the current schema version (for testing/debugging)
func GetSQLiteSchemaVersion(db *sql.DB) (int, error) {
	return getCurrentSQLiteVersion(db)
}
