package health

import (
	"net/http"

	"github.com/thisdougb/health/internal/core"
	"github.com/thisdougb/health/internal/handlers"
)

// State is the public interface for the health monitoring system
type State struct {
	impl *core.StateImpl
}

// NewState creates a new health monitoring state instance
func NewState() *State {
	return &State{
		impl: core.NewState(),
	}
}

// SetConfig method sets the identity string for this metrics instance, and
// the sample size of for rolling average metrics. The identity string
// will be in the Dump() output. A unique ID means we can find
// this node in a k8s cluster, for example.
func (s *State) SetConfig(identity string, rollingDataSize int) {
	s.impl.Info(identity, rollingDataSize)
}

// IncrMetric increments a simple counter metric by one. Metrics start with a zero
// value, so the very first call to IncrMetric() always results in a value of 1.
func (s *State) IncrMetric(name string) {
	s.impl.IncrMetric(name)
}

// UpdateRollingMetric adds data point for this metric, and re-calculates the
// rolling average metric value. Rolling averages are typical float types, so
// we expect a float64 type as the data point parameter.
func (s *State) UpdateRollingMetric(name string, value float64) {
	s.impl.UpdateRollingMetric(name, value)
}

// Dump returns a JSON byte-string.
func (s *State) Dump() string {
	return s.impl.Dump()
}

// HealthHandler returns an HTTP handler that serves health metrics as JSON
// This provides a standard /health endpoint for containerized applications
func (s *State) HealthHandler() http.HandlerFunc {
	return handlers.HealthHandler(s.impl)
}

// MetricsHandler returns an HTTP handler that serves detailed metrics
// This is an alias for HealthHandler for semantic clarity
func (s *State) MetricsHandler() http.HandlerFunc {
	return s.HealthHandler()
}

// StatusHandler returns a simple UP/DOWN status endpoint
// Returns 200 OK for healthy, 503 Service Unavailable for unhealthy
func (s *State) StatusHandler() http.HandlerFunc {
	return handlers.StatusHandler(s.impl)
}

// Component-based metrics

// IncrComponentMetric increments a counter metric for a specific component
func (s *State) IncrComponentMetric(component, name string) {
	s.impl.IncrComponentMetric(component, name)
}

// UpdateComponentRollingMetric updates a rolling metric for a specific component
func (s *State) UpdateComponentRollingMetric(component, name string, value float64) {
	s.impl.UpdateComponentRollingMetric(component, name, value)
}
