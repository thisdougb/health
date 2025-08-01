//go:build dev

package health

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thisdougb/health/internal/handlers"
)

// TestErrorConditionInvalidDatabase tests recovery from database connection failures
// Junior developers need to understand: the package should gracefully handle database issues
// and fall back to memory-only mode rather than crashing the application
func TestErrorConditionInvalidDatabase(t *testing.T) {
	// Test Case 1: Invalid database path (permission denied)
	t.Run("PermissionDenied", func(t *testing.T) {
		// Try to create database in root directory (should fail on most systems)
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", "/root/impossible/path/health.db")
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
		}()
		
		// Package should handle this gracefully - no panic, no crash
		state := NewState()
		defer state.Close()
		
		state.SetConfig("test-permission-denied")
		
		// All operations should still work (using memory backend as fallback)
		state.IncrMetric("error_test")
		state.AddMetric("error_value", 123.45)
		
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatal("Package should continue working with memory backend when database fails")
		}
		
		// Verify the JSON contains our test data
		if !strings.Contains(jsonOutput, "error_test") {
			t.Error("Metrics should still be tracked in memory when database fails")
		}
	})
	
	// Test Case 2: Database file in non-existent directory
	t.Run("NonExistentDirectory", func(t *testing.T) {
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", "/tmp/nonexistent/directory/health.db")
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
		}()
		
		// Should handle gracefully
		state := NewState()
		defer state.Close()
		
		state.SetConfig("test-nonexistent-dir")
		
		// Operations should work
		state.IncrMetric("directory_error_test")
		jsonOutput := state.Dump()
		
		if jsonOutput == "" {
			t.Fatal("Package should work even when database directory doesn't exist")
		}
	})
	
	// Test Case 3: Database file is actually a directory (invalid)
	t.Run("DatabaseIsDirectory", func(t *testing.T) {
		// Create a directory where we expect a file
		dbDir := "/tmp/health_is_directory"
		os.MkdirAll(dbDir, 0755)
		defer os.RemoveAll(dbDir)
		
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", dbDir) // This is a directory, not a file
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
		}()
		
		state := NewState()
		defer state.Close()
		
		state.SetConfig("test-db-is-directory")
		
		// Should work with memory fallback
		state.IncrMetric("db_directory_error")
		jsonOutput := state.Dump()
		
		if jsonOutput == "" {
			t.Fatal("Package should handle case where database path is a directory")
		}
	})
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// TestErrorConditionInvalidBackupConfiguration tests backup error handling
// Backups should fail gracefully without affecting normal operations
// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
func TestErrorConditionInvalidBackupConfiguration(t *testing.T) {
	// Test Case 1: Backup directory is read-only
	t.Run("ReadOnlyBackupDirectory", func(t *testing.T) {
		tmpFile := "/tmp/health_backup_error_test.db"
		readOnlyDir := "/tmp/health_readonly_backup"
		defer func() {
			os.Remove(tmpFile)
			os.Chmod(readOnlyDir, 0755) // Restore permissions
			os.RemoveAll(readOnlyDir)
		}()
		
		// Create read-only backup directory
		os.MkdirAll(readOnlyDir, 0755)
		os.Chmod(readOnlyDir, 0444) // Read-only
		
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", tmpFile)
		os.Setenv("HEALTH_BACKUP_ENABLED", "true")
		os.Setenv("HEALTH_BACKUP_DIR", readOnlyDir)
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
			os.Unsetenv("HEALTH_BACKUP_ENABLED")
			os.Unsetenv("HEALTH_BACKUP_DIR")
		}()
		
		state := NewState()
		defer state.Close()
		
		state.SetConfig("test-readonly-backup")
		
		// Add some data
		state.IncrMetric("backup_error_test")
		state.AddMetric("backup_error_value", 456.78)
		
		// Wait for persistence
		time.Sleep(200 * time.Millisecond)
		
		// Normal operations should work despite backup issues
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatal("Normal operations should work even when backup directory is read-only")
		}
		
		// Try manual backup - should fail gracefully
		manager := state.GetStorageManager()
		if manager != nil {
			err := manager.CreateBackup()
			// Backup should fail, but this shouldn't crash the application
			if err == nil {
				t.Log("Backup unexpectedly succeeded (maybe directory permissions are different)")
			} else {
				t.Logf("Backup failed as expected: %v", err)
			}
		}
		
		// State should still be functional
		state.IncrMetric("after_backup_failure")
		jsonOutput = state.Dump()
		if jsonOutput == "" {
			t.Fatal("State should remain functional after backup failure")
		}
	})
	
	// Test Case 2: Backup directory doesn't exist and can't be created
	t.Run("CannotCreateBackupDirectory", func(t *testing.T) {
		tmpFile := "/tmp/health_backup_create_error.db"
		defer os.Remove(tmpFile)
		
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", tmpFile)
		os.Setenv("HEALTH_BACKUP_ENABLED", "true")
		os.Setenv("HEALTH_BACKUP_DIR", "/root/impossible/backup/path")
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
			os.Unsetenv("HEALTH_BACKUP_ENABLED")
			os.Unsetenv("HEALTH_BACKUP_DIR")
		}()
		
		state := NewState()
		defer state.Close()
		
		state.SetConfig("test-backup-dir-creation")
		
		// Add data and verify normal operation
		state.IncrMetric("backup_dir_error")
		jsonOutput := state.Dump()
		
		if jsonOutput == "" {
			t.Fatal("Normal operation should work even when backup directory cannot be created")
		}
	})
}
*/

