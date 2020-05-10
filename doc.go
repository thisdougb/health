/*
Package health provides an easy way to record and report metrics.

A good example is using the health package in a service architecture
running on k8s. Each container can run a /health http handler that
simple returns the json output from health.Dump(). A dashboard can
consume that json ouput, using it for alerting, graphs, logs, etc.

Using a standard metrics output across all app services, means it is
trivial to build a dashboard that auto-discoveres any container.

The intention is that this package is used in a similar way to /proc
on *nix systems. It is the responsibility of the metrics consumer to
handle rates over time (message per second, for example). This keeps
the health package simple.

In a DevOps team, ops can run a consumer at high frequency while
troubleshooting, for example per second. While a log aggregator like
DataDog can consume at per minute.

Example:

	// a unique ID so we know where these metrics came from
	nodeID := "worker-123xyz"

	// sample size for rolling averages
	rollingDataSize := 5

	var s State
	s.Info(nodeID, rollingDataSize)

	for i := 0; i < 10; i++ {
		// simple incrementer metric
		s.IncrMetric("example-counter-metric")

		// add data point for rolling average metric
		s.UpdateRollingMetric("example-avg-metric", float64(i))
	}
	s.Dump("json")

Output:
	{
		"Identity": "node-ac3e6",
		"Started": 1589108939,
		"RollingDataSize": 5,
		"Metrics": {
			"example-counter-metric": 10
		},
		"RollingMetrics": {
			"example-avg-metric": 4.5
		}
	}
*/
package health
