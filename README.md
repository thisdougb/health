[![thisdougb](https://circleci.com/gh/thisdougb/health.svg?style=shield)](https://circleci.com/gh/thisdougb/health)

# health
A Go package to make tracking and reporting metrics in containers app's easy.

A good example is using the health package in a service architecture
running on k8s. Each container can run a /health http handler that
simple returns the json output from health.Dump(). A dashboard can
consume that json ouput, using it for alerting, graphs, logs, etc.

Using a standard metrics output across all app services, means it is
trivial to build a dashboard that auto-discovers any container.

The intention is that this package is used in a similar way to /proc
on *nix systems. It is the responsibility of the metrics consumer to
handle rates over time (message per second, for example). This keeps
the health package simple.

In a DevOps team, ops can run a consumer at high frequency while
troubleshooting, for example per second. While a log aggregator like
DataDog can consume at per minute.


## Install
```
go get -u -v github.com/thisdougb/health
```
## Example
I use the package to track metrics in container apps that have a /health http handler.
```
// Example using health metrics.
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/thisdougb/health"
)

var s health.State

func main() {

	nodeName := "node-ac3e6"
	rollingDataSize := 5
	s.Info(nodeName, rollingDataSize)

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/health", handleHealth)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	s.IncrMetric("indexRequest")
	// doSomethingUseful()
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Health: %s\n", s.Dump())
}
```
When we call our web app, and then /health we can see the metric:
```
$ curl http://localhost:8000/
$ curl http://localhost:8000/health
Health: {
    "Identity": "node-ac3e6",
    "Started": 1589113356,
    "RollingDataSize": 5,
    "Metrics": {
        "indexRequest": 1
    },
    "RollingMetrics": null
} 
```
