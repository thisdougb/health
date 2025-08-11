package storage

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestNewMemoryBackend(t *testing.T) {
	backend := NewMemoryBackend()
	if backend == nil {
		t.Fatal("NewMemoryBackend returned nil")
	}
	if backend.Len() != 0 {
		t.Errorf("Expected empty backend, got %d metrics", backend.Len())
	}
}

func TestWriteMetrics_ShouldFail(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now()

	// Memory backend should reject raw metrics and require processed data
	metrics := []MetricEntry{
		{
			Timestamp: now,
			Component: "test",
			Name:      "counter",
			Value:     42,
			Type:      "value",
		},
	}

	err := backend.WriteMetrics(metrics)
	if err == nil {
		t.Error("Expected WriteMetrics to fail for memory backend, but it succeeded")
	}
	if backend.Len() != 0 {
		t.Errorf("Expected 0 metrics after failed write, got %d", backend.Len())
	}
}

func TestWriteMetricsData(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now()

	tests := []struct {
		name     string
		metrics  []MetricsDataEntry
		expected int
	}{
		{
			name:     "empty metrics",
			metrics:  []MetricsDataEntry{},
			expected: 0,
		},
		{
			name: "single metric",
			metrics: []MetricsDataEntry{
				{
					TimeWindowKey: now.Format("20060102150400"),
					Component:     "test",
					Metric:        "requests",
					MinValue:      1.0,
					MaxValue:      1.0,
					AvgValue:      1.0,
					Count:         42,
				},
			},
			expected: 1,
		},
		{
			name: "multiple metrics",
			metrics: []MetricsDataEntry{
				{
					TimeWindowKey: now.Format("20060102150400"),
					Component:     "web",
					Metric:        "requests",
					MinValue:      100.0,
					MaxValue:      100.0,
					AvgValue:      100.0,
					Count:         1,
				},
				{
					TimeWindowKey: now.Add(time.Minute).Format("20060102150400"),
					Component:     "db",
					Metric:        "response_time",
					MinValue:      15.5,
					MaxValue:      15.5,
					AvgValue:      15.5,
					Count:         1,
				},
			},
			expected: 3, // 1 from previous test + 2 new
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.WriteMetricsData(tt.metrics)
			if err != nil {
				t.Errorf("WriteMetricsData failed: %v", err)
			}
			if backend.Len() != tt.expected {
				t.Errorf("Expected %d metrics, got %d", tt.expected, backend.Len())
			}
		})
	}
}