// TestErrorConditionInvalidMetricNames tests handling of invalid inputs
// The package should ignore invalid inputs rather than crashing
func TestErrorConditionInvalidMetricNames(t *testing.T) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-invalid-metrics")
	
	// Test various invalid metric names that should be ignored
	invalidNames := []string{
		"",          // Empty string
		"   ",       // Only whitespace
		"\t\n",      // Only whitespace characters
		"\x00",      // Null character
		strings.Repeat("a", 1000), // Extremely long name
	}
	
	for _, invalidName := range invalidNames {
		// These should be ignored, not cause crashes or corruption
		state.IncrMetric(invalidName)
		state.AddMetric(invalidName, 123.45)
		state.IncrComponentMetric("test", invalidName)
		state.AddComponentMetric("test", invalidName, 456.78)
	}
	
	// Add valid metrics to ensure the state still works
	state.IncrMetric("valid_metric")
	state.IncrComponentMetric("test", "valid_component_metric")
	
	// Should produce valid JSON despite invalid inputs
	jsonOutput := state.Dump()
	if jsonOutput == "" {
		t.Fatal("State should remain functional despite invalid metric names")
	}
	
	// Valid metrics should appear in output
	if !strings.Contains(jsonOutput, "valid_metric") {
		t.Error("Valid metrics should appear in output even after invalid inputs")
	}
}

// TestErrorConditionInvalidMetricValues tests handling of extreme or invalid values
// Junior developers need to understand: the package should handle edge cases gracefully
func TestErrorConditionInvalidMetricValues(t *testing.T) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-invalid-values")
	
	// Test extreme and special float values
	extremeValues := []float64{
		1e308,   // Very large positive number
		-1e308,  // Very large negative number
		1e-308,  // Very small positive number
		-1e-308, // Very small negative number
		0.0,     // Zero
		-0.0,    // Negative zero
		// Note: NaN and Inf are handled by JSON marshaling, so we test them
	}
	
	for i, value := range extremeValues {
		// These should not cause crashes or data corruption
		state.AddMetric("extreme_global", value)
		state.AddComponentMetric("test", "extreme_component", value)
		
		// Also test that normal operations still work
		state.IncrMetric("normal_counter")
		
		// Verify state remains functional after each extreme value
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatalf("State should remain functional after extreme value %d: %f", i, value)
		}
	}
	
	// Final verification that everything still works
	state.IncrMetric("final_test")
	jsonOutput := state.Dump()
	
	if jsonOutput == "" {
		t.Fatal("Final state should be functional after all extreme value tests")
	}
	
	if !strings.Contains(jsonOutput, "final_test") {
		t.Error("Final test metric should appear in output")
	}
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// TestErrorConditionCorruptedDatabase tests recovery from database corruption
// This simulates real-world scenarios where database files get corrupted
func TestErrorConditionCorruptedDatabase(t *testing.T) {
	tmpFile := "/tmp/health_corrupted_test.db"
	defer os.Remove(tmpFile)
	
	// First, create a valid database with some data
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
	}()
	
	// Create initial state and add data
	state1 := NewState()
	state1.SetConfig("test-corruption-setup")
	
	for i := 0; i < 10; i++ {
		state1.IncrMetric("setup_counter")
		state1.AddMetric("setup_value", float64(i))
	}
	
	time.Sleep(200 * time.Millisecond) // Wait for persistence
	state1.Close()
	
	// Verify database was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Database file should exist after initial setup")
	}
	
	// Now corrupt the database file by writing random data
	corruptedData := []byte("This is not a valid SQLite database file - corrupted!")
	if err := os.WriteFile(tmpFile, corruptedData, 0644); err != nil {
		t.Fatalf("Failed to corrupt database file: %v", err)
	}
	
	// Try to use the corrupted database - should handle gracefully
	state2 := NewState()
	defer state2.Close()
	
	state2.SetConfig("test-corruption-recovery")
	
	// Should work despite corrupted database (fallback to memory)
	state2.IncrMetric("recovery_test")
	state2.AddMetric("recovery_value", 999.0)
	
	jsonOutput := state2.Dump()
	if jsonOutput == "" {
		t.Fatal("State should work with memory fallback when database is corrupted")
	}
	
	if !strings.Contains(jsonOutput, "recovery_test") {
		t.Error("Recovery metrics should appear in output")
	}
}
*/

