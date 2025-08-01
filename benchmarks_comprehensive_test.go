package health

import (
	"os"
	"testing"
)

// BenchmarkIncrMetric measures the performance of incrementing global metrics
// This is the most common operation, so it needs to be extremely fast
// Target: < 100 nanoseconds per operation
func BenchmarkIncrMetric(b *testing.B) {
	// Create state with memory backend for fastest possible performance
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-incr")
	
	// Reset timer to exclude setup time from benchmark
	b.ResetTimer()
	
	// Run the operation b.N times (Go benchmark framework decides how many)
	for i := 0; i < b.N; i++ {
		// This is what we're measuring - should be extremely fast
		state.IncrMetric("benchmark_counter")
	}
}

// BenchmarkIncrComponentMetric measures component-specific metric performance
// Should be similar to global metrics since they use the same underlying mechanism
// Target: < 120 nanoseconds per operation
func BenchmarkIncrComponentMetric(b *testing.B) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-component")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Alternate between different components to test map access patterns
		if i%2 == 0 {
			state.IncrComponentMetric("webserver", "requests")
		} else {
			state.IncrComponentMetric("database", "queries")
		}
	}
}

// BenchmarkAddMetric measures raw value metric performance with memory backend
// These are stored in memory temporarily then passed to persistence backend
// Target: < 50 nanoseconds per operation (no I/O in memory backend)
func BenchmarkAddMetric(b *testing.B) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-add")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Add varying values to simulate real usage
		state.AddMetric("response_time", float64(100+i%100))
	}
}

// BenchmarkAddComponentMetric measures component raw value performance
// Similar to AddMetric but with component organization
// Target: < 60 nanoseconds per operation
func BenchmarkAddComponentMetric(b *testing.B) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-add-component")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Rotate through different components and metrics
		components := []string{"webserver", "database", "cache"}
		metrics := []string{"response_time", "query_time", "hit_rate"}
		
		component := components[i%len(components)]
		metric := metrics[i%len(metrics)]
		value := float64(i % 1000)
		
		state.AddComponentMetric(component, metric, value)
	}
}

// BenchmarkDump measures JSON export performance
// This is called by HTTP health check endpoints, so it must be fast
// Target: < 5 microseconds for typical usage (few dozen metrics)
func BenchmarkDump(b *testing.B) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-dump")
	
	// Pre-populate with realistic number of metrics
	// Simulate a typical microservice with multiple components
	components := []string{"webserver", "database", "cache", "queue", "auth"}
	metrics := []string{"requests", "errors", "timeouts", "retries"}
	
	for _, component := range components {
		for _, metric := range metrics {
			for i := 0; i < 10; i++ {
				state.IncrComponentMetric(component, metric)
			}
		}
	}
	
	// Also add some global metrics
	for i := 0; i < 20; i++ {
		state.IncrMetric("global_requests")
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// This is what we're measuring - JSON serialization performance
		_ = state.Dump()
	}
}

// BenchmarkSQLitePersistence measures SQLite backend performance
// This shows the overhead of persistence vs memory-only storage
// Important: This measures async queuing time, not actual disk I/O time
func BenchmarkSQLitePersistence(b *testing.B) {
	// Set up temporary SQLite database
	tmpFile := "/tmp/health_benchmark.db"
	defer os.Remove(tmpFile)
	
	// Configure SQLite backend
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_FLUSH_INTERVAL", "1s")
	os.Setenv("HEALTH_BATCH_SIZE", "100")
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_FLUSH_INTERVAL")
		os.Unsetenv("HEALTH_BATCH_SIZE")
	}()
	
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-sqlite")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Mix of counter and value metrics to simulate real usage
		state.IncrMetric("sqlite_counter")
		state.AddMetric("sqlite_value", float64(i))
	}
	
	// Note: This benchmark measures queuing time, not I/O time
	// Actual disk writes happen asynchronously in background
}

// BenchmarkConcurrentAccess measures performance under concurrent load
// Critical for understanding how the package performs in multi-threaded applications
// This benchmarks the mutex overhead and contention
func BenchmarkConcurrentAccess(b *testing.B) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-concurrent")
	
	b.ResetTimer()
	
	// Run benchmark with multiple goroutines accessing simultaneously
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine runs this loop
		for pb.Next() {
			// Mix of read and write operations that happen in real applications
			state.IncrMetric("concurrent_counter")
			state.AddMetric("concurrent_value", 123.45)
			
			// Occasional JSON dump (read operation) during writes
			// This tests reader performance while writers are active
			if pb.Next() {
				_ = state.Dump()
			}
		}
	})
}

