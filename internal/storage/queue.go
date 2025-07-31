package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// SQLiteWriteQueue manages async batch writing to SQLite
type SQLiteWriteQueue struct {
	db            *sql.DB
	flushInterval time.Duration
	batchSize     int
	queue         []MetricEntry
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	insertStmt    *sql.Stmt
}

// NewSQLiteWriteQueue creates a new async write queue for SQLite
func NewSQLiteWriteQueue(db *sql.DB, flushInterval time.Duration, batchSize int) *SQLiteWriteQueue {
	ctx, cancel := context.WithCancel(context.Background())

	return &SQLiteWriteQueue{
		db:            db,
		flushInterval: flushInterval,
		batchSize:     batchSize,
		queue:         make([]MetricEntry, 0, batchSize),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the background queue processing
func (q *SQLiteWriteQueue) Start() error {
	// Prepare insert statement
	stmt, err := q.db.Prepare(`
		INSERT INTO metrics (timestamp, component, name, value, type) 
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	q.insertStmt = stmt

	// Start background goroutine
	q.wg.Add(1)
	go q.processQueue()

	return nil
}

// Stop gracefully shuts down the queue processing
func (q *SQLiteWriteQueue) Stop() {
	// Cancel context to signal shutdown
	q.cancel()

	// Wait for background goroutine to finish
	q.wg.Wait()

	// Close prepared statement
	if q.insertStmt != nil {
		q.insertStmt.Close()
	}

	// Flush any remaining items
	q.flushQueue()
}

// Enqueue adds metrics to the write queue
func (q *SQLiteWriteQueue) Enqueue(metrics []MetricEntry) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Add metrics to queue
	q.queue = append(q.queue, metrics...)

	// Flush if batch size reached
	if len(q.queue) >= q.batchSize {
		return q.flushQueueUnsafe()
	}

	return nil
}

// processQueue runs the background queue processing loop
func (q *SQLiteWriteQueue) processQueue() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			// Shutdown signal received
			return
		case <-ticker.C:
			// Periodic flush
			q.flushQueue()
		}
	}
}

// flushQueue writes all queued metrics to SQLite (thread-safe)
func (q *SQLiteWriteQueue) flushQueue() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.flushQueueUnsafe(); err != nil {
		log.Printf("Failed to flush queue: %v", err)
	}
}

// flushQueueUnsafe writes all queued metrics to SQLite (assumes lock held)
func (q *SQLiteWriteQueue) flushQueueUnsafe() error {
	if len(q.queue) == 0 {
		return nil
	}

	// Start transaction
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	// Prepare statement within transaction
	stmt := tx.Stmt(q.insertStmt)
	defer stmt.Close()

	// Insert all queued metrics
	for _, metric := range q.queue {
		// Convert value to float64 for storage
		var value float64
		switch v := metric.Value.(type) {
		case int:
			value = float64(v)
		case float64:
			value = v
		default:
			log.Printf("Unknown metric value type: %T", v)
			continue
		}

		_, err := stmt.Exec(
			metric.Timestamp.Unix(),
			metric.Component,
			metric.Name,
			value,
			metric.Type,
		)
		if err != nil {
			log.Printf("Failed to insert metric: %v", err)
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	// Clear queue after successful write
	q.queue = q.queue[:0]

	return nil
}

// QueueSize returns the current number of queued items (for testing)
func (q *SQLiteWriteQueue) QueueSize() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// ForceFlush immediately flushes all queued items (for testing)
func (q *SQLiteWriteQueue) ForceFlush() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.flushQueueUnsafe()
}
