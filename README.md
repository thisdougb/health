# health
An easy way to track metrics in Go apps.

In a DevOps environment running containers, speed and simplicity are desirable qualities.
The motivation for this package is to enable quick implementation of application metrics, and easy consumption.


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
