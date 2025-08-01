//go:build dev

package health

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestMemorySizing calculates memory requirements for different metric collection rates
func TestMemorySizing(t *testing.T) {
	// Test scenarios: metrics per second for 1 hour
	scenarios := []struct {
		name           string
		metricsPerSec  int
		durationHours  int
	}{
		{"Low Volume", 100, 1},
		{"Medium Volume", 1000, 1},
		{"High Volume", 10000, 1},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Disable persistence for pure memory testing
			os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
			defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")

			// Force garbage collection and get baseline memory
			runtime.GC()
			var baseMem runtime.MemStats
			runtime.ReadMemStats(&baseMem)

			state := NewState()
			state.SetConfig(fmt.Sprintf("memory-sizing-%s", scenario.name))
			defer state.Close()

			// Calculate total metrics to collect
			totalSeconds := scenario.durationHours * 3600
			totalMetrics := scenario.metricsPerSec * totalSeconds

			t.Logf("Scenario: %s", scenario.name)
			t.Logf("  Rate: %d metrics/second", scenario.metricsPerSec)
			t.Logf("  Duration: %d hours", scenario.durationHours)
			t.Logf("  Total metrics: %d", totalMetrics)

			// Collect metrics without time delays - simulate the volume
			start := time.Now()
			
			// Mix of counter metrics and value metrics (50/50 split)
			counterMetrics := totalMetrics / 2
			valueMetrics := totalMetrics / 2

			// Collect counter metrics (IncrMetric)
			for i := 0; i < counterMetrics; i++ {
				metricName := fmt.Sprintf("counter_%d", i%100) // 100 unique counter names
				state.IncrMetric(metricName)
				
				// Also test component metrics (20% of counters)
				if i%5 == 0 {
					componentName := fmt.Sprintf("component_%d", i%10) // 10 unique components
					state.IncrComponentMetric(componentName, metricName)
				}
			}

			// Collect value metrics (AddMetric)
			for i := 0; i < valueMetrics; i++ {
				metricName := fmt.Sprintf("value_%d", i%100) // 100 unique value names
				value := float64(i) * 1.23 // Some variation in values
				state.AddMetric(metricName, value)
				
				// Also test component metrics (20% of values)
				if i%5 == 0 {
					componentName := fmt.Sprintf("component_%d", i%10) // 10 unique components
					state.AddComponentMetric(componentName, metricName, value)
				}
			}

			collectionTime := time.Since(start)

			// Force garbage collection and measure memory usage
			runtime.GC()
			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			// Calculate memory usage
			memoryUsed := int64(afterMem.Alloc - baseMem.Alloc)
			if memoryUsed < 0 {
				memoryUsed = int64(afterMem.Alloc) // GC might have reduced baseline
			}

			// Test a few Dump operations to measure JSON overhead
			dumpStart := time.Now()
			jsonSize := 0
			for i := 0; i < 10; i++ {
				json := state.Dump()
				jsonSize = len(json)
			}
			avgDumpTime := time.Since(dumpStart) / 10

			// Calculate metrics
			bytesPerMetric := float64(memoryUsed) / float64(totalMetrics)
			metricsPerSecond := float64(totalMetrics) / collectionTime.Seconds()

			// Report results
			t.Logf("Memory Usage Results:")
			t.Logf("  Collection time: %v", collectionTime)
			t.Logf("  Total memory used: %d bytes (%.2f MB)", memoryUsed, float64(memoryUsed)/(1024*1024))
			t.Logf("  Memory per metric: %.2f bytes", bytesPerMetric)
			t.Logf("  Actual collection rate: %.0f metrics/second", metricsPerSecond)
			t.Logf("  JSON size: %d bytes (%.2f KB)", jsonSize, float64(jsonSize)/1024)
			t.Logf("  Avg JSON generation time: %v", avgDumpTime)

			// Calculate extrapolations
			calculateExtrapolations(t, scenario.metricsPerSec, memoryUsed, totalMetrics)

			// Verify performance is acceptable
			if metricsPerSecond < float64(scenario.metricsPerSec) {
				t.Logf("Note: Achieved rate (%.0f/sec) is lower than target (%d/sec) due to test overhead", 
					metricsPerSecond, scenario.metricsPerSec)
			}

			// Memory usage should be reasonable
			maxExpectedMB := float64(totalMetrics) * 0.001 // Very rough estimate: 1KB per 1000 metrics
			actualMB := float64(memoryUsed) / (1024 * 1024)
			if actualMB > maxExpectedMB && actualMB > 10 { // Don't fail on small memory usage
				t.Logf("Warning: Memory usage (%.2f MB) higher than rough estimate (%.2f MB)", 
					actualMB, maxExpectedMB)
			}
		})
	}
}

