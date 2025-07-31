package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_CreateBackup_EventDriven(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	backupDir := filepath.Join(tempDir, "backups")

	// Set up environment for SQLite with backup enabled
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", dbPath)
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")
	os.Setenv("HEALTH_BACKUP_DIR", backupDir)

	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
	}()

	// Create manager from config
	manager, err := NewManagerFromConfig()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Add some test metrics to have data to backup
	err = manager.PersistMetric("test", "counter", 42, "counter")
	if err != nil {
		t.Fatalf("Failed to persist metric: %v", err)
	}

	err = manager.PersistMetric("test", "value", 123.45, "value")
	if err != nil {
		t.Fatalf("Failed to persist metric: %v", err)
	}

	// Wait a moment for async writes to complete
	time.Sleep(100 * time.Millisecond)

	// Test event-driven backup creation
	err = manager.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup file was created
	date := time.Now().Format("20060102")
	expectedBackup := filepath.Join(backupDir, "health_"+date+".db")

	if _, err := os.Stat(expectedBackup); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created: %s", expectedBackup)
	}

	// Test listing backups
	backups, err := manager.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}

	expectedFileName := "health_" + date + ".db"
	if backups[0] != expectedFileName {
		t.Errorf("Expected backup file %s, got %s", expectedFileName, backups[0])
	}
}

func TestManager_CreateBackup_MemoryBackend(t *testing.T) {
	// Create manager with memory backend (no persistence)
	manager := NewManager(NewMemoryBackend(), false)
	defer manager.Close()

	// Backup should be no-op for memory backend
	err := manager.CreateBackup()
	if err != nil {
		t.Errorf("Backup should be no-op for memory backend, got error: %v", err)
	}
}

func TestManager_CreateBackup_BackupDisabled(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Set up environment for SQLite with backup disabled
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", dbPath)
	os.Setenv("HEALTH_BACKUP_ENABLED", "false")

	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
	}()

	// Create manager from config
	manager, err := NewManagerFromConfig()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Backup should be no-op when disabled
	err = manager.CreateBackup()
	if err != nil {
		t.Errorf("Backup should be no-op when disabled, got error: %v", err)
	}
}

func TestManager_BackupOnClose_EventDriven(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	backupDir := filepath.Join(tempDir, "backups")

	// Set up environment for SQLite with backup enabled
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", dbPath)
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")
	os.Setenv("HEALTH_BACKUP_DIR", backupDir)

	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
	}()

	// Create manager from config
	manager, err := NewManagerFromConfig()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add some test metrics
	err = manager.PersistMetric("close_test", "counter", 100, "counter")
	if err != nil {
		t.Fatalf("Failed to persist metric: %v", err)
	}

	// Wait for async write
	time.Sleep(100 * time.Millisecond)

	// Close manager - this should trigger event-driven backup
	err = manager.Close()
	if err != nil {
		t.Fatalf("Failed to close manager: %v", err)
	}

	// Verify backup was created on close
	date := time.Now().Format("20060102")
	expectedBackup := filepath.Join(backupDir, "health_"+date+".db")

	if _, err := os.Stat(expectedBackup); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created on close: %s", expectedBackup)
	}
}

func TestManager_GetBackupInfo(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	backupDir := filepath.Join(tempDir, "backups")

	// Set up environment with persistence AND backup configuration
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", dbPath)
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")
	os.Setenv("HEALTH_BACKUP_DIR", backupDir)
	os.Setenv("HEALTH_BACKUP_RETENTION_DAYS", "720h") // 30 days
	os.Setenv("HEALTH_BACKUP_INTERVAL", "12h")

	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
		os.Unsetenv("HEALTH_BACKUP_RETENTION_DAYS")
		os.Unsetenv("HEALTH_BACKUP_INTERVAL")
	}()

	// Create manager from config
	manager, err := NewManagerFromConfig()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Test backup info
	info := manager.GetBackupInfo()

	expectedKeys := []string{"enabled", "backup_dir", "retention_days", "backup_interval"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Expected key %s in backup info", key)
		}
	}

	if info["enabled"] != true {
		t.Errorf("Expected backup enabled=true, got %v", info["enabled"])
	}

	if info["backup_dir"] != backupDir {
		t.Errorf("Expected backup_dir=%s, got %v", backupDir, info["backup_dir"])
	}

	if info["retention_days"] != 30 {
		t.Errorf("Expected retention_days=30, got %v", info["retention_days"])
	}
}