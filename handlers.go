package health

import (
	"fmt"
	"net/http"
)

// HealthHandler returns an HTTP handler that serves health metrics as JSON
// This provides a standard /health endpoint for containerized applications
func (s *State) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s\n", s.Dump())
	}
}

// MetricsHandler returns an HTTP handler that serves detailed metrics
// This is an alias for HealthHandler for semantic clarity
func (s *State) MetricsHandler() http.HandlerFunc {
	return s.HealthHandler()
}

// StatusHandler returns a simple UP/DOWN status endpoint
// Returns 200 OK for healthy, 503 Service Unavailable for unhealthy
func (s *State) StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For now, always return healthy (200)
		// Future: implement actual health checks
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "UP\n")
	}
}