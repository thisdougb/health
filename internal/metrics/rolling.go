package metrics

// RollingMetric implements a circular buffer for calculating rolling averages
type RollingMetric struct {
	data  []float64
	index int
}

// Add a value to an existing data array and return the rolling average
func (rm *RollingMetric) Add(value float64) float64 {
	dataLength := len(rm.data)

	// simple index wrap-around technique
	if rm.index >= dataLength {
		rm.index = 0
	}
	rm.data[rm.index] = value

	var total float64
	for i := 0; i < dataLength; i++ {
		total += rm.data[i]
	}
	rm.index++

	return float64(total) / float64(dataLength)
}

// NewRollingMetric creates a new rolling metric with the specified size
func NewRollingMetric(size int) *RollingMetric {
	return &RollingMetric{
		data:  make([]float64, size),
		index: 0,
	}
}
