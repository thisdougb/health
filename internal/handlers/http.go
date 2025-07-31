package handlers

import (
	"fmt"
	"net/http"
)

// StateInterface defines the interface that handlers need from the state
type StateInterface interface {
	Dump() string
}

// HealthHandler returns an HTTP handler that serves health metrics as JSON
// This provides a standard /health endpoint for containerized applications
func HealthHandler(state StateInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s\n", state.Dump())
	}
}

// MetricsHandler returns an HTTP handler that serves detailed metrics
// This is an alias for HealthHandler for semantic clarity
func MetricsHandler(state StateInterface) http.HandlerFunc {
	return HealthHandler(state)
}

// StatusHandler returns a simple UP/DOWN status endpoint
// Returns 200 OK for healthy, 503 Service Unavailable for unhealthy
func StatusHandler(state StateInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For now, always return healthy (200)
		// Future: implement actual health checks
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "UP\n")
	}
}