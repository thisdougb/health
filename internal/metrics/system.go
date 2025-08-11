package metrics

import (
	"context"
	"runtime"
	"time"
	"unsafe"

	"github.com/thisdougb/health/internal/config"
)

// SystemCollector manages automatic collection and reporting of system metrics
type SystemCollector struct {
	state      StateInterface
	startTime  time.Time
	interval   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	enabled    bool
}

// StateInterface defines the methods we need from the state implementation
type StateInterface interface {
	AddMetric(component, name string, value float64)
	IncrComponentMetric(component, name string)
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector(state StateInterface) *SystemCollector {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Get sample rate from config (default 60 seconds)
	sampleRate := config.IntValue("HEALTH_SAMPLE_RATE")
	interval := time.Duration(sampleRate) * time.Second
	
	return &SystemCollector{
		state:     state,
		startTime: time.Now(),
		interval:  interval,
		ctx:       ctx,
		cancel:    cancel,
		enabled:   true,
	}
}

// NewSystemCollectorWithInterval creates a collector with custom interval
func NewSystemCollectorWithInterval(state StateInterface, interval time.Duration) *SystemCollector {
	collector := NewSystemCollector(state)
	collector.interval = interval
	return collector
}

// Start begins background collection of system metrics
func (sc *SystemCollector) Start() {
	if !sc.enabled {
		return
	}

	go sc.collectLoop()
}

// Stop gracefully stops the system metrics collection
func (sc *SystemCollector) Stop() {
	sc.cancel()
}

// SetEnabled enables or disables system metrics collection
func (sc *SystemCollector) SetEnabled(enabled bool) {
	sc.enabled = enabled
}

// collectLoop runs the periodic system metrics collection
func (sc *SystemCollector) collectLoop() {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	// Collect initial metrics immediately
	sc.collectSystemMetrics()

	for {
		select {
		case <-sc.ctx.Done():
			return
		case <-ticker.C:
			sc.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics gathers all system metrics and records them
func (sc *SystemCollector) collectSystemMetrics() {
	// CPU utilization percentage - simplified approach using runtime stats
	cpuPercent := sc.getCPUPercent()
	sc.state.AddMetric("system", "cpu_percent", cpuPercent)

	// Memory usage in bytes
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	sc.state.AddMetric("system", "memory_bytes", float64(memStats.Alloc))

	// Health data size - estimate based on metrics map memory usage
	healthDataSize := sc.getHealthDataSize(memStats)
	sc.state.AddMetric("system", "health_data_size", float64(healthDataSize))

	// Number of active goroutines
	goroutines := runtime.NumGoroutine()
	sc.state.AddMetric("system", "goroutines", float64(goroutines))

	// Application uptime in seconds
	uptime := time.Since(sc.startTime).Seconds()
	sc.state.AddMetric("system", "uptime_seconds", uptime)

	// Log collection for debugging (disabled)
	// log.Printf("System metrics collected at %v: CPU=%.1f%%, Mem=%dMB, Goroutines=%d, Uptime=%.0fs",
	//	timestamp.Format("15:04:05"),
	//	cpuPercent,
	//	memStats.Alloc/(1024*1024),
	//	goroutines,
	//	uptime)
}

// getCPUPercent provides a simplified CPU usage calculation
// Note: This is a basic implementation. For production use, consider more sophisticated CPU monitoring
func (sc *SystemCollector) getCPUPercent() float64 {
	// Get current runtime statistics
	var rtm1, rtm2 runtime.MemStats
	runtime.ReadMemStats(&rtm1)
	
	// Sample CPU-related metrics over a short interval
	time.Sleep(100 * time.Millisecond)
	
	runtime.ReadMemStats(&rtm2)
	
	// Calculate a rough CPU utilization based on GC activity and memory allocation
	// This is simplified - in production you might want to use system-specific APIs
	gcCycles := rtm2.NumGC - rtm1.NumGC
	allocDiff := rtm2.TotalAlloc - rtm1.TotalAlloc
	
	// Basic heuristic: more GC cycles and allocations indicate higher CPU usage
	cpuEstimate := float64(gcCycles)*10.0 + float64(allocDiff)/1024/1024*0.1
	
	// Cap at reasonable values (0-100%)
	if cpuEstimate > 100.0 {
		cpuEstimate = 100.0
	}
	if cpuEstimate < 0.0 {
		cpuEstimate = 0.0
	}
	
	return cpuEstimate
}

// getHealthDataSize estimates the memory usage of health metrics data
func (sc *SystemCollector) getHealthDataSize(memStats *runtime.MemStats) int64 {
	// Estimate based on heap objects and string allocation
	// This is an approximation - exact measurement would require instrumentation
	
	// Assume each metric entry takes roughly 100 bytes (name + value + overhead)
	estimatedEntries := memStats.HeapObjects / 10 // rough estimate
	estimatedSize := estimatedEntries * 100
	
	// Convert to int64 and ensure reasonable bounds
	size := int64(estimatedSize)
	if size < 1024 { // minimum 1KB
		size = 1024
	}
	if size > 100*1024*1024 { // cap at 100MB
		size = 100 * 1024 * 1024
	}
	
	return size
}

// GetInterval returns the current collection interval
func (sc *SystemCollector) GetInterval() time.Duration {
	return sc.interval
}

// SetInterval updates the collection interval (takes effect on next collection cycle)
func (sc *SystemCollector) SetInterval(interval time.Duration) {
	sc.interval = interval
}

// IsEnabled returns whether system metrics collection is enabled
func (sc *SystemCollector) IsEnabled() bool {
	return sc.enabled
}

// CollectOnce performs a single collection of system metrics (useful for testing)
func (sc *SystemCollector) CollectOnce() {
	if sc.enabled {
		sc.collectSystemMetrics()
	}
}

// sizeOf returns the approximate size of a value in bytes
func sizeOf(v interface{}) uintptr {
	return unsafe.Sizeof(v)
}