package health

import (
	"strconv"
	"strings"
	"testing"
)

func TestInfoMethodSetters(t *testing.T) {
	// Test setting the identity and rolling data size.
	//
	identity := "workerXYZ"
	rDataSize := 5

	s := NewState()
	s.SetConfig(identity, rDataSize)
	result := s.Dump()

	searchFor := "\"Identity\": \"" + identity + "\","
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Info failed to set Identity")
	}

	searchFor = "\"RollingDataSize\": " + strconv.Itoa(rDataSize) + ","
	searchResult = strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Info failed to set RollingDataPoints")
	}
}

func TestInfoMethodSetterDefaults(t *testing.T) {
	// Test setting the identity and rolling data size uses defaults
	// when no values are supplied.
	identity := ""
	rDataSize := 0

	s := NewState()
	s.SetConfig(identity, rDataSize)
	result := s.Dump()

	searchFor := "\"Identity\": \"identity unset\","
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Info failed to set default Identity")
	}

	searchFor = "\"RollingDataSize\": 10,"
	searchResult = strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Info failed to set default RollingDataPoints")
	}
}

func TestIncrMetric(t *testing.T) {
	// Test incrementing a simple metric.
	//
	metricName := "myMetric"
	metricName2 := "myMetric2"

	s := NewState()
	s.SetConfig("test", 10)

	// Test single incr
	s.IncrMetric(metricName)
	result := s.Dump()
	searchFor := "\"" + metricName + "\": 1"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed. Result: %s", result)
	}

	// Test second incr on same metric
	s.IncrMetric(metricName)
	result = s.Dump()
	searchFor = "\"" + metricName + "\": 2"
	searchResult = strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric second increment failed")
	}

	// Test incr on second metric
	s.IncrMetric(metricName2)
	result = s.Dump()
	searchFor = "\"" + metricName2 + "\": 1"
	searchResult = strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Increment on second metric failed")
	}
}

func TestIncrMetricIgnoresEmptyName(t *testing.T) {
	// Test incrementing a metric when supplying no name string
	// ignores the incr and sets no value.
	metricName := ""

	s := NewState()
	s.SetConfig("test", 10)

	// Test single incr
	s.IncrMetric(metricName)
	result := s.Dump()
	searchFor := "\"Metrics\": null"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed")
	}
}
func TestRollingMetricNewValue(t *testing.T) {
	// Test setting a single value for a rolling metric.
	//
	metricName := "myRollingMetric"
	metricValue := 1.0

	s := NewState()
	s.SetConfig("test", 10)
	s.UpdateRollingMetric(metricName, metricValue)
	result := s.Dump()

	searchFor := "\"" + metricName + "\": 0.1"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed")
	}
}

func TestMultipleRollingMetrics(t *testing.T) {
	// Test for correct value when supplying many rolling data points.
	//
	metricName := "myRollingMetric"
	metricName2 := "myRollingMetric2"
	metricValue := 1.0

	s := NewState()
	s.SetConfig("test", 10)
	s.UpdateRollingMetric(metricName, metricValue)
	s.UpdateRollingMetric(metricName2, metricValue)
	result := s.Dump()

	searchFor := "\"" + metricName + "\": 0.1"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed")
	}
}