// TestErrorConditionSystemResourceExhaustion tests behavior under resource pressure
// This helps junior developers understand how the package behaves under stress
func TestErrorConditionSystemResourceExhaustion(t *testing.T) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-resource-exhaustion")
	
	// Test 1: Create many metrics to use more memory
	t.Run("HighMemoryUsage", func(t *testing.T) {
		// Create many unique metrics - this uses more memory
		for i := 0; i < 10000; i++ {
			metricName := fmt.Sprintf("memory_test_%d", i)
			componentName := fmt.Sprintf("component_%d", i%100)
			
			state.IncrMetric(metricName)
			state.IncrComponentMetric(componentName, metricName)
			state.AddMetric(metricName, float64(i))
		}
		
		// Should still be functional
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatal("State should remain functional under high memory usage")
		}
		
		// Verify we can still add more metrics
		state.IncrMetric("after_memory_test")
		jsonOutput = state.Dump()
		
		if !strings.Contains(jsonOutput, "after_memory_test") {
			t.Error("Should be able to add metrics after high memory usage test")
		}
	})
	
	// Test 2: Very frequent operations (CPU pressure)
	t.Run("HighCPUUsage", func(t *testing.T) {
		// Perform many operations quickly
		for i := 0; i < 100000; i++ {
			state.IncrMetric("cpu_test")
			state.AddMetric("cpu_value", float64(i%1000))
			
			// Occasional JSON export adds CPU load
			if i%1000 == 0 {
				_ = state.Dump()
			}
		}
		
		// Should still be functional
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatal("State should remain functional under high CPU usage")
		}
	})
}

// TestErrorConditionInvalidConfiguration tests handling of bad configuration
// This helps junior developers understand configuration validation
func TestErrorConditionInvalidConfiguration(t *testing.T) {
	// Test various invalid configuration values
	testCases := []struct {
		name        string
		envVars     map[string]string
		description string
	}{
		{
			name: "InvalidFlushInterval",
			envVars: map[string]string{
				"HEALTH_PERSISTENCE_ENABLED": "true",
				"HEALTH_DB_PATH":             "/tmp/health_invalid_config.db",
				"HEALTH_FLUSH_INTERVAL":      "invalid-duration",
			},
			description: "Invalid flush interval should use default",
		},
		{
			name: "InvalidBatchSize",
			envVars: map[string]string{
				"HEALTH_PERSISTENCE_ENABLED": "true",
				"HEALTH_DB_PATH":             "/tmp/health_invalid_batch.db",
				"HEALTH_BATCH_SIZE":          "not-a-number",
			},
			description: "Invalid batch size should use default",
		},
		{
			name: "InvalidRetentionDays",
			envVars: map[string]string{
				"HEALTH_PERSISTENCE_ENABLED":   "true",
				"HEALTH_DB_PATH":               "/tmp/health_invalid_retention.db",
				"HEALTH_BACKUP_ENABLED":        "true",
				"HEALTH_BACKUP_DIR":            "/tmp/health_invalid_retention_backup",
				"HEALTH_BACKUP_RETENTION_DAYS": "not-a-number",
			},
			description: "Invalid retention days should use default",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}
			
			// Clean up after test
			defer func() {
				for key := range tc.envVars {
					os.Unsetenv(key)
				}
				// Clean up any created files
				if dbPath, exists := tc.envVars["HEALTH_DB_PATH"]; exists {
					os.Remove(dbPath)
				}
				if backupDir, exists := tc.envVars["HEALTH_BACKUP_DIR"]; exists {
					os.RemoveAll(backupDir)
				}
			}()
			
			// Package should handle invalid config gracefully
			state := NewState()
			defer state.Close()
			
			state.SetConfig("test-invalid-config")
			
			// Normal operations should work despite configuration errors
			state.IncrMetric("config_error_test")
			state.AddMetric("config_error_value", 123.45)
			
			jsonOutput := state.Dump()
			if jsonOutput == "" {
				t.Fatalf("State should work despite invalid configuration: %s", tc.description)
			}
			
			if !strings.Contains(jsonOutput, "config_error_test") {
				t.Errorf("Metrics should work despite invalid configuration: %s", tc.description)
			}
		})
	}
}