func TestReadMetrics(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now().Truncate(time.Minute)

	// Setup time series test data
	tsMetrics := []MetricsDataEntry{
		{
			TimeWindowKey: now.Add(-2 * time.Hour).Format("20060102150400"),
			Component:     "web",
			Metric:        "requests",
			MinValue:      1.0,  // Counter metric
			MaxValue:      1.0,
			AvgValue:      1.0,
			Count:         50,   // 50 requests in this window
		},
		{
			TimeWindowKey: now.Add(-1 * time.Hour).Format("20060102150400"), 
			Component:     "web",
			Metric:        "requests",
			MinValue:      1.0,  // Counter metric
			MaxValue:      1.0,
			AvgValue:      1.0,
			Count:         75,   // 75 requests in this window
		},
		{
			TimeWindowKey: now.Format("20060102150400"),
			Component:     "db",
			Metric:        "queries",
			MinValue:      1.0,  // Counter metric
			MaxValue:      1.0,
			AvgValue:      1.0,
			Count:         25,   // 25 queries in this window
		},
		{
			TimeWindowKey: now.Add(time.Hour).Format("20060102150400"),
			Component:     "cache",
			Metric:        "hit_rate",
			MinValue:      0.80, // Value metric with stats
			MaxValue:      0.90,
			AvgValue:      0.85,
			Count:         100,  // 100 samples in this window
		},
	}

	err := backend.WriteMetricsData(tsMetrics)
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	tests := []struct {
		name      string
		component string
		start     time.Time
		end       time.Time
		expected  int
	}{
		{
			name:      "all components, all time",
			component: "",
			start:     now.Add(-3 * time.Hour),
			end:       now.Add(2 * time.Hour),
			expected:  4,
		},
		{
			name:      "web component only", 
			component: "web",
			start:     now.Add(-3 * time.Hour),
			end:       now.Add(2 * time.Hour),
			expected:  2,
		},
		{
			name:      "time range filter",
			component: "",
			start:     now.Add(-90 * time.Minute),
			end:       now.Add(30 * time.Minute),
			expected:  2, // db queries at now, web requests at -1hr
		},
		{
			name:      "no matches",
			component: "nonexistent",
			start:     now.Add(-3 * time.Hour),
			end:       now.Add(2 * time.Hour),
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := backend.ReadMetrics(tt.component, tt.start, tt.end)
			if err != nil {
				t.Errorf("ReadMetrics failed: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}

			// Verify results are sorted by timestamp
			for i := 1; i < len(results); i++ {
				if results[i].Timestamp.Before(results[i-1].Timestamp) {
					t.Error("Results not sorted by timestamp")
					break
				}
			}

			// Verify all metrics return aggregated data (all are now value metrics)
			for _, result := range results {
				// All metrics should have aggregated stats map
				if statsMap, ok := result.Value.(map[string]interface{}); !ok {
					t.Errorf("Expected metric %s to return stats map, got %T", result.Name, result.Value)
				} else {
					// Verify required fields exist
					if _, hasAvg := statsMap["avg"]; !hasAvg {
						t.Errorf("Expected metric %s stats to have 'avg' field", result.Name)
					}
					if _, hasMin := statsMap["min"]; !hasMin {
						t.Errorf("Expected metric %s stats to have 'min' field", result.Name)
					}
					if _, hasMax := statsMap["max"]; !hasMax {
						t.Errorf("Expected metric %s stats to have 'max' field", result.Name)
					}
					if _, hasCount := statsMap["count"]; !hasCount {
						t.Errorf("Expected metric %s stats to have 'count' field", result.Name)
					}
				}
				
				// All metrics should have type "value" now
				if result.Type != "value" {
					t.Errorf("Expected metric %s to have type 'value', got '%s'", result.Name, result.Type)
				}
			}
		})
	}
}

func TestListComponents(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now()

	// Test empty backend
	components, err := backend.ListComponents()
	if err != nil {
		t.Errorf("ListComponents failed: %v", err)
	}
	if len(components) != 0 {
		t.Errorf("Expected no components, got %v", components)
	}

	// Add test data using processed metrics format
	testMetrics := []MetricsDataEntry{
		{TimeWindowKey: now.Format("20060102150400"), Component: "web", Metric: "requests", MinValue: 1, MaxValue: 1, AvgValue: 1, Count: 1},
		{TimeWindowKey: now.Format("20060102150400"), Component: "db", Metric: "queries", MinValue: 2, MaxValue: 2, AvgValue: 2, Count: 1},
		{TimeWindowKey: now.Format("20060102150400"), Component: "web", Metric: "errors", MinValue: 3, MaxValue: 3, AvgValue: 3, Count: 1},
		{TimeWindowKey: now.Format("20060102150400"), Component: "cache", Metric: "hits", MinValue: 4, MaxValue: 4, AvgValue: 4, Count: 1},
	}

	err = backend.WriteMetricsData(testMetrics)
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	components, err = backend.ListComponents()
	if err != nil {
		t.Errorf("ListComponents failed: %v", err)
	}

	expected := []string{"cache", "db", "web"}
	sort.Strings(components)

	if !reflect.DeepEqual(components, expected) {
		t.Errorf("Expected components %v, got %v", expected, components)
	}
}

func TestClose(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now()

	// Add some data using processed metrics format
	testMetrics := []MetricsDataEntry{
		{TimeWindowKey: now.Format("20060102150400"), Component: "test", Metric: "metric", MinValue: 1, MaxValue: 1, AvgValue: 1, Count: 1},
	}
	err := backend.WriteMetricsData(testMetrics)
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	if backend.Len() != 1 {
		t.Errorf("Expected 1 metric before close, got %d", backend.Len())
	}

	err = backend.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if backend.Len() != 0 {
		t.Errorf("Expected 0 metrics after close, got %d", backend.Len())
	}
}

func TestClear(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now()

	// Add some data using processed metrics format
	testMetrics := []MetricsDataEntry{
		{TimeWindowKey: now.Format("20060102150400"), Component: "test", Metric: "metric1", MinValue: 1, MaxValue: 1, AvgValue: 1, Count: 1},
		{TimeWindowKey: now.Format("20060102150400"), Component: "test", Metric: "metric2", MinValue: 2, MaxValue: 2, AvgValue: 2, Count: 1},
	}
	err := backend.WriteMetricsData(testMetrics)
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	if backend.Len() != 2 {
		t.Errorf("Expected 2 metrics before clear, got %d", backend.Len())
	}

	backend.Clear()

	if backend.Len() != 0 {
		t.Errorf("Expected 0 metrics after clear, got %d", backend.Len())
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ Backend = (*MemoryBackend)(nil)
}
