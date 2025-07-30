package health

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/thisdougb/health/internal/metrics"
)

// State holds our health data, and calculates rolling average metrics
// whenever a new data point is added. It is exported to allow JSON
// access, and is not meant to be manipulated directly.
type State struct {
	Identity           string
	Started            int64
	RollingDataSize    int
	Metrics            map[string]int
	rollingMetricsData map[string]*metrics.RollingMetric
	RollingMetrics     map[string]float64
}

var mu sync.Mutex // writer lock

// Info method sets the identity string for this metrics instance, and
// the sample size of for rolling average metrics. The identity string
// will be in the Dump() output. A unique ID means we can find
// this node in a k8s cluster, for example.
func (s *State) Info(identity string, rollingDataSize int) {

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
func (s *State) IncrMetric(name string) {

	if len(name) < 1 { // no name, no entry
		return
	}

	mu.Lock() // enter CRITICAL SECTION
	if s.Metrics == nil {
		s.Metrics = make(map[string]int)
	}

	s.Metrics[name]++
	mu.Unlock() // end CRITICAL SECTION
}

// UpdateRollingMetric adds data point for this metric, and re-calculates the
// rolling average metric value. Rolling averages are typical float types, so
// we expect a float64 type as the data point parameter.
func (s *State) UpdateRollingMetric(name string, value float64) {

	if len(name) < 1 { // no name, no entry
		return
	}

	mu.Lock() // enter CRITICAL SECTION
	_, ok := s.RollingMetrics[name]
	if !ok {
		if s.RollingMetrics == nil {
			s.rollingMetricsData = make(map[string]*metrics.RollingMetric)
		}
		s.rollingMetricsData[name] = metrics.NewRollingMetric(s.RollingDataSize)
	}

	metric := s.rollingMetricsData[name]
	newValue := metric.Add(value)

	// update RollingMetrics for nicer json output
	if s.RollingMetrics == nil {
		s.RollingMetrics = make(map[string]float64)
	}
	s.RollingMetrics[name] = newValue
	mu.Unlock() // end CRITICAL SECTION
}

// Dump returns a JSON byte-string.
// We do not use a reader lock because it is probably unnecessary here,
// in this scenario.
func (s *State) Dump() string {

	var dataString string

	data, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		log.Fatalf("JSON Marshalling failed: %s", err)
	}
	dataString = string(data)

	return dataString
}
