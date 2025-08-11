package health

import (
	"net/http"

	"github.com/thisdougb/health/internal/core"
	"github.com/thisdougb/health/internal/handlers"
	"github.com/thisdougb/health/internal/storage"
)

// State is the primary interface for the time-windowed health monitoring system.
// It provides methods for collecting metrics in time windows, HTTP handlers for
// health endpoints, and access to time-series data for analysis.
type State struct {
	impl *core.StateImpl
}

// NewState creates a new time-windowed health monitoring state instance.
// Uses environment variables for configuration (HEALTH_PERSISTENCE_ENABLED, etc.).
// Returns a State ready for metric collection and HTTP serving.
func NewState() *State {
	return &State{
		impl: core.NewState(),
	}
}

// NewStateWithPersistence creates a health monitoring state instance with custom persistence.
// Allows explicit control over storage backend instead of environment variable configuration.
// Used primarily for testing and advanced deployment scenarios.
func NewStateWithPersistence(persistence *storage.Manager) *State {
	return &State{
		impl: core.NewStateWithPersistence(persistence),
	}
}

// SetConfig sets the identity string for this metrics instance.
// The identity appears in JSON output and HTTP responses to distinguish
// different application instances (useful in clusters and distributed systems).
func (s *State) SetConfig(identity string) {
	s.impl.Info(identity)
}

// IncrMetric increments a global counter metric by one in the current time window.
// Counter metrics are stored in memory and included in JSON output.
// Thread-safe with sub-microsecond performance for high-frequency collection.
func (s *State) IncrMetric(name string) {
	s.impl.IncrMetric(name)
}

// AddMetric records a raw global metric value for time-series persistence.
// Raw values are queued for storage with statistical aggregation (min/max/avg/count).
// Thread-safe with microsecond performance, not included in JSON counter output.
func (s *State) AddMetric(name string, value float64) {
	s.impl.AddGlobalMetric(name, value)
}

// Dump returns current counter metrics as JSON string.
// Includes identity, start time, and counter metrics organized by component.
// Raw metric values are excluded (stored separately for time-series analysis).
func (s *State) Dump() string {
	return s.impl.Dump()
}

// HealthHandler returns an HTTP handler that serves counter metrics as JSON.
// Provides standard /health endpoint compatible with containerized health checks.
// Returns 200 OK with JSON payload containing current counter values.
func (s *State) HealthHandler() http.HandlerFunc {
	return handlers.HealthHandler(s.impl)
}

// MetricsHandler returns an HTTP handler that serves counter metrics as JSON.
// Alias for HealthHandler providing semantic clarity in routing configurations.
// Both methods return identical JSON counter metric data.
func (s *State) MetricsHandler() http.HandlerFunc {
	return s.HealthHandler()
}

// StatusHandler returns a simple UP/DOWN health status endpoint.
// Returns 200 OK with "OK" body for healthy state.
// Used for basic health checks that don't need detailed metrics.
func (s *State) StatusHandler() http.HandlerFunc {
	return handlers.StatusHandler(s.impl)
}

// HandleHealthRequest handles flexible health URL patterns with storage-backed queries.
// Supports time-series parameters and optional component routing from URL path.
// URL patterns: /health, /health/{component}, /health/{component}/timeseries
// Query params: interval, lookback/lookahead, date, time (all optional)
// Default when no time params: interval=1m, lookback=1h (past hour in 1-minute intervals)
func (s *State) HandleHealthRequest(w http.ResponseWriter, r *http.Request) {
	handlers.HandleHealthRequestUnified(s.impl, w, r)
}

// TimeSeriesHandler returns an HTTP handler for sar-style time series queries.
// Provides statistical aggregation with configurable time windows and date ranges.
// Query parameters: window (5m,1h), lookback/lookahead (2h,1d), date/time (optional).
// Returns JSON with min/max/avg/count statistics for specified component.
func (s *State) TimeSeriesHandler(component string) http.HandlerFunc {
	return handlers.TimeSeriesHandler(s.impl, component)
}

// IncrComponentMetric increments a counter metric for a specific component.
// Counter metrics are organized by component in JSON output for easy filtering.
// Thread-safe with sub-microsecond performance, included in health endpoint responses.
func (s *State) IncrComponentMetric(component, name string) {
	s.impl.IncrComponentMetric(component, name)
}

// AddComponentMetric records a raw metric value for a specific component.
// Raw values are queued for time-series storage with statistical aggregation.
// Thread-safe with microsecond performance, available via TimeSeriesHandler queries.
func (s *State) AddComponentMetric(component, name string, value float64) {
	s.impl.AddMetric(component, name, value)
}

// GetStorageManager returns the storage manager for administrative operations.
// Provides access to backup creation, data extraction, and retention management.
// Used for operational tasks and administrative data analysis.
func (s *State) GetStorageManager() *storage.Manager {
	return s.impl.GetStorageManager()
}

// Close gracefully shuts down the health monitoring system.
// Flushes pending raw metrics to storage and triggers backup creation if enabled.
// Should be called during application shutdown to prevent data loss.
func (s *State) Close() error {
	return s.impl.Close()
}
