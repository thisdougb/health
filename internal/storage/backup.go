package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupHealthDatabase creates a backup of the health database using SQLite VACUUM INTO
// Following tripkist backup patterns with atomic backup creation and automatic cleanup
func BackupHealthDatabase(db *sql.DB, config *BackupConfig) error {
	if !config.Enabled {
		return nil // Backup disabled
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with date format
	date := time.Now().Format("20060102")
	backupPath := filepath.Join(config.BackupDir, fmt.Sprintf("health_%s.db", date))

	// Remove existing backup for today if it exists (atomic replacement)
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to remove existing backup: %w", err)
		}
	}

	// Create atomic backup using SQLite VACUUM INTO
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath)); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Cleanup old backups according to retention policy
	if err := CleanupHealthBackups(config); err != nil {
		return fmt.Errorf("backup succeeded but cleanup failed: %w", err)
	}

	return nil
}

// CleanupHealthBackups removes old backup files according to retention policy
// Following tripkist patterns with daily and monthly retention logic
func CleanupHealthBackups(config *BackupConfig) error {
	files, err := os.ReadDir(config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Backup directory doesn't exist yet
		}
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	now := time.Now()
	dailyCutoff := now.AddDate(0, 0, -config.RetentionDays)

	for _, file := range files {
		// Only process health backup files
		if !strings.HasPrefix(file.Name(), "health_") || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}

		// Extract date from filename
		datePart := strings.TrimPrefix(file.Name(), "health_")
		datePart = strings.TrimSuffix(datePart, ".db")

		fileDate, err := time.Parse("20060102", datePart)
		if err != nil {
			continue // Skip files with invalid date format
		}

		// Keep files within retention period
		if fileDate.After(dailyCutoff) {
			continue
		}

		// Remove old backup file
		filePath := filepath.Join(config.BackupDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", file.Name(), err)
		}
	}

	return nil
}

// ListHealthBackups returns a list of available backup files sorted by date
// Following tripkist patterns for backup discovery
func ListHealthBackups(config *BackupConfig) ([]string, error) {
	files, err := os.ReadDir(config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "health_") && strings.HasSuffix(file.Name(), ".db") {
			backups = append(backups, file.Name())
		}
	}

	// Sort backups by date (filename sorting gives chronological order)
	sort.Strings(backups)
	return backups, nil
}

// RestoreHealthDatabase restores the health database from a specific backup file
// Following tripkist patterns with file copying and error handling
func RestoreHealthDatabase(backupFileName string, targetDBPath string, config *BackupConfig) error {
	backupPath := filepath.Join(config.BackupDir, backupFileName)

	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetDBPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Copy backup file to target location
	if err := copyFile(backupPath, targetDBPath); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	return nil
}

// FindBackupForDate finds a backup file for a specific date (YYYYMMDD format)
// Following tripkist patterns for date-based backup lookup
func FindBackupForDate(targetDate string, config *BackupConfig) (string, error) {
	backupFileName := "health_" + targetDate + ".db"
	backupPath := filepath.Join(config.BackupDir, backupFileName)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup for date %s not found", targetDate)
	}

	return backupFileName, nil
}

// GetBackupRetentionInfo returns current backup retention configuration
// Following tripkist patterns for configuration reporting
func GetBackupRetentionInfo(config *BackupConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":         config.Enabled,
		"backup_dir":      config.BackupDir,
		"retention_days":  config.RetentionDays,
		"backup_interval": config.BackupInterval.String(),
	}
}

// copyFile copies a file from src to dst following tripkist patterns
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	return err
}

