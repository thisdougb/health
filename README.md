# health

[![Go Report](https://goreportcard.com/badge/github.com/thisdougb/health)](https://goreportcard.com/badge/github.com/thisdougb/health)

A lightweight Go package for tracking and reporting metrics designed for AI-powered analysis. Built for Claude Code to answer complex questions about your application's behavior.

> **Developed with Claude Code** - This project showcases professional-grade software development using AI pair programming with [Claude Code](https://claude.ai/code).

## Install

```bash
go get -u github.com/thisdougb/health
```

## AI-First Metrics

Instead of staring at dashboards, ask Claude questions about your application:

- *"Claude, show me user signups for this week and how that has affected the significant pinch points of the app"*
- *"What's the correlation between API response times and error rates over the last 24 hours?"*
- *"Which metrics show unusual patterns since the last deployment?"*
- *"Break down performance bottlenecks by feature usage"*

## Features

- **Zero dependencies** - Pure in-memory metrics by default
- **Schema-less metrics** - Create metrics on-demand with any name
- **Thread-safe** - Safe for concurrent use across goroutines
- **Claude Code optimized** - SQL queryable storage for AI analysis
- **Pluggable storage** - Optional SQLite, InfluxDB, or custom backends
- **Embeddable first** - Library, not service architecture

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/thisdougb/health"
)

// Global metrics instance - shared across your application
var metrics health.State

func main() {
    // Initialize metrics with a unique service identifier and rolling window size
    // "my-service" helps Claude identify this instance in multi-service environments
    // 10 is the sample size for calculating rolling averages
    metrics.Info("my-service", 10)
    
    http.HandleFunc("/api", handleAPI)
    http.HandleFunc("/health", handleHealth)
    http.ListenAndServe(":8080", nil)
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
    // Schema-less metrics - no pre-definition needed, created on first use
    
    // Simple counters: increment by 1 each time
    metrics.IncrMetric("user-signups")    // Tracks total signups
    metrics.IncrMetric("api-requests")    // Tracks total API calls
    
    // Rolling averages: tracks recent values for trend analysis
    metrics.UpdateRollingMetric("response-time", 245.0)  // Current response time in ms
    metrics.UpdateRollingMetric("cpu-usage", 67.2)       // Current CPU percentage
    
    // Your API logic here...
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
    // Export current metrics as JSON for monitoring systems or Claude Code analysis
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, "%s\n", metrics.Dump())
}
```

## AI-Powered Analysis

Enable persistent storage for Claude Code to analyze historical patterns:

```bash
# Enable SQLite storage for AI queries
export HEALTH_STORAGE=sqlite
export HEALTH_SQLITE_FILE=./metrics.db

# Your app runs unchanged - Claude can now query historical data
go run main.go
```

**Claude Code can now analyze patterns by requesting raw metric data:**
- Request metrics for specific names over time periods
- Claude interprets the raw data to answer complex questions
- No complex queries needed - simple data retrieval with time ranges

## Storage Backends

| Backend | Status | Claude Integration |
|---------|--------|--------------------|
| In-Memory | ✅ Default | Current state only |
| SQLite | ✅ Available | Full historical analysis |
| InfluxDB | 🚧 Planned | Time-series queries |
| Prometheus | 🚧 Planned | Metrics ecosystem |
| CloudWatch | 🚧 Planned | AWS integration |

## Configuration

All configuration via environment variables with sensible defaults:

```bash
# Storage backend (default: in-memory only)
HEALTH_STORAGE=sqlite|influxdb|prometheus

# SQLite specific (optimized for Claude Code queries)
HEALTH_SQLITE_FILE=./health.db        # Database file path
HEALTH_BATCH_SIZE=100                 # Batch write size
HEALTH_BATCH_INTERVAL_MS=1000         # Batch write interval

# Enable WAL mode for better concurrency (default: true)
HEALTH_WAL_ENABLED=true
```

## API Reference

### Core Methods
- `Info(identity, rollingSize)` - Initialize metrics instance
- `IncrMetric(name)` - Increment counter metric
- `UpdateRollingMetric(name, value)` - Add data point to rolling average
- `Dump()` - Export current state as JSON

### Claude Code Integration Methods (when storage enabled)
- `GetMetrics(names, startTime, endTime, granularity)` - Retrieve raw metric data
- `ExportData(since)` - Export metrics since timestamp

## Design Philosophy

- **AI-first interface** - Built for Claude Code analysis, not human dashboards
- **Simple data retrieval** - Endpoints return metrics for `{name1, name2}` by `{startTime, endTime, granularity}`
- **Claude interprets complexity** - No clever queries or optimizations, just raw data blocks for AI analysis
- **Embeddable first** - Library, not service architecture
- **Progressive enhancement** - Add features without breaking simplicity
- **Graceful degradation** - Falls back to in-memory if storage fails
- **Zero breaking changes** - Existing code continues to work unchanged