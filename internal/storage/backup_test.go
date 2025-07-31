package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestBackupHealthDatabase(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create test table and data
	_, err = db.Exec(`
		CREATE TABLE metrics (
			id INTEGER PRIMARY KEY,
			timestamp INTEGER,
			component TEXT,
			name TEXT,
			value REAL,
			type TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO metrics (timestamp, component, name, value, type) 
		VALUES (?, ?, ?, ?, ?)
	`, time.Now().Unix(), "test", "counter", 42.0, "counter")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test backup with enabled config
	backupDir := filepath.Join(tempDir, "backups")
	config := &BackupConfig{
		Enabled:        true,
		BackupDir:      backupDir,
		RetentionDays:  30,
		BackupInterval: 24 * time.Hour,
	}

	err = BackupHealthDatabase(db, config)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Verify backup file exists
	date := time.Now().Format("20060102")
	expectedBackup := filepath.Join(backupDir, "health_"+date+".db")
	
	if _, err := os.Stat(expectedBackup); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created: %s", expectedBackup)
	}

	// Verify backup contains data
	backupDB, err := sql.Open("sqlite3", expectedBackup)
	if err != nil {
		t.Fatalf("Failed to open backup database: %v", err)
	}
	defer backupDB.Close()

	var count int
	err = backupDB.QueryRow("SELECT COUNT(*) FROM metrics").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query backup database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row in backup, got %d", count)
	}
}

func TestBackupHealthDatabase_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Test backup with disabled config
	config := &BackupConfig{
		Enabled: false,
	}

	err = BackupHealthDatabase(db, config)
	if err != nil {
		t.Fatalf("Backup should succeed when disabled: %v", err)
	}

	// Verify no backup directory was created
	if _, err := os.Stat(config.BackupDir); !os.IsNotExist(err) {
		t.Error("Backup directory should not exist when backup is disabled")
	}
}

func TestCleanupHealthBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	// Create backup directory
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test backup files with different dates
	now := time.Now()
	testFiles := []struct {
		date   time.Time
		name   string
		expect bool
	}{
		{now, "health_" + now.Format("20060102") + ".db", true},                                    // Today - keep
		{now.AddDate(0, 0, -5), "health_" + now.AddDate(0, 0, -5).Format("20060102") + ".db", true}, // 5 days ago - keep
		{now.AddDate(0, 0, -35), "health_" + now.AddDate(0, 0, -35).Format("20060102") + ".db", false}, // 35 days ago - remove
		{now.AddDate(0, 0, -50), "health_" + now.AddDate(0, 0, -50).Format("20060102") + ".db", false}, // 50 days ago - remove
	}

	// Create test files
	for _, tf := range testFiles {
		filePath := filepath.Join(backupDir, tf.name)
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.name, err)
		}
		file.Close()
	}

	// Create non-backup file that should be ignored
	nonBackupFile := filepath.Join(backupDir, "other_file.txt")
	file, err := os.Create(nonBackupFile)
	if err != nil {
		t.Fatalf("Failed to create non-backup file: %v", err)
	}
	file.Close()

	// Test cleanup with 30-day retention
	config := &BackupConfig{
		BackupDir:     backupDir,
		RetentionDays: 30,
	}

	err = CleanupHealthBackups(config)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify expected files remain/removed
	for _, tf := range testFiles {
		filePath := filepath.Join(backupDir, tf.name)
		_, err := os.Stat(filePath)
		
		if tf.expect && os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist but it was removed", tf.name)
		} else if !tf.expect && err == nil {
			t.Errorf("Expected file %s to be removed but it still exists", tf.name)
		}
	}

	// Verify non-backup file is untouched
	if _, err := os.Stat(nonBackupFile); os.IsNotExist(err) {
		t.Error("Non-backup file should not be removed")
	}
}

func TestListHealthBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	config := &BackupConfig{
		BackupDir: backupDir,
	}

	// Test empty directory
	backups, err := ListHealthBackups(config)
	if err != nil {
		t.Fatalf("Failed to list backups from empty directory: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("Expected 0 backups, got %d", len(backups))
	}

	// Create backup directory and files
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	expectedBackups := []string{
		"health_20240101.db",
		"health_20240102.db", 
		"health_20240103.db",
	}

	for _, backup := range expectedBackups {
		filePath := filepath.Join(backupDir, backup)
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create test backup %s: %v", backup, err)
		}
		file.Close()
	}

	// Create non-backup file that should be ignored
	nonBackupFile := filepath.Join(backupDir, "other_file.txt")
	file, err := os.Create(nonBackupFile)
	if err != nil {
		t.Fatalf("Failed to create non-backup file: %v", err)
	}
	file.Close()

	// Test listing backups
	backups, err = ListHealthBackups(config)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != len(expectedBackups) {
		t.Errorf("Expected %d backups, got %d", len(expectedBackups), len(backups))
	}

	// Verify backups are sorted
	for i, backup := range backups {
		if backup != expectedBackups[i] {
			t.Errorf("Expected backup %s at index %d, got %s", expectedBackups[i], i, backup)
		}
	}
}

