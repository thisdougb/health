package core

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/thisdougb/health/internal/metrics"
)

// StateImpl holds our health data, and calculates rolling average metrics
// whenever a new data point is added. This is the internal implementation.
type StateImpl struct {
	Identity           string
	Started            int64
	RollingDataSize    int
	Metrics            map[string]map[string]int
	rollingMetricsData map[string]*metrics.RollingMetric
	RollingMetrics     map[string]map[string]float64
	mu                 sync.Mutex // writer lock
}

// NewState creates a new state instance
func NewState() *StateImpl {
	return &StateImpl{}
}

// Info method sets the identity string for this metrics instance, and
// the sample size of for rolling average metrics. The identity string
// will be in the Dump() output. A unique ID means we can find
// this node in a k8s cluster, for example.
func (s *StateImpl) Info(identity string, rollingDataSize int) {
	defaultIdentity := "identity unset"
	defaultRollingDataSize := 10

	t := time.Now()
	s.Started = t.Unix()

	if len(identity) == 0 {
		s.Identity = defaultIdentity
	} else {
		s.Identity = identity
	}

	if rollingDataSize < 1 {
		s.RollingDataSize = defaultRollingDataSize
	} else {
		s.RollingDataSize = rollingDataSize
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
	s.mu.Unlock() // end CRITICAL SECTION
}

// UpdateRollingMetric adds data point for this metric, and re-calculates the
// rolling average metric value. Rolling averages are typical float types, so
// we expect a float64 type as the data point parameter.
// This method handles global rolling metrics.
func (s *StateImpl) UpdateRollingMetric(name string, value float64) {
	s.UpdateComponentRollingMetric("Global", name, value)
}

// UpdateComponentRollingMetric updates a rolling metric for a specific component
func (s *StateImpl) UpdateComponentRollingMetric(component, name string, value float64) {
	if len(name) < 1 { // no name, no entry
		return
	}

	metricKey := component + "_" + name
	s.mu.Lock() // enter CRITICAL SECTION

	// Initialize rolling metrics data if needed
	if s.rollingMetricsData == nil {
		s.rollingMetricsData = make(map[string]*metrics.RollingMetric)
	}

	_, ok := s.rollingMetricsData[metricKey]
	if !ok {
		s.rollingMetricsData[metricKey] = metrics.NewRollingMetric(s.RollingDataSize)
	}

	metric := s.rollingMetricsData[metricKey]
	newValue := metric.Add(value)

	// update RollingMetrics for json output
	if s.RollingMetrics == nil {
		s.RollingMetrics = make(map[string]map[string]float64)
	}
	if s.RollingMetrics[component] == nil {
		s.RollingMetrics[component] = make(map[string]float64)
	}
	s.RollingMetrics[component][name] = newValue
	s.mu.Unlock() // end CRITICAL SECTION
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
