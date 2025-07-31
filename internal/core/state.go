package core

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/thisdougb/health/internal/metrics"
	"github.com/thisdougb/health/internal/storage"
)

// StateImpl holds our health data and persists all metric values.
// This is the internal implementation.
type StateImpl struct {
	Identity        string
	Started         int64
	Metrics         map[string]map[string]int
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

	s.mu.Lock() // enter CRITICAL SECTION
	if s.Metrics == nil {
		s.Metrics = make(map[string]map[string]int)
	}
	if s.Metrics[component] == nil {
		s.Metrics[component] = make(map[string]int)
	}

	s.Metrics[component][name]++
	currentValue := s.Metrics[component][name]
	s.mu.Unlock() // end CRITICAL SECTION

	// Persist metric asynchronously (non-blocking) - the storage backend handles queuing
	if err := s.persistence.PersistMetric(component, name, currentValue, "counter"); err != nil {
		log.Printf("Warning: Failed to persist counter metric %s.%s: %v", component, name, err)
	}
}

// AddMetric records a raw metric value for a specific component
func (s *StateImpl) AddMetric(component, name string, value float64) {
	if len(name) < 1 { // no name, no entry
		return
	}

	// Persist metric asynchronously (non-blocking) - the storage backend handles queuing
	if err := s.persistence.PersistMetric(component, name, value, "value"); err != nil {
		log.Printf("Warning: Failed to persist metric %s.%s: %v", component, name, err)
	}
}

// AddGlobalMetric records a raw metric value in the Global component
func (s *StateImpl) AddGlobalMetric(name string, value float64) {
	s.AddMetric("Global", name, value)
}

// Dump returns a JSON byte-string.
// We do not use a reader lock because it is probably unnecessary here,
// in this scenario.
func (s *StateImpl) Dump() string {
	var dataString string

	data, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		log.Fatalf("JSON Marshalling failed: %s", err)
	}
	dataString = string(data)

	return dataString
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
