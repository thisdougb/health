package health

import (
	"net/http"

	"github.com/thisdougb/health/internal/core"
	"github.com/thisdougb/health/internal/handlers"
	"github.com/thisdougb/health/internal/storage"
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

// NewStateWithPersistence creates a new health monitoring state instance with specified persistence
func NewStateWithPersistence(persistence *storage.Manager) *State {
	return &State{
		impl: core.NewStateWithPersistence(persistence),
	}
}

// SetConfig method sets the identity string for this metrics instance.
// The identity string will be in the Dump() output. A unique ID means we can find
// this node in a k8s cluster, for example.
func (s *State) SetConfig(identity string) {
	s.impl.Info(identity)
}

// IncrMetric increments a simple counter metric by one. Metrics start with a zero
// value, so the very first call to IncrMetric() always results in a value of 1.
func (s *State) IncrMetric(name string) {
	s.impl.IncrMetric(name)
}

// AddMetric records a raw metric value that gets persisted to storage
func (s *State) AddMetric(name string, value float64) {
	s.impl.AddGlobalMetric(name, value)
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

// HandleHealthRequest handles flexible health URL patterns directly
// This method allows integration with custom routing and middleware
// Supports: /health/, /health/status, /health/{component}, /health/{component}/status
func (s *State) HandleHealthRequest(w http.ResponseWriter, r *http.Request) {
	// For now, delegate to the basic health handler
	// This can be extended later to support component-specific routing
	handler := s.HealthHandler()
	handler(w, r)
}

// TimeSeriesHandler returns handler for sar-style time series queries
// Supports URL patterns: /health/{component}?window={duration}&lookback={duration}&date={date}&time={time}
//                     or: /health/{component}?window={duration}&lookahead={duration}&date={date}&time={time}
func (s *State) TimeSeriesHandler(component string) http.HandlerFunc {
	return handlers.TimeSeriesHandler(s.impl, component)
}

// Component-based metrics

// IncrComponentMetric increments a counter metric for a specific component
func (s *State) IncrComponentMetric(component, name string) {
	s.impl.IncrComponentMetric(component, name)
}

// AddComponentMetric records a raw metric value for a specific component
func (s *State) AddComponentMetric(component, name string, value float64) {
	s.impl.AddMetric(component, name, value)
}

// GetStorageManager returns the storage manager for administrative operations
// This enables access to backup, restore, and data extraction functions
func (s *State) GetStorageManager() *storage.Manager {
	return s.impl.GetStorageManager()
}

// Close gracefully shuts down the health state and flushes any pending data
func (s *State) Close() error {
	return s.impl.Close()
}
