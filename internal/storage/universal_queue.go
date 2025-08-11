package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// MetricsQueue is a universal queue that works with any storage backend.
// It handles async batch processing of raw metrics before passing them to the configured backend.
// This ensures consistent behavior between different storage backends (memory, SQLite, etc.).
type MetricsQueue struct {
	backend       Backend        // The configured storage backend (memory, SQLite, etc.)
	flushInterval time.Duration  // How often to flush queued metrics to backend
	batchSize     int           // Maximum queue size before triggering flush
	queue         []MetricEntry // In-memory queue of raw metrics waiting to be processed
	mu            sync.Mutex    // Protects queue operations
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewMetricsQueue creates a new universal metrics queue that works with any backend.
// The queue handles batching, timing, and processing of raw metrics before sending to backend.
//
// Parameters:
//   - backend: Any storage backend implementation (memory, SQLite, etc.)
//   - flushInterval: How often to process queued metrics (e.g., 60s)
//   - batchSize: Maximum metrics to queue before auto-flush (e.g., 100)
func NewMetricsQueue(backend Backend, flushInterval time.Duration, batchSize int) *MetricsQueue {
	ctx, cancel := context.WithCancel(context.Background())

	return &MetricsQueue{
		backend:       backend,
		flushInterval: flushInterval,
		batchSize:     batchSize,
		queue:         make([]MetricEntry, 0, batchSize),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins background processing of the metrics queue.
// This starts a goroutine that periodically flushes queued metrics to the backend.
// Must be called before using Enqueue().
func (q *MetricsQueue) Start() error {
	// Start background queue processing
	q.wg.Add(1)
	go q.processQueue()

	return nil
}

// Stop gracefully shuts down the queue processing.
// Waits for background processing to complete and flushes any remaining metrics.
// Should be called during application shutdown.
func (q *MetricsQueue) Stop() {
	// Signal shutdown to background goroutine
	q.cancel()

	// Wait for background processing to complete
	q.wg.Wait()

	// Flush any remaining queued metrics
	q.flushQueue()
}

// Enqueue adds raw metrics to the processing queue.
// Metrics are queued for async processing and will be flushed to the backend either:
// 1. When batch size is reached (immediate flush)
// 2. When flush interval timer expires (periodic flush)
//
// This method is thread-safe and non-blocking.
func (q *MetricsQueue) Enqueue(metrics []MetricEntry) error {
	if len(metrics) == 0 {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Add metrics to queue
	q.queue = append(q.queue, metrics...)

	// Auto-flush if batch size reached
	if len(q.queue) >= q.batchSize {
		return q.flushQueueUnsafe()
	}

	return nil
}

// ForceFlush immediately processes all queued metrics and sends them to the backend.
// This is useful for testing or when you need to ensure all metrics are persisted.
// This method is thread-safe and will block until flush completes.
func (q *MetricsQueue) ForceFlush() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.flushQueueUnsafe()
}

// processQueue runs the background queue processing loop.
// This handles periodic flushing based on flushInterval.
// Runs in a separate goroutine started by Start().
func (q *MetricsQueue) processQueue() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			// Shutdown signal received
			return
		case <-ticker.C:
			// Periodic flush based on timer
			q.flushQueue()
		}
	}
}

// flushQueue processes all queued metrics and sends them to the backend.
// This is the thread-safe version that acquires the lock.
func (q *MetricsQueue) flushQueue() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.flushQueueUnsafe(); err != nil {
		log.Printf("Failed to flush metrics queue: %v", err)
	}
}

// flushQueueUnsafe processes queued metrics and performs CRUD operations on backend.
// Assumes the caller holds the queue lock (mu).
// 
// Processing steps:
// 1. Takes snapshot of current queue
// 2. Clears the queue to allow new metrics during processing  
// 3. Aggregates raw metrics into storage format
// 4. Calls simple CRUD operations on backend to store processed data
func (q *MetricsQueue) flushQueueUnsafe() error {
	if len(q.queue) == 0 {
		return nil // Nothing to flush
	}

	// Take snapshot of current queue for processing
	metricsToProcess := make([]MetricEntry, len(q.queue))
	copy(metricsToProcess, q.queue)

	// Clear queue immediately to allow new metrics during backend processing
	q.queue = q.queue[:0]

	// Process and aggregate raw metrics into storage format
	// Queue handles all business logic, backends only do CRUD
	aggregatedData := q.aggregateMetrics(metricsToProcess)

	// Call simple CRUD operation on backend to store processed data
	if err := q.backend.WriteMetricsData(aggregatedData); err != nil {
		return fmt.Errorf("backend CRUD operation failed: %w", err)
	}

	return nil
}

// aggregateMetrics processes raw metrics into the storage format.
// This handles all business logic including time windowing and statistical aggregation.
// Returns data in the format expected by backend CRUD operations.
func (q *MetricsQueue) aggregateMetrics(rawMetrics []MetricEntry) []MetricsDataEntry {
	// Group metrics by component, name, and time window for aggregation
	groups := make(map[string]*aggregationGroup)

	for _, metric := range rawMetrics {
		// Create time window key (truncate to minute for aggregation)
		timeWindow := metric.Timestamp.Truncate(time.Minute)
		windowKey := timeWindow.Format("20060102150405")

		// Create unique key for grouping: component+metric+timewindow
		groupKey := fmt.Sprintf("%s|%s|%s", metric.Component, metric.Name, windowKey)

		// Initialize group if first time seeing this combination
		if groups[groupKey] == nil {
			groups[groupKey] = &aggregationGroup{
				Component:     metric.Component,
				Metric:        metric.Name,
				TimeWindowKey: windowKey,
				Values:        make([]float64, 0),
			}
		}

		// Convert metric value to float64 for aggregation
		if floatValue, ok := convertToFloat64(metric.Value); ok {
			groups[groupKey].Values = append(groups[groupKey].Values, floatValue)
		}
	}

	// Convert aggregated groups into storage entries
	var result []MetricsDataEntry
	for _, group := range groups {
		if len(group.Values) == 0 {
			continue // Skip groups with no valid values
		}

		// Calculate statistical aggregates
		min, max, avg := calculateStats(group.Values)

		entry := MetricsDataEntry{
			TimeWindowKey: group.TimeWindowKey,
			Component:     group.Component,
			Metric:        group.Metric,
			MinValue:      min,
			MaxValue:      max,
			AvgValue:      avg,
			Count:         len(group.Values),
		}

		result = append(result, entry)
	}

	return result
}

// aggregationGroup holds metrics for statistical aggregation
type aggregationGroup struct {
	Component     string
	Metric        string
	TimeWindowKey string
	Values        []float64
}

// convertToFloat64 converts various numeric types to float64 for aggregation
func convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

// calculateStats computes min, max, and average from a slice of float64 values
func calculateStats(values []float64) (min, max, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	min = values[0]
	max = values[0]
	sum := 0.0

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	avg = sum / float64(len(values))
	return min, max, avg
}