func TestRestoreHealthDatabase(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	// Create backup directory
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create a test backup file with database content
	backupFile := "health_20240101.db"
	backupPath := filepath.Join(backupDir, backupFile)
	
	// Create backup database with test data
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		t.Fatalf("Failed to create backup database: %v", err)
	}

	_, err = backupDB.Exec(`
		CREATE TABLE metrics (
			id INTEGER PRIMARY KEY,
			timestamp INTEGER,
			component TEXT,
			name TEXT,
			value REAL,
			type TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create backup table: %v", err)
	}

	_, err = backupDB.Exec(`
		INSERT INTO metrics (timestamp, component, name, value, type) 
		VALUES (?, ?, ?, ?, ?)
	`, time.Now().Unix(), "restored", "test_metric", 123.0, "counter")
	if err != nil {
		t.Fatalf("Failed to insert backup data: %v", err)
	}
	backupDB.Close()

	// Test restore
	config := &BackupConfig{
		BackupDir: backupDir,
	}

	targetDBPath := filepath.Join(tempDir, "restored.db")
	err = RestoreHealthDatabase(backupFile, targetDBPath, config)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify restored database
	restoredDB, err := sql.Open("sqlite3", targetDBPath)
	if err != nil {
		t.Fatalf("Failed to open restored database: %v", err)
	}
	defer restoredDB.Close()

	var component, name string
	var value float64
	err = restoredDB.QueryRow("SELECT component, name, value FROM metrics").Scan(&component, &name, &value)
	if err != nil {
		t.Fatalf("Failed to query restored database: %v", err)
	}

	if component != "restored" || name != "test_metric" || value != 123.0 {
		t.Errorf("Restored data mismatch: got %s/%s/%f, expected restored/test_metric/123.0", 
			component, name, value)
	}
}

func TestRestoreHealthDatabase_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	config := &BackupConfig{
		BackupDir: filepath.Join(tempDir, "backups"),
	}

	targetDBPath := filepath.Join(tempDir, "restored.db")
	err := RestoreHealthDatabase("nonexistent.db", targetDBPath, config)
	
	if err == nil {
		t.Fatal("Expected error for non-existent backup file")
	}

	// Check that the error message indicates file not found
	expectedMsg := "backup file not found"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedMsg, err)
	}
}

func TestFindBackupForDate(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	// Create backup directory and file
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	testDate := "20240115"
	backupFile := "health_" + testDate + ".db"
	backupPath := filepath.Join(backupDir, backupFile)
	
	file, err := os.Create(backupPath)
	if err != nil {
		t.Fatalf("Failed to create test backup: %v", err)
	}
	file.Close()

	config := &BackupConfig{
		BackupDir: backupDir,
	}

	// Test finding existing backup
	foundFile, err := FindBackupForDate(testDate, config)
	if err != nil {
		t.Fatalf("Failed to find backup: %v", err)
	}

	if foundFile != backupFile {
		t.Errorf("Expected %s, got %s", backupFile, foundFile)
	}

	// Test finding non-existent backup
	_, err = FindBackupForDate("20240201", config)
	if err == nil {
		t.Fatal("Expected error for non-existent backup")
	}
}

func TestGetBackupRetentionInfo(t *testing.T) {
	config := &BackupConfig{
		Enabled:        true,
		BackupDir:      "/test/backups",
		RetentionDays:  45,
		BackupInterval: 12 * time.Hour,
	}

	info := GetBackupRetentionInfo(config)

	expectedKeys := []string{"enabled", "backup_dir", "retention_days", "backup_interval"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Expected key %s in retention info", key)
		}
	}

	if info["enabled"] != true {
		t.Errorf("Expected enabled=true, got %v", info["enabled"])
	}

	if info["backup_dir"] != "/test/backups" {
		t.Errorf("Expected backup_dir=/test/backups, got %v", info["backup_dir"])
	}

	if info["retention_days"] != 45 {
		t.Errorf("Expected retention_days=45, got %v", info["retention_days"])
	}
}

func TestLoadBackupConfig(t *testing.T) {
	// Test defaults
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if config.Backup.Enabled != false {
		t.Errorf("Expected default enabled=false, got %t", config.Backup.Enabled)
	}

	if config.Backup.BackupDir != "/data/backups/health" {
		t.Errorf("Expected default backup dir, got %s", config.Backup.BackupDir)
	}

	if config.Backup.RetentionDays != 30 {
		t.Errorf("Expected default retention=30, got %d", config.Backup.RetentionDays)
	}

	if config.Backup.BackupInterval != 24*time.Hour {
		t.Errorf("Expected default interval=24h, got %v", config.Backup.BackupInterval)
	}

	// Test environment variable override
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")
	os.Setenv("HEALTH_BACKUP_DIR", "/custom/backup/path")
	os.Setenv("HEALTH_BACKUP_RETENTION_DAYS", "720h") // 30 days
	os.Setenv("HEALTH_BACKUP_INTERVAL", "6h")
	
	defer func() {
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
		os.Unsetenv("HEALTH_BACKUP_RETENTION_DAYS") 
		os.Unsetenv("HEALTH_BACKUP_INTERVAL")
	}()

	config, err = LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config with env vars: %v", err)
	}

	if config.Backup.Enabled != true {
		t.Errorf("Expected enabled=true from env, got %t", config.Backup.Enabled)
	}

	if config.Backup.BackupDir != "/custom/backup/path" {
		t.Errorf("Expected custom backup dir from env, got %s", config.Backup.BackupDir)
	}

	if config.Backup.RetentionDays != 30 {
		t.Errorf("Expected retention=30 from env, got %d", config.Backup.RetentionDays)
	}

	if config.Backup.BackupInterval != 6*time.Hour {
		t.Errorf("Expected interval=6h from env, got %v", config.Backup.BackupInterval)
	}
}