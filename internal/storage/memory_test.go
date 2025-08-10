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

func TestWriteMetrics(t *testing.T) {
	backend := NewMemoryBackend()
	now := time.Now()

	tests := []struct {
		name     string
		metrics  []MetricEntry
		expected int
	}{
		{
			name:     "empty metrics",
			metrics:  []MetricEntry{},
			expected: 0,
		},
		{
			name: "single metric",
			metrics: []MetricEntry{
				{
					Timestamp: now,
					Component: "test",
					Name:      "counter",
					Value:     42,
					Type:      "counter",
				},
			},
			expected: 1,
		},
		{
			name: "multiple metrics",
			metrics: []MetricEntry{
				{
					Timestamp: now,
					Component: "web",
					Name:      "requests",
					Value:     100,
					Type:      "counter",
				},
				{
					Timestamp: now.Add(time.Second),
					Component: "db",
					Name:      "response_time",
					Value:     15.5,
					Type:      "rolling",
				},
			},
			expected: 3, // 1 from previous test + 2 new
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.WriteMetrics(tt.metrics)
			if err != nil {
				t.Errorf("WriteMetrics failed: %v", err)
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
	tsMetrics := []TimeSeriesEntry{
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

	err := backend.WriteTimeSeriesMetrics(tsMetrics)
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

			// Verify counter vs value metric types
			for _, result := range results {
				if result.Name == "hit_rate" {
					// Value metric should have stats map
					if _, ok := result.Value.(map[string]interface{}); !ok {
						t.Errorf("Expected value metric %s to return stats map", result.Name)
					}
				} else {
					// Counter metrics should have integer count
					if _, ok := result.Value.(int); !ok {
						t.Errorf("Expected counter metric %s to return integer count", result.Name)
					}
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

	// Add test data
	testMetrics := []MetricEntry{
		{Timestamp: now, Component: "web", Name: "requests", Value: 1, Type: "counter"},
		{Timestamp: now, Component: "db", Name: "queries", Value: 2, Type: "counter"},
		{Timestamp: now, Component: "web", Name: "errors", Value: 3, Type: "counter"},
		{Timestamp: now, Component: "cache", Name: "hits", Value: 4, Type: "counter"},
	}

	err = backend.WriteMetrics(testMetrics)
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

	// Add some data
	testMetrics := []MetricEntry{
		{Timestamp: now, Component: "test", Name: "metric", Value: 1, Type: "counter"},
	}
	err := backend.WriteMetrics(testMetrics)
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

	// Add some data
	testMetrics := []MetricEntry{
		{Timestamp: now, Component: "test", Name: "metric1", Value: 1, Type: "counter"},
		{Timestamp: now, Component: "test", Name: "metric2", Value: 2, Type: "counter"},
	}
	err := backend.WriteMetrics(testMetrics)
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
