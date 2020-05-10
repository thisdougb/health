package health

import (
	"strconv"
	"strings"
	"testing"
)

func TestInfoMethodSetters(t *testing.T) {
	// func (s *state) Info(string, int)
	//
	identity := "workerXYZ"
	rDataSize := 5

	var s State
	s.Info(identity, rDataSize)
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
	// func (s *state) Info(string, int)
	//
	identity := ""
	rDataSize := 0

	var s State
	s.Info(identity, rDataSize)
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
	// func (s *state) Info(string, int)
	//
	metricName := "myMetric"
	metricName2 := "myMetric2"

	var s State
	s.Info("test", 10)

	// Test single incr
	s.IncrMetric(metricName)
	result := s.Dump()
	searchFor := "\"" + metricName + "\": 1"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed")
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

func TestRollingMetricNewValue(t *testing.T) {
	// func (s *state) Info(string, int)
	//
	metricName := "myRollingMetric"
	metricValue := 1.0

	var s State
	s.Info("test", 10)
	s.UpdateRollingMetric(metricName, metricValue)
	result := s.Dump()

	searchFor := "\"" + metricName + "\": 0.1"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed")
	}
}

func TestMultipleRollingMetrics(t *testing.T) {
	// func (s *state) Info(string, int)
	//
	metricName := "myRollingMetric"
	metricName2 := "myRollingMetric2"
	metricValue := 1.0

	var s State
	s.Info("test", 10)
	s.UpdateRollingMetric(metricName, metricValue)
	s.UpdateRollingMetric(metricName2, metricValue)
	result := s.Dump()

	searchFor := "\"" + metricName + "\": 0.1"
	searchResult := strings.Index(result, searchFor)
	if searchResult < 0 {
		t.Errorf("Metric increment failed")
	}
}
