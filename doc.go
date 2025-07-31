/*
Package health provides an easy way to record and report metrics in containerized applications.

The package supports two types of metrics:
- Counter metrics: Simple incrementers stored in memory and optionally persisted
- Raw value metrics: Individual values persisted to storage backend for analysis

A good example is using the health package in a service architecture
running on k8s. Each container can run a /health http handler that
returns the json output from health.Dump() for real-time counter metrics.
Historical analysis is performed by querying the persistence backend directly.

Using a standard metrics output across all app services means it is
trivial to build a dashboard that auto-discovers any container.

The package supports optional persistence to SQLite for historical analysis,
configured via environment variables. When persistence is disabled, the
package works purely in-memory like the original design.

Example:

	// Create a new health state instance (automatically loads persistence config)
	s := health.NewState()
	defer s.Close() // Always close gracefully to flush pending data

	// Configure with unique instance identifier
	s.SetConfig("worker-123xyz")

	for i := 0; i < 10; i++ {
		// Counter metrics (stored in memory + persisted)
		s.IncrMetric("requests")
		s.IncrComponentMetric("webserver", "requests")

		// Raw value metrics (persisted to storage backend for analysis)
		s.AddMetric("response_time", float64(i*10+50))
		s.AddComponentMetric("database", "query_time", float64(i*5+10))
	}

	// Export counter metrics as JSON (raw values are in storage backend)
	jsonOutput := s.Dump()

Output (counter metrics only):

	{
		"Identity": "worker-123xyz", 
		"Started": 1589108939,
		"Metrics": {
			"Global": {
				"requests": 10
			},
			"webserver": {
				"requests": 10
			}
		}
	}

Persistence Configuration:

	// Enable SQLite persistence via environment variables
	HEALTH_PERSISTENCE_ENABLED=true
	HEALTH_DB_PATH="/data/health.db"
	HEALTH_FLUSH_INTERVAL="60s"
	HEALTH_BATCH_SIZE="100"

The separation of counter metrics (real-time status) and raw values (historical analysis)
provides flexibility for different use cases while maintaining high performance for
metric collection operations.
*/
package health