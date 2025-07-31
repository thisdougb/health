package metrics

import "testing"

func TestAddNewValue(t *testing.T) {
	// Test Add() correctly adds a value to the data points array

	testValue := 1.0
	testDataLength := 3

	rm := NewRollingMetric(testDataLength)
	rm.Add(testValue)

	if rm.data[0] != testValue {
		t.Errorf("Adding value %f to RollingMetric failed, got: %f", testValue, rm.data[0])
	}
}

func TestAddValuesToWrap(t *testing.T) {
	// Test Add() correctly wraps its index pointer when adding many data points,
	// the index point loops so the fourth value ends up in index 0

	testValues := [4]float64{1.0, 2.0, 3.0, 4.0} // 4 should wrap back to index 0
	testDataLength := 3

	rm := NewRollingMetric(testDataLength)

	for _, value := range testValues {
		rm.Add(value)
	}

	if rm.data[0] != 4 {
		t.Errorf("Error wrapping around data index while adding values")
	}
}
