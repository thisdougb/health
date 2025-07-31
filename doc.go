/*
Package health provides an easy way to record and report metrics.

⚠️  WARNING: BREAKING CHANGES IN PROGRESS
This package is currently undergoing a major refactor to improve the API design.
Breaking changes will occur without notice. Do not use in production until this
warning is removed.

A good example is using the health package in a service architecture
running on k8s. Each container can run a /health http handler that
simply returns the json output from health.Dump(). A dashboard can
consume that json output, using it for alerting, graphs, logs, etc.

Using a standard metrics output across all app services, means it is
trivial to build a dashboard that auto-discovers any container.

The intention is that this package is used in a similar way to /proc
on *nix systems. It is the responsibility of the metrics consumer to
handle rates over time (message per second, for example). This keeps
the health package simple.

In a DevOps team, ops can run a consumer at high frequency while
troubleshooting, for example per second. While a log aggregator like
DataDog can consume at per minute.

Example:

	// Create a new health state instance
	s := health.NewState()
	
	// Configure with unique ID and rolling average sample size
	s.SetConfig("worker-123xyz", 5)

	for i := 0; i < 10; i++ {
		// Simple incrementer metric
		s.IncrMetric("example-counter-metric")

		// Add data point for rolling average metric
		s.UpdateRollingMetric("example-avg-metric", float64(i))
		
		// Component-specific metrics
		s.IncrComponentMetric("webserver", "requests")
		s.UpdateComponentRollingMetric("database", "query-time", float64(i*10))
	}
	
	// Export as JSON
	jsonOutput := s.Dump()

Output:
	{
		"Identity": "worker-123xyz",
		"Started": 1589108939,
		"RollingDataSize": 5,
		"Metrics": {
			"example-counter-metric": 10,
			"webserver_requests": 10
		},
		"RollingMetrics": {
			"example-avg-metric": 4.5,
			"database_query-time": 45
		}
	}
*/
package health
