//go:build longrunning

package health

import (
	"fmt"
	"strings"
	"testing"
)

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