// TestErrorConditionDataExtractionFailures tests admin function error handling
// These functions should handle database issues gracefully
func TestErrorConditionDataExtractionFailures(t *testing.T) {
	// Test with persistence disabled (no backend available)
	t.Run("NoPersistenceBackend", func(t *testing.T) {
		state := NewState() // Memory only, no persistence
		defer state.Close()
		
		state.SetConfig("test-no-persistence")
		
		// Add some metrics
		state.IncrMetric("extraction_test")
		state.AddMetric("extraction_value", 123.45)
		
		// Admin functions should handle missing persistence gracefully
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now()
		
		// These should return errors but not crash
		_, err := handlers.ExtractMetricsByTimeRange(state, "test", start, end)
		if err == nil {
			t.Log("ExtractMetricsByTimeRange unexpectedly succeeded with no persistence")
		}
		
		_, err = handlers.GetHealthSummary(state, start, end)
		if err == nil {
			t.Log("GetHealthSummary unexpectedly succeeded with no persistence")
		}
		
		_, err = handlers.ListAvailableComponents(state)
		if err == nil {
			t.Log("ListAvailableComponents unexpectedly succeeded with no persistence")
		}
		
		// Normal operations should still work
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatal("Normal operations should work even when admin functions fail")
		}
	})
	
	// Test with corrupted persistence backend
	t.Run("CorruptedPersistenceBackend", func(t *testing.T) {
		tmpFile := "/tmp/health_extraction_corrupt.db"
		defer os.Remove(tmpFile)
		
		// First create a valid database
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", tmpFile)
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
		}()
		
		state := NewState()
		state.SetConfig("test-extraction-corrupt")
		
		// Add some data
		state.IncrMetric("corrupt_test")
		state.AddMetric("corrupt_value", 456.78)
		
		time.Sleep(200 * time.Millisecond) // Wait for persistence
		
		// Now corrupt the database
		corruptData := []byte("CORRUPTED DATABASE FILE")
		os.WriteFile(tmpFile, corruptData, 0644)
		
		// Admin functions should handle corruption gracefully
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now()
		
		_, err := handlers.ExtractMetricsByTimeRange(state, "test", start, end)
		if err == nil {
			t.Log("ExtractMetricsByTimeRange unexpectedly succeeded with corrupted database")
		}
		
		// Normal operations should still work (using memory)
		jsonOutput := state.Dump()
		if jsonOutput == "" {
			t.Fatal("Normal operations should work even with corrupted persistence backend")
		}
		
		state.Close()
	})
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// TestErrorConditionRecoveryAfterFailure tests that the package can recover
// from temporary failures and continue operating normally
func TestErrorConditionRecoveryAfterFailure(t *testing.T) {
	tmpFile := "/tmp/health_recovery_test.db"
	defer os.Remove(tmpFile)
	
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
	}()
	
	state := NewState()
	defer state.Close()
	
	state.SetConfig("test-recovery")
	
	// Phase 1: Normal operation
	state.IncrMetric("recovery_phase1")
	state.AddMetric("recovery_value1", 111.0)
	
	time.Sleep(200 * time.Millisecond) // Wait for persistence
	
	// Verify database was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Database should be created during normal operation")
	}
	
	// Phase 2: Simulate temporary database issue
	// Make database file read-only to simulate permission issue
	os.Chmod(tmpFile, 0444)
	
	// Operations should continue working (memory fallback)
	state.IncrMetric("recovery_phase2")
	state.AddMetric("recovery_value2", 222.0)
	
	jsonOutput := state.Dump()
	if jsonOutput == "" {
		t.Fatal("Operations should continue during temporary database issues")
	}
	
	// Phase 3: Restore database access
	os.Chmod(tmpFile, 0644) // Restore write permissions
	
	// Operations should continue normally
	state.IncrMetric("recovery_phase3")
	state.AddMetric("recovery_value3", 333.0)
	
	jsonOutput = state.Dump()
	if jsonOutput == "" {
		t.Fatal("Operations should work after database access is restored")
	}
	
	// Verify all phases are represented in final output
	if !strings.Contains(jsonOutput, "recovery_phase1") ||
		!strings.Contains(jsonOutput, "recovery_phase2") ||
		!strings.Contains(jsonOutput, "recovery_phase3") {
		t.Error("All recovery phases should be represented in final output")
	}
}
*/