// BenchmarkSystemMetricsOverhead measures automatic system metrics overhead
// These run every minute in background, should have minimal impact
func BenchmarkSystemMetricsOverhead(b *testing.B) {
	// Use direct SystemCollector for isolated measurement
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-system")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate system metrics collection
		// In real usage, this happens automatically every minute
		state.AddMetric("cpu_percent", 45.2)
		state.AddMetric("memory_bytes", 1048576)
		state.AddMetric("goroutines", 12)
		state.AddMetric("uptime_seconds", 3600)
		state.AddMetric("health_data_size", 2048)
	}
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// BenchmarkDataExtraction measures admin function performance
// These are used for historical analysis and monitoring dashboards
// Should be fast enough for interactive use (< 1 millisecond)
func BenchmarkDataExtraction(b *testing.B) {
	// Set up SQLite backend with test data
	tmpFile := "/tmp/health_extract_benchmark.db"
	defer os.Remove(tmpFile)
	
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_FLUSH_INTERVAL", "100ms")
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_FLUSH_INTERVAL")
	}()
	
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-extract")
	
	// Pre-populate with realistic dataset
	components := []string{"webserver", "database", "cache"}
	for _, component := range components {
		for i := 0; i < 100; i++ {
			state.IncrComponentMetric(component, "requests")
			state.AddComponentMetric(component, "response_time", float64(100+i))
		}
	}
	
	// Wait for data to be persisted
	time.Sleep(200 * time.Millisecond)
	
	// Define time range for extraction
	start := time.Now().Add(-10 * time.Minute)
	end := time.Now()
	
	b.ResetTimer()
	
	// Benchmark different extraction operations
	b.Run("ExtractMetricsByTimeRange", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := handlers.ExtractMetricsByTimeRange(state, "webserver", start, end)
			if err != nil {
				b.Fatalf("ExtractMetricsByTimeRange failed: %v", err)
			}
		}
	})
	
	b.Run("GetHealthSummary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := handlers.GetHealthSummary(state, start, end)
			if err != nil {
				b.Fatalf("GetHealthSummary failed: %v", err)
			}
		}
	})
	
	b.Run("ListAvailableComponents", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := handlers.ListAvailableComponents(state)
			if err != nil {
				b.Fatalf("ListAvailableComponents failed: %v", err)
			}
		}
	})
}
*/

// NOTE: Claude must come back and think harder how we implement the test without using sleep
/*
// BenchmarkBackupOperations measures backup performance
// Backups should be fast enough for graceful shutdown (< 1 second typical)
func BenchmarkBackupOperations(b *testing.B) {
	// Set up SQLite with backup enabled
	tmpFile := "/tmp/health_backup_benchmark.db"
	backupDir := "/tmp/health_backup_bench"
	defer func() {
		os.Remove(tmpFile)
		os.RemoveAll(backupDir)
	}()
	
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", tmpFile)
	os.Setenv("HEALTH_BACKUP_ENABLED", "true")
	os.Setenv("HEALTH_BACKUP_DIR", backupDir)
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_BACKUP_ENABLED")
		os.Unsetenv("HEALTH_BACKUP_DIR")
	}()
	
	// Create state and populate with data
	state := NewState()
	state.SetConfig("benchmark-backup")
	
	// Add substantial amount of data to make backup meaningful
	for i := 0; i < 1000; i++ {
		state.IncrMetric("backup_counter")
		state.AddMetric("backup_value", float64(i))
		state.IncrComponentMetric("webserver", "requests")
		state.AddComponentMetric("database", "query_time", float64(20+i%50))
	}
	
	// Wait for data to be persisted
	time.Sleep(1 * time.Second)
	
	b.ResetTimer()
	
	// Benchmark backup creation
	// Note: We can only run this once per b.N since each backup creates a unique file
	for i := 0; i < b.N; i++ {
		// Get storage manager and create backup
		manager := state.GetStorageManager()
		if manager == nil {
			b.Fatal("Storage manager is nil")
		}
		
		err := manager.CreateBackup()
		if err != nil {
			b.Fatalf("Backup creation failed: %v", err)
		}
	}
	
	state.Close()
}
*/

// BenchmarkMemoryUsage measures memory allocation patterns
// Important for understanding memory overhead and potential leaks
func BenchmarkMemoryUsage(b *testing.B) {
	// This benchmark measures allocations per operation
	b.ReportAllocs()
	
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-memory")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Operations that should minimize allocations
		state.IncrMetric("memory_test")
		state.AddMetric("memory_value", float64(i))
		
		// JSON dump allocates memory for serialization
		if i%100 == 0 {
			_ = state.Dump()
		}
	}
}

// BenchmarkStorageBackends compares memory vs SQLite backend performance
// This helps understand the performance trade-off of persistence
func BenchmarkStorageBackends(b *testing.B) {
	// Test memory backend
	b.Run("MemoryBackend", func(b *testing.B) {
		state := NewState() // Uses memory backend by default
		defer state.Close()
		
		state.SetConfig("benchmark-memory-backend")
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			state.IncrMetric("backend_counter")
			state.AddMetric("backend_value", float64(i))
		}
	})
	
	// Test SQLite backend
	b.Run("SQLiteBackend", func(b *testing.B) {
		tmpFile := "/tmp/health_backend_benchmark.db"
		defer os.Remove(tmpFile)
		
		os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
		os.Setenv("HEALTH_DB_PATH", tmpFile)
		defer func() {
			os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
			os.Unsetenv("HEALTH_DB_PATH")
		}()
		
		state := NewState()
		defer state.Close()
		
		state.SetConfig("benchmark-sqlite-backend")
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			state.IncrMetric("backend_counter")
			state.AddMetric("backend_value", float64(i))
		}
	})
}

// BenchmarkRealWorldScenario simulates realistic application usage patterns
// This combines all operations in proportions similar to production usage
func BenchmarkRealWorldScenario(b *testing.B) {
	state := NewState()
	defer state.Close()
	
	state.SetConfig("benchmark-real-world")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate realistic application behavior:
		// - 70% counter increments (most common)
		// - 25% value measurements  
		// - 5% JSON exports (health checks)
		
		operation := i % 20
		
		switch {
		case operation < 14: // 70% - counter increments
			if operation%2 == 0 {
				state.IncrMetric("requests")
			} else {
				state.IncrComponentMetric("webserver", "http_requests")
			}
			
		case operation < 19: // 25% - value measurements
			if operation%2 == 0 {
				state.AddMetric("response_time", float64(100+operation*10))
			} else {
				state.AddComponentMetric("database", "query_time", float64(20+operation*2))
			}
			
		default: // 5% - JSON exports
			_ = state.Dump()
		}
	}
}