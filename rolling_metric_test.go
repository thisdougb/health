package health

import "testing"

func TestAddNewValue(t *testing.T) {

	testValue := 1.0
	testDataLength := 3

	var rm rollingMetric
	rm.data = make([]float64, testDataLength)

	rm.Add(testValue)

	if rm.data[0] != testValue {
		t.Errorf("Adding value %f to movingAverageMetric failed, got: %f", testValue, rm.data[0])
	}

}

func TestAddValuesToWrap(t *testing.T) {

	testValues := [4]float64{1.0, 2.0, 3.0, 4.0} // 4 should wrap back to index 0
	testDataLength := 3

	var rm rollingMetric
	rm.data = make([]float64, testDataLength)

	for _, value := range testValues {
		rm.Add(value)
	}

	if rm.data[0] != 4 {
		t.Errorf("Error wrapping around data index while adding values")
	}

}
