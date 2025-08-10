package core

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thisdougb/health/internal/config"
	"github.com/thisdougb/health/internal/metrics"
	"github.com/thisdougb/health/internal/storage"
)

// getCurrentTimeKey returns a string key representing the current time window
// Uses HEALTH_SAMPLE_RATE config value for window duration (default 60 seconds)
func getCurrentTimeKey() string {
	now := time.Now()
	windowSeconds := config.IntValue("HEALTH_SAMPLE_RATE")
	windowDuration := time.Duration(windowSeconds) * time.Second
	
	// Truncate to the configured time window boundary
	truncated := now.Truncate(windowDuration)
	return fmt.Sprintf("%d", truncated.Unix())
}

// StateImpl holds our health data and persists all metric values.
// This is the internal implementation.
type StateImpl struct {
	Identity        string
	Started         int64
	SampledMetrics  map[string]map[string]map[string][]float64 // component -> timekey -> metric -> values
	persistence     *storage.Manager
	systemCollector *metrics.SystemCollector
	mu              sync.Mutex // writer lock
}

// NewState creates a new state instance
func NewState() *StateImpl {
	// Try to initialize persistence from environment config
	persistence, err := storage.NewManagerFromConfig()
	if err != nil {
		// Log error but continue without persistence
		log.Printf("Warning: Failed to initialize persistence: %v", err)
		persistence = storage.NewManager(nil, false)
	}

	state := &StateImpl{
		persistence: persistence,
	}

	// Initialize and start system metrics collector
	state.systemCollector = metrics.NewSystemCollector(state)
	state.systemCollector.Start()
	
	return state
}

// NewStateWithPersistence creates a new state instance with specified persistence manager
func NewStateWithPersistence(persistence *storage.Manager) *StateImpl {
	state := &StateImpl{
		persistence: persistence,
	}

	// Initialize and start system metrics collector
	state.systemCollector = metrics.NewSystemCollector(state)
	state.systemCollector.Start()
	
	return state
}

// Info method sets the identity string for this metrics instance.
// The identity string will be in the Dump() output. A unique ID means we can find
// this node in a k8s cluster, for example.
func (s *StateImpl) Info(identity string) {
	defaultIdentity := "identity unset"

	t := time.Now()
	s.Started = t.Unix()

	if len(identity) == 0 {
		s.Identity = defaultIdentity
	} else {
		s.Identity = identity
	}
}

// IncrMetric increments a simple counter metric by one. Metrics start with a zero
// value, so the very first call to IncrMetric() always results in a value of 1.
// This method handles global metrics.
func (s *StateImpl) IncrMetric(name string) {
	s.IncrComponentMetric("Global", name)
}

// IncrComponentMetric increments a counter metric for a specific component
func (s *StateImpl) IncrComponentMetric(component, name string) {
	if len(name) < 1 { // no name, no entry
		return
	}

	timeKey := getCurrentTimeKey()

	s.mu.Lock() // enter CRITICAL SECTION
	if s.SampledMetrics == nil {
		s.SampledMetrics = make(map[string]map[string]map[string][]float64)
	}
	if s.SampledMetrics[component] == nil {
		s.SampledMetrics[component] = make(map[string]map[string][]float64)
	}
	if s.SampledMetrics[component][timeKey] == nil {
		s.SampledMetrics[component][timeKey] = make(map[string][]float64)
	}

	// Append 1.0 for each counter increment
	s.SampledMetrics[component][timeKey][name] = append(s.SampledMetrics[component][timeKey][name], 1.0)
	s.mu.Unlock() // end CRITICAL SECTION
}

// AddMetric records a raw metric value for a specific component
func (s *StateImpl) AddMetric(component, name string, value float64) {
	if len(name) < 1 { // no name, no entry
		return
	}

	timeKey := getCurrentTimeKey()

	s.mu.Lock() // enter CRITICAL SECTION
	if s.SampledMetrics == nil {
		s.SampledMetrics = make(map[string]map[string]map[string][]float64)
	}
	if s.SampledMetrics[component] == nil {
		s.SampledMetrics[component] = make(map[string]map[string][]float64)
	}
	if s.SampledMetrics[component][timeKey] == nil {
		s.SampledMetrics[component][timeKey] = make(map[string][]float64)
	}

	// Append the actual value for raw metrics
	s.SampledMetrics[component][timeKey][name] = append(s.SampledMetrics[component][timeKey][name], value)
	s.mu.Unlock() // end CRITICAL SECTION
}

// AddGlobalMetric records a raw metric value in the Global component
func (s *StateImpl) AddGlobalMetric(name string, value float64) {
	s.AddMetric("Global", name, value)
}

// Dump returns a JSON byte-string showing current time window statistics.
// Uses mutex protection to prevent race conditions during concurrent access.
// This ensures thread safety when reading metrics while writes are happening.
func (s *StateImpl) Dump() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a structure for JSON output with current window statistics
	currentTimeKey := getCurrentTimeKey()
	output := map[string]interface{}{
		"Identity": s.Identity,
		"Started":  s.Started,
		"Metrics":  make(map[string]map[string]interface{}),
	}

	// Convert current time window data to aggregated statistics
	if s.SampledMetrics != nil {
		for component, timeWindows := range s.SampledMetrics {
			if timeData, exists := timeWindows[currentTimeKey]; exists && len(timeData) > 0 {
				componentMetrics := make(map[string]interface{})
				for metricName, values := range timeData {
					if len(values) > 0 {
						// For counter metrics (all 1.0 values), show count
						// For value metrics, show current statistics
						if allOnes(values) {
							componentMetrics[metricName] = len(values)
						} else {
							componentMetrics[metricName] = map[string]interface{}{
								"count": len(values),
								"min":   minValue(values),
								"max":   maxValue(values),
								"avg":   avgValue(values),
							}
						}
					}
				}
				if len(componentMetrics) > 0 {
					output["Metrics"].(map[string]map[string]interface{})[component] = componentMetrics
				}
			}
		}
	}

	data, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		log.Fatalf("JSON Marshalling failed: %s", err)
	}

	return string(data)
}

// Helper functions for statistics calculation
func allOnes(values []float64) bool {
	for _, v := range values {
		if v != 1.0 {
			return false
		}
	}
	return true
}

func minValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func avgValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// GetStorageManager returns the storage manager for administrative operations
func (s *StateImpl) GetStorageManager() *storage.Manager {
	return s.persistence
}

// Close gracefully shuts down the state instance and flushes any pending data
func (s *StateImpl) Close() error {
	// Stop system metrics collection
	if s.systemCollector != nil {
		s.systemCollector.Stop()
	}
	
	// Close persistence manager
	if s.persistence != nil {
		return s.persistence.Close()
	}
	return nil
}
