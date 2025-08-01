//go:build dev

package health

import (
	"os"
	"testing"
	"time"

	"github.com/thisdougb/health/internal/metrics"
)

// BenchmarkStateOperations benchmarks core state operations with system metrics always enabled
func BenchmarkStateOperations(b *testing.B) {
	// Create state with persistence disabled for pure memory performance
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
	
	state := NewState()
	state.SetConfig("benchmark-operations")
	defer state.Close()
	
	// System metrics are always enabled and running in background
	
	b.ResetTimer()
	
	b.Run("IncrMetric", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state.IncrMetric("requests")
		}
	})
	
	b.Run("AddMetric", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state.AddMetric("response_time", 123.45)
		}
	})
	
	b.Run("Dump", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state.Dump()
		}
	})
}

// BenchmarkStateWithPersistence benchmarks state operations with both system metrics and persistence
func BenchmarkStateWithPersistence(b *testing.B) {
	// Create temporary database
	tmpDir := b.TempDir()
	dbPath := tmpDir + "/bench.db"
	
	// Enable persistence
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "true")
	os.Setenv("HEALTH_DB_PATH", dbPath)
	os.Setenv("HEALTH_FLUSH_INTERVAL", "1s")
	defer func() {
		os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
		os.Unsetenv("HEALTH_DB_PATH")
		os.Unsetenv("HEALTH_FLUSH_INTERVAL")
	}()
	
	state := NewState()
	state.SetConfig("benchmark-with-persistence")
	defer state.Close()
	
	b.ResetTimer()
	
	b.Run("IncrMetric", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state.IncrMetric("requests")
		}
	})
	
	b.Run("AddMetric", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state.AddMetric("response_time", 123.45)
		}
	})
	
	b.Run("Dump", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state.Dump()
		}
	})
}

// BenchmarkSystemMetricsCollection benchmarks the system metrics collection itself
func BenchmarkSystemMetricsCollection(b *testing.B) {
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
	
	state := NewState()
	state.SetConfig("benchmark-system-collection")
	defer state.Close()
	
	b.ResetTimer()
	
	// Benchmark a single manual collection (system metrics always run in background)
	// This measures the cost of collecting system metrics
	for i := 0; i < b.N; i++ {
		// Force a system metrics collection by creating a new collector temporarily
		collector := metrics.NewSystemCollector(state.impl)
		collector.CollectOnce()
	}
}

// TestPerformanceImpact measures the performance of operations with system metrics enabled
func TestPerformanceImpact(t *testing.T) {
	// Since system metrics are always enabled, we test that operations are still fast
	const iterations = 10000
	
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
	
	state := NewState()
	state.SetConfig("perf-test")
	defer state.Close()
	
	// Test IncrMetric performance
	start := time.Now()
	for i := 0; i < iterations; i++ {
		state.IncrMetric("test_metric")
	}
	incrDuration := time.Since(start)
	
	// Test AddMetric performance  
	start = time.Now()
	for i := 0; i < iterations; i++ {
		state.AddMetric("test_value", float64(i))
	}
	addDuration := time.Since(start)
	
	// Test Dump performance
	start = time.Now()
	for i := 0; i < 100; i++ { // Fewer iterations for Dump as it's more expensive
		_ = state.Dump()
	}
	dumpDuration := time.Since(start)
	
	// Calculate average times
	avgIncrTime := float64(incrDuration.Nanoseconds()) / float64(iterations)
	avgAddTime := float64(addDuration.Nanoseconds()) / float64(iterations)
	avgDumpTime := float64(dumpDuration.Nanoseconds()) / 100.0
	
	t.Logf("Performance results for %d iterations:", iterations)
	t.Logf("  IncrMetric: %v total, %.2f ns/op", incrDuration, avgIncrTime)
	t.Logf("  AddMetric:  %v total, %.2f ns/op", addDuration, avgAddTime)
	t.Logf("  Dump (100): %v total, %.2f ns/op", dumpDuration, avgDumpTime)
	
	// Verify operations are fast enough for production use
	if avgIncrTime > 1000 { // 1000ns = 1μs
		t.Errorf("IncrMetric too slow: %.2f ns (should be < 1000ns)", avgIncrTime)
	}
	
	if avgAddTime > 1000 {
		t.Errorf("AddMetric too slow: %.2f ns (should be < 1000ns)", avgAddTime)
	}
	
	if avgDumpTime > 100000 { // 100μs should be reasonable for JSON serialization
		t.Errorf("Dump too slow: %.2f ns (should be < 100000ns)", avgDumpTime)
	}
	
	t.Logf("All operations meet performance requirements")
}

// NOTE: Claude must come back and think harder how we implement the test without using sleep
// TestSystemMetricsMemoryUsage tests that system metrics don't cause memory leaks
/*
func TestSystemMetricsMemoryUsage(t *testing.T) {
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")
	
	state := NewState()
	state.SetConfig("memory-test")
	defer state.Close()
	
	// Let system metrics run for a bit
	time.Sleep(100 * time.Millisecond)
	
	// Get initial memory stats
	var initialMem, finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)
	
	// Run some operations while system metrics are collecting
	for i := 0; i < 1000; i++ {
		state.IncrMetric("memory_test")
		state.AddMetric("value_test", float64(i))
		if i%100 == 0 {
			_ = state.Dump() // Generate some JSON output
		}
	}
	
	// Wait for any background collection
	time.Sleep(200 * time.Millisecond)
	
	// Get final memory stats
	runtime.GC()
	runtime.ReadMemStats(&finalMem)
	
	// Check memory usage didn't grow unreasonably
	var memGrowth int64
	if finalMem.Alloc > initialMem.Alloc {
		memGrowth = int64(finalMem.Alloc - initialMem.Alloc)
	} else {
		memGrowth = 0 // GC may have reduced memory usage
	}
	
	t.Logf("Memory usage:")
	t.Logf("  Initial: %d bytes", initialMem.Alloc)
	t.Logf("  Final:   %d bytes", finalMem.Alloc)
	t.Logf("  Growth:  %d bytes", memGrowth)
	
	// Memory growth should be reasonable (less than 1MB for this test)
	// Note: GC may actually reduce memory, so we only check if it grew significantly
	if memGrowth > 1024*1024 {
		t.Errorf("Memory growth too large: %d bytes (should be < 1MB)", memGrowth)
	} else {
		t.Logf("Memory usage is within acceptable bounds")
	}
}
*/