// calculateExtrapolations projects memory usage for longer time periods
func calculateExtrapolations(t *testing.T, metricsPerSec int, memoryUsedBytes int64, totalMetrics int) {
	// Calculate bytes per metric
	bytesPerMetric := float64(memoryUsedBytes) / float64(totalMetrics)
	
	// Time periods to calculate
	periods := []struct {
		name     string
		hours    int
		days     int
		months   int
	}{
		{"1 Hour", 1, 0, 0},
		{"1 Day", 24, 1, 0},
		{"1 Week", 168, 7, 0},
		{"1 Month", 720, 30, 1},
		{"12 Months", 8760, 365, 12},
	}

	t.Logf("Memory Projections for %d metrics/second:", metricsPerSec)
	t.Logf("  (Based on %.2f bytes per metric)", bytesPerMetric)

	for _, period := range periods {
		// Calculate total metrics for this period
		totalSeconds := period.hours * 3600
		periodMetrics := int64(metricsPerSec) * int64(totalSeconds)
		
		// Calculate memory usage
		memoryBytes := float64(periodMetrics) * bytesPerMetric
		memoryMB := memoryBytes / (1024 * 1024)
		memoryGB := memoryMB / 1024

		// Format output based on size
		var memoryStr string
		if memoryGB >= 1 {
			memoryStr = fmt.Sprintf("%.2f GB", memoryGB)
		} else if memoryMB >= 1 {
			memoryStr = fmt.Sprintf("%.2f MB", memoryMB)
		} else {
			memoryStr = fmt.Sprintf("%.0f KB", memoryBytes/1024)
		}

		t.Logf("  %s: %d metrics → %s", period.name, periodMetrics, memoryStr)
	}
}

// TestMemoryGrowthPattern tests how memory usage grows with increasing metric counts
func TestMemoryGrowthPattern(t *testing.T) {
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")

	// Test different metric counts to understand growth pattern
	metricCounts := []int{100, 500, 1000, 5000, 10000, 50000, 100000}
	
	t.Logf("Memory Growth Pattern Analysis:")
	t.Logf("Metrics\tMemory (MB)\tPer Metric (bytes)")

	for _, count := range metricCounts {
		func() {
			runtime.GC()
			var baseMem runtime.MemStats
			runtime.ReadMemStats(&baseMem)

			state := NewState()
			state.SetConfig(fmt.Sprintf("growth-test-%d", count))
			defer state.Close()

			// Add metrics with variety (counters and values)
			for i := 0; i < count; i++ {
				if i%2 == 0 {
					state.IncrMetric(fmt.Sprintf("counter_%d", i%100))
				} else {
					state.AddMetric(fmt.Sprintf("value_%d", i%100), float64(i)*1.23)
				}
			}

			runtime.GC()
			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			memoryUsed := int64(afterMem.Alloc - baseMem.Alloc)
			if memoryUsed < 0 {
				memoryUsed = int64(afterMem.Alloc)
			}

			memoryMB := float64(memoryUsed) / (1024 * 1024)
			bytesPerMetric := float64(memoryUsed) / float64(count)

			t.Logf("%d\t%.3f\t\t%.1f", count, memoryMB, bytesPerMetric)
		}()
	}
}

// TestUniqueMetricNames tests memory usage with different numbers of unique metric names
func TestUniqueMetricNames(t *testing.T) {
	os.Setenv("HEALTH_PERSISTENCE_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PERSISTENCE_ENABLED")

	scenarios := []struct {
		name           string
		totalMetrics   int
		uniqueNames    int
	}{
		{"Few Names, Many Values", 10000, 10},
		{"Moderate Names", 10000, 100}, 
		{"Many Names", 10000, 1000},
		{"Unique Every Time", 10000, 10000},
	}

	t.Logf("Impact of Unique Metric Names on Memory Usage:")

	for _, scenario := range scenarios {
		func() {
			runtime.GC()
			var baseMem runtime.MemStats
			runtime.ReadMemStats(&baseMem)

			state := NewState()
			state.SetConfig(fmt.Sprintf("unique-names-%s", scenario.name))
			defer state.Close()

			// Add metrics with controlled uniqueness
			for i := 0; i < scenario.totalMetrics; i++ {
				metricName := fmt.Sprintf("metric_%d", i%scenario.uniqueNames)
				if i%2 == 0 {
					state.IncrMetric(metricName)
				} else {
					state.AddMetric(metricName, float64(i)*1.23)
				}
			}

			runtime.GC()
			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			memoryUsed := int64(afterMem.Alloc - baseMem.Alloc)
			if memoryUsed < 0 {
				memoryUsed = int64(afterMem.Alloc)
			}

			memoryMB := float64(memoryUsed) / (1024 * 1024)
			bytesPerMetric := float64(memoryUsed) / float64(scenario.totalMetrics)

			t.Logf("  %s:", scenario.name)
			t.Logf("    Total metrics: %d, Unique names: %d", scenario.totalMetrics, scenario.uniqueNames)
			t.Logf("    Memory: %.3f MB (%.1f bytes/metric)", memoryMB, bytesPerMetric)
		}()
	}
}