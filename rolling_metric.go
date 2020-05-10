package health

type rollingMetric struct {
	data  []float64
	index int
}

// Add a value to an existing data array
func (rm *rollingMetric) Add(value float64) float64 {

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
