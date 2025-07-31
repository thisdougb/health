package metrics

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// MockState implements StateInterface for testing
type MockState struct {
	metrics map[string]map[string]float64
	counters map[string]map[string]int
	mu      sync.Mutex
}

func NewMockState() *MockState {
	return &MockState{
		metrics:  make(map[string]map[string]float64),
		counters: make(map[string]map[string]int),
	}
}

func (ms *MockState) AddMetric(component, name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if ms.metrics[component] == nil {
		ms.metrics[component] = make(map[string]float64)
	}
	ms.metrics[component][name] = value
}

func (ms *MockState) IncrComponentMetric(component, name string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if ms.counters[component] == nil {
		ms.counters[component] = make(map[string]int)
	}
	ms.counters[component][name]++
}

func (ms *MockState) GetMetric(component, name string) (float64, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if ms.metrics[component] == nil {
		return 0, false
	}
	val, exists := ms.metrics[component][name]
	return val, exists
}

func (ms *MockState) GetCounter(component, name string) (int, bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if ms.counters[component] == nil {
		return 0, false
	}
	val, exists := ms.counters[component][name]
	return val, exists
}

func (ms *MockState) GetAllMetrics() map[string]map[string]float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	// Create a deep copy to avoid race conditions
	result := make(map[string]map[string]float64)
	for component, metrics := range ms.metrics {
		result[component] = make(map[string]float64)
		for name, value := range metrics {
			result[component][name] = value
		}
	}
	return result
}

func TestNewSystemCollector(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	if collector == nil {
		t.Fatal("Expected collector to be created, got nil")
	}
	
	if collector.state != mockState {
		t.Error("Expected state to be set correctly")
	}
	
	if collector.interval != 1*time.Minute {
		t.Errorf("Expected default interval to be 1 minute, got %v", collector.interval)
	}
	
	if !collector.enabled {
		t.Error("Expected collector to be enabled by default")
	}
}

func TestNewSystemCollectorWithInterval(t *testing.T) {
	mockState := NewMockState()
	customInterval := 30 * time.Second
	collector := NewSystemCollectorWithInterval(mockState, customInterval)
	
	if collector.interval != customInterval {
		t.Errorf("Expected interval to be %v, got %v", customInterval, collector.interval)
	}
}

func TestCollectOnce(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	// Collect metrics once
	collector.CollectOnce()
	
	// Verify all expected system metrics were collected
	expectedMetrics := []string{
		"cpu_percent",
		"memory_bytes", 
		"health_data_size",
		"goroutines",
		"uptime_seconds",
	}
	
	for _, metricName := range expectedMetrics {
		if value, exists := mockState.GetMetric("system", metricName); !exists {
			t.Errorf("Expected metric 'system.%s' to be collected", metricName)
		} else {
			// Basic sanity checks for metric values
			switch metricName {
			case "cpu_percent":
				if value < 0 || value > 100 {
					t.Errorf("CPU percent should be between 0-100, got %f", value)
				}
			case "memory_bytes":
				if value <= 0 {
					t.Errorf("Memory bytes should be positive, got %f", value)
				}
			case "health_data_size":
				if value < 1024 { // minimum 1KB as per implementation
					t.Errorf("Health data size should be at least 1KB, got %f", value)
				}
			case "goroutines":
				if value < 1 {
					t.Errorf("Goroutines count should be at least 1, got %f", value)
				}
			case "uptime_seconds":
				if value < 0 {
					t.Errorf("Uptime should be non-negative, got %f", value)
				}
			}
		}
	}
}

func TestSetEnabled(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	// Test disabling
	collector.SetEnabled(false)
	if collector.IsEnabled() {
		t.Error("Expected collector to be disabled")
	}
	
	// Test that collection doesn't happen when disabled
	collector.CollectOnce()
	if len(mockState.GetAllMetrics()) > 0 {
		t.Error("Expected no metrics to be collected when disabled")
	}
	
	// Test re-enabling
	collector.SetEnabled(true)
	if !collector.IsEnabled() {
		t.Error("Expected collector to be enabled")
	}
	
	collector.CollectOnce()
	if len(mockState.GetAllMetrics()) == 0 {
		t.Error("Expected metrics to be collected when enabled")
	}
}

