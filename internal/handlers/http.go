package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thisdougb/health/internal/storage"
)

// StateInterface defines the interface that handlers need from the state
type StateInterface interface {
	Dump() string
	GetStorageManager() *storage.Manager
}

// TimeSeriesParams holds parsed query parameters
type TimeSeriesParams struct {
	Window    time.Duration
	Lookback  *time.Duration
	Lookahead *time.Duration
	Date      *time.Time
	Time      *time.Time
}

// RequestParams represents the original query parameters from the request
type RequestParams struct {
	Window    string `json:"window,omitempty"`
	Lookback  string `json:"lookback,omitempty"`
	Lookahead string `json:"lookahead,omitempty"`
	Date      string `json:"date,omitempty"`
	Time      string `json:"time,omitempty"`
}

// TimeSeriesResponse represents aggregated time series data
type TimeSeriesResponse struct {
	Component     string                 `json:"component"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	ReferenceTime time.Time              `json:"reference_time"`
	RequestParams RequestParams          `json:"request_params"`
	Metrics       map[string]interface{} `json:"metrics"`
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

// TimeSeriesHandler returns handler for sar-style time series queries
// Supports URL patterns: /health/{component}?window={duration}&lookback={duration}&date={date}&time={time}
//                     or: /health/{component}?window={duration}&lookahead={duration}&date={date}&time={time}
func TimeSeriesHandler(state StateInterface, component string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		params, err := parseTimeSeriesParams(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid parameters: %v", err), http.StatusBadRequest)
			return
		}

		// Validate mutually exclusive lookback/lookahead
		if params.Lookback != nil && params.Lookahead != nil {
			http.Error(w, "lookback and lookahead are mutually exclusive", http.StatusBadRequest)
			return
		}

		if params.Lookback == nil && params.Lookahead == nil {
			http.Error(w, "either lookback or lookahead must be specified", http.StatusBadRequest)
			return
		}

		// Check if persistence is enabled
		manager := state.GetStorageManager()
		if manager == nil || !manager.IsEnabled() {
			http.Error(w, "time series queries require persistence to be enabled", http.StatusServiceUnavailable)
			return
		}

		// Calculate reference time
		referenceTime := calculateReferenceTime(params)

		// Calculate time range
		var startTime, endTime time.Time

		if params.Lookback != nil {
			startTime = referenceTime.Add(-*params.Lookback)
			endTime = referenceTime
		} else {
			startTime = referenceTime
			endTime = referenceTime.Add(*params.Lookahead)
		}

		// Read metrics from storage
		metrics, err := manager.ReadMetrics(component, startTime, endTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read metrics: %v", err), http.StatusInternalServerError)
			return
		}

		// Aggregate metrics by window
		aggregatedMetrics := aggregateMetricsByWindow(metrics, params.Window)

		// Build request params from original query parameters  
		requestParams := RequestParams{
			Window: r.URL.Query().Get("window"),
		}
		if params.Lookback != nil {
			requestParams.Lookback = r.URL.Query().Get("lookback")
		}
		if params.Lookahead != nil {
			requestParams.Lookahead = r.URL.Query().Get("lookahead")
		}
		if dateStr := r.URL.Query().Get("date"); dateStr != "" {
			requestParams.Date = dateStr
		}
		if timeStr := r.URL.Query().Get("time"); timeStr != "" {
			requestParams.Time = timeStr
		}

		// Build response
		response := TimeSeriesResponse{
			Component:     component,
			StartTime:     startTime,
			EndTime:       endTime,
			ReferenceTime: referenceTime,
			RequestParams: requestParams,
			Metrics:       aggregatedMetrics,
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// parseTimeSeriesParams parses query parameters for time series requests
func parseTimeSeriesParams(r *http.Request) (*TimeSeriesParams, error) {
	params := &TimeSeriesParams{}

	// Parse window parameter (required)
	windowStr := r.URL.Query().Get("window")
	if windowStr == "" {
		return nil, fmt.Errorf("window parameter is required")
	}
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("invalid window duration: %v", err)
	}
	params.Window = window

	// Parse lookback parameter (optional)
	lookbackStr := r.URL.Query().Get("lookback")
	if lookbackStr != "" {
		lookback, err := time.ParseDuration(lookbackStr)
		if err != nil {
			return nil, fmt.Errorf("invalid lookback duration: %v", err)
		}
		params.Lookback = &lookback
	}

	// Parse lookahead parameter (optional)
	lookaheadStr := r.URL.Query().Get("lookahead")
	if lookaheadStr != "" {
		lookahead, err := time.ParseDuration(lookaheadStr)
		if err != nil {
			return nil, fmt.Errorf("invalid lookahead duration: %v", err)
		}
		params.Lookahead = &lookahead
	}

	// Parse date parameter (optional, defaults to today)
	dateStr := r.URL.Query().Get("date")
	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD: %v", err)
		}
		params.Date = &date
	}

	// Parse time parameter (optional, defaults to current time)
	timeStr := r.URL.Query().Get("time")
	if timeStr != "" {
		// Parse time in HH:MM:SS format
		timeParts := strings.Split(timeStr, ":")
		if len(timeParts) < 2 || len(timeParts) > 3 {
			return nil, fmt.Errorf("invalid time format, use HH:MM:SS or HH:MM")
		}

		hour, err := strconv.Atoi(timeParts[0])
		if err != nil || hour < 0 || hour > 23 {
			return nil, fmt.Errorf("invalid hour: %s", timeParts[0])
		}

		minute, err := strconv.Atoi(timeParts[1])
		if err != nil || minute < 0 || minute > 59 {
			return nil, fmt.Errorf("invalid minute: %s", timeParts[1])
		}

		second := 0
		if len(timeParts) == 3 {
			second, err = strconv.Atoi(timeParts[2])
			if err != nil || second < 0 || second > 59 {
				return nil, fmt.Errorf("invalid second: %s", timeParts[2])
			}
		}

		// Create time on a fixed date (will be combined with date later)
		parsedTime := time.Date(2000, 1, 1, hour, minute, second, 0, time.UTC)
		params.Time = &parsedTime
	}

	return params, nil
}

// calculateReferenceTime combines date and time parameters to create reference time
func calculateReferenceTime(params *TimeSeriesParams) time.Time {
	now := time.Now()

	// Start with today's date
	referenceDate := now
	if params.Date != nil {
		referenceDate = *params.Date
	}

	// Use current time if not specified
	referenceTime := now
	if params.Time != nil {
		referenceTime = *params.Time
	}

	// Combine date and time
	return time.Date(
		referenceDate.Year(), referenceDate.Month(), referenceDate.Day(),
		referenceTime.Hour(), referenceTime.Minute(), referenceTime.Second(),
		0, time.UTC,
	)
}

// aggregateMetricsByWindow aggregates metrics into time windows for sar-style output
func aggregateMetricsByWindow(metrics []storage.MetricEntry, window time.Duration) map[string]interface{} {
	if len(metrics) == 0 {
		return make(map[string]interface{})
	}

	// Group metrics by name and time window
	windowGroups := make(map[string]map[int64][]float64)

	for _, metric := range metrics {
		// Calculate window bucket (unix timestamp divided by window seconds)
		windowStart := metric.Timestamp.Truncate(window).Unix()

		if windowGroups[metric.Name] == nil {
			windowGroups[metric.Name] = make(map[int64][]float64)
		}

		// Convert value to float64 for aggregation
		var val float64
		switch v := metric.Value.(type) {
		case float64:
			val = v
		case float32:
			val = float64(v)
		case int:
			val = float64(v)
		case int64:
			val = float64(v)
		default:
			continue // Skip unsupported types
		}

		windowGroups[metric.Name][windowStart] = append(windowGroups[metric.Name][windowStart], val)
	}

	// Calculate averages for each window
	result := make(map[string]interface{})
	for metricName, windows := range windowGroups {
		windowAverages := make(map[string]float64)

		for windowStart, values := range windows {
			// Calculate average for this window
			sum := 0.0
			for _, val := range values {
				sum += val
			}
			avg := sum / float64(len(values))

			// Convert timestamp back to readable format
			windowTime := time.Unix(windowStart, 0).UTC().Format("15:04:05")
			windowAverages[windowTime] = avg
		}

		result[metricName] = windowAverages
	}

	return result
}