func TestSetInterval(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	newInterval := 5 * time.Second
	collector.SetInterval(newInterval)
	
	if collector.GetInterval() != newInterval {
		t.Errorf("Expected interval to be %v, got %v", newInterval, collector.GetInterval())
	}
}

func TestStartAndStop(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollectorWithInterval(mockState, 100*time.Millisecond)
	
	// Start collection
	collector.Start()
	
	// Wait for at least one collection cycle
	time.Sleep(200 * time.Millisecond)
	
	// Stop collection
	collector.Stop()
	
	// Verify metrics were collected
	metrics := mockState.GetAllMetrics()
	if len(metrics) == 0 {
		t.Error("Expected metrics to be collected during background collection")
	}
	
	if systemMetrics, exists := metrics["system"]; !exists {
		t.Error("Expected 'system' component to exist in collected metrics")
	} else {
		// Check that we have the expected system metrics
		expectedCount := 5 // cpu_percent, memory_bytes, health_data_size, goroutines, uptime_seconds
		if len(systemMetrics) != expectedCount {
			t.Errorf("Expected %d system metrics, got %d", expectedCount, len(systemMetrics))
		}
	}
}

func TestConcurrentCollection(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	// Run multiple concurrent collections
	var wg sync.WaitGroup
	numRoutines := 10
	
	wg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go func() {
			defer wg.Done()
			collector.CollectOnce()
		}()
	}
	
	wg.Wait()
	
	// Verify that metrics were collected (values may be overwritten due to concurrent access)
	metrics := mockState.GetAllMetrics()
	if len(metrics) == 0 {
		t.Error("Expected metrics to be collected even with concurrent access")
	}
}

func TestGetCPUPercent(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	cpuPercent := collector.getCPUPercent()
	
	// CPU percent should be within valid range
	if cpuPercent < 0 || cpuPercent > 100 {
		t.Errorf("CPU percent should be between 0-100, got %f", cpuPercent)
	}
}

func TestGetHealthDataSize(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	// Test with some mock memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	size := collector.getHealthDataSize(&memStats)
	
	// Size should be at least 1KB and not more than 100MB
	if size < 1024 {
		t.Errorf("Health data size should be at least 1KB, got %d", size)
	}
	if size > 100*1024*1024 {
		t.Errorf("Health data size should not exceed 100MB, got %d", size)
	}
}

func TestMetricValueRanges(t *testing.T) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	// Collect metrics multiple times to test consistency
	for i := 0; i < 3; i++ {
		collector.CollectOnce()
		time.Sleep(10 * time.Millisecond)
	}
	
	metrics := mockState.GetAllMetrics()
	systemMetrics := metrics["system"]
	
	// Test that uptime increases over time
	uptime := systemMetrics["uptime_seconds"]
	if uptime <= 0 {
		t.Errorf("Uptime should be positive, got %f", uptime)
	}
	
	// Test that goroutines count is reasonable
	goroutines := systemMetrics["goroutines"]
	if goroutines < 1 || goroutines > 10000 { // reasonable bounds for test environment
		t.Errorf("Goroutines count seems unreasonable: %f", goroutines)
	}
	
	// Test that memory usage is reasonable
	memory := systemMetrics["memory_bytes"]
	if memory < 1024 || memory > 1024*1024*1024 { // between 1KB and 1GB
		t.Errorf("Memory usage seems unreasonable: %f bytes", memory)
	}
}

// Benchmark tests
func BenchmarkCollectOnce(b *testing.B) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.CollectOnce()
	}
}

func BenchmarkGetCPUPercent(b *testing.B) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.getCPUPercent()
	}
}

func BenchmarkGetHealthDataSize(b *testing.B) {
	mockState := NewMockState()
	collector := NewSystemCollector(mockState)
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.getHealthDataSize(&memStats)
	}
}