# health

[![Go Report](https://goreportcard.com/badge/github.com/thisdougb/health)](https://goreportcard.com/badge/github.com/thisdougb/health)

⚠️  **WARNING: BREAKING CHANGES IN PROGRESS**  
This package is currently undergoing a major refactor to improve the API design. Breaking changes will occur without notice. **Do not use in production** until this warning is removed.

---

A lightweight Go package for tracking and reporting application metrics designed for AI-first problem resolution. Import this library to enable AI-powered analysis of your application's behavior and performance patterns.

> **Developed with Claude Code** - This project showcases professional-grade software development using AI pair programming with [Claude Code](https://claude.ai/code).

## Install

```bash
go get -u github.com/thisdougb/health
```

## AI-First Metrics

Instead of staring at dashboards, ask Claude questions about your application. When investigating production issues, you can use the underlying API calls directly with Claude Code for immediate analysis:

```
user: I need to analyze the webserver performance during yesterday's incident. Show me 5-minute aggregated response times from 2pm to 4pm on January 15th.

claude: I'll help you analyze the webserver performance during that time period.

curl "https://myapp.com/health/webserver/timeseries?window=5m&lookback=2h&date=2025-01-15&time=16:00:00"

[Analyzes the JSON response and provides insights about response time patterns, spikes, and correlations with other metrics]
```

Natural language questions Claude can answer:

- *"Claude, show me user signups for this week and how that has affected the significant pinch points of the app"*
- *"What's the correlation between API response times and error rates over the last 24 hours?"*
- *"Which metrics show unusual patterns since the last deployment?"*
- *"Break down performance bottlenecks by feature usage"*

## Features

- **Time Series Analysis** - sar-style queries with intuitive parameters (`?window=5m&lookback=2h`)
- **Component-based organization** - Organize metrics by application component
- **Zero dependencies** - Pure in-memory metrics by default  
- **Schema-less metrics** - Create metrics on-demand with any name
- **Thread-safe** - Safe for concurrent use across goroutines
- **External router compatible** - Works with reverse proxies and load balancers
- **Memory-first persistence** - Optional SQLite with zero performance impact
- **Development-friendly** - No database setup required for development

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/thisdougb/health"
)

var metrics *health.State

func init() {
    // Create a new health state instance
    metrics = health.NewState()
    
    // Configure with unique service identifier
    // "my-service" helps Claude identify this instance in multi-service environments
    metrics.SetConfig("my-service")
}

func main() {
    http.HandleFunc("/api", handleAPI)
    http.HandleFunc("/health", handleHealth)
    http.ListenAndServe(":8080", nil)
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
    // Counter metrics (stored in memory + persisted)
    metrics.IncrMetric("total-requests") 
    metrics.IncrComponentMetric("webserver", "requests")
    metrics.IncrComponentMetric("database", "queries")
    
    // Raw value metrics (persisted to storage for analysis)
    metrics.AddMetric("memory-usage", 1024.5)
    metrics.AddComponentMetric("webserver", "response-time", 245.0)
    metrics.AddComponentMetric("cache", "hit-rate", 0.85)
    
    // Your API logic here...
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
    // Flexible health endpoint handling
    if !authenticated(r) {
        http.Error(w, "Unauthorized", 401)
        return
    }
    metrics.HandleHealthRequest(w, r) // Handles all /health/* patterns
}
```

## Time Series Analysis (sar-style)

Query historical metrics with intuitive parameters for powerful analysis:

```go
// Set up time series endpoints for different components
http.HandleFunc("/health/webserver/timeseries", metrics.TimeSeriesHandler("webserver"))
http.HandleFunc("/health/database/timeseries", metrics.TimeSeriesHandler("database"))
```

**Simple Query Examples:**
```bash
# Look back 2 hours with 5-minute averages - perfect for incident analysis
curl "https://myapp.com/health/webserver/timeseries?window=5m&lookback=2h"

# Analyze specific time period (great for post-mortem analysis)
curl "https://myapp.com/health/database/timeseries?window=1m&lookback=1h&date=2025-01-15&time=14:30:00"

# Forward-looking analysis (prediction scenarios)
curl "https://myapp.com/health/system/timeseries?window=10s&lookahead=30m"
```

**Response Example:**
```json
{
  "component": "webserver",
  "start_time": "2025-01-15T08:00:00Z",
  "end_time": "2025-01-15T10:00:00Z",
  "reference_time": "2025-01-15T10:00:00Z",
  "request_params": {"window": "5m", "lookback": "2h", "date": "2025-01-15", "time": "10:00:00"},
  "metrics": {
    "http_requests": {"08:00:00": 1250, "08:05:00": 1423, "08:10:00": 1567, "08:15:00": 1482},
    "response_time_ms": {"08:00:00": 45.2, "08:05:00": 52.1, "08:10:00": 48.7, "08:15:00": 51.3},
    "error_rate": {"08:00:00": 0.12, "08:05:00": 0.08, "08:10:00": 0.15, "08:15:00": 0.09},
    "active_connections": {"08:00:00": 127, "08:05:00": 143, "08:10:00": 156, "08:15:00": 148}
  }
}
```

**Parameters:**
- `window` - Aggregation period (e.g., "5m", "1h", "30s") 
- `lookback` - Look back from reference time (mutually exclusive with lookahead)
- `lookahead` - Look forward from reference time (mutually exclusive with lookback)
- `date` - Reference date YYYY-MM-DD (defaults to today)
- `time` - Reference time HH:MM:SS (defaults to now)

## AI-Powered Analysis

Enable persistent storage for Claude Code to analyze historical patterns:

```bash
# Enable SQLite storage for AI queries
export HEALTH_PERSISTENCE_ENABLED=true
export HEALTH_DB_PATH=./metrics.db

# Your app runs unchanged - Claude can now query historical data
go run main.go
```

**External Router Compatibility:**
- Reverse proxies: Route `/serviceA/health/` to different backend services
- Load balancers: Component-specific health routing
- Container orchestration: Health checks in containerized environments such as Kubernetes

## Storage Models

| Model | Performance | Persistence | Use Case |
|-------|-------------|-------------|----------|
| Memory-Only | Fastest | None | Development, testing |
| SQLite Background | Fast | Full | Production with history |

**Memory-First Design**: All metric operations use memory for speed. Optional SQLite persistence runs in background Go routines with zero performance impact on your application.

## Memory Requirements

The package has been tested for memory usage at different metric collection rates:

| Rate | 1 Hour | 1 Day | 1 Week | 1 Month | 12 Months |
|------|--------|-------|--------|---------|-----------|
| **100 metrics/sec** | 205 KB | 4.81 MB | 33.69 MB | 144.40 MB | 1.72 GB |
| **1,000 metrics/sec** | 226 KB | 5.30 MB | 37.13 MB | 159.14 MB | 1.89 GB |
| **10,000 metrics/sec** | 237 KB | 5.55 MB | 38.86 MB | 166.54 MB | 1.98 GB |

**Key Characteristics:**
- Base memory overhead: ~200 KB regardless of collection rate
- Per-metric overhead: 0.01-0.58 bytes per metric (highly efficient)
- JSON serialization: Typically 4-5 KB for standard counter output
- System metrics: Automatic background collection adds negligible overhead

## Configuration

All configuration via environment variables with sensible defaults:

```bash
# Persistence (default: memory-only)
HEALTH_PERSISTENCE_ENABLED=true       # Enable SQLite persistence
HEALTH_DB_PATH="./health.db"          # Database file path
HEALTH_FLUSH_INTERVAL="60s"           # Background sync interval
HEALTH_BATCH_SIZE="100"               # Metrics per batch write

# Backup management
HEALTH_BACKUP_ENABLED=true            # Enable backups
HEALTH_BACKUP_DIR="./backups"         # Backup directory
HEALTH_BACKUP_RETENTION_DAYS="30"     # Days to keep backups
```

## API Reference

### Core Methods

**Initialization:**
- `NewState()` - Create new health state instance
- `SetConfig(identity)` - Configure metrics instance

**Counter Metrics (Memory + Persistence):**
- `IncrMetric(name)` - Increment global counter
- `IncrComponentMetric(component, name)` - Increment component counter

**Raw Value Metrics (Persistence Only):**
- `AddMetric(name, value)` - Add global raw value for analysis
- `AddComponentMetric(component, name, value)` - Add component raw value for analysis

**Data Access:**
- `Dump()` - Export current state as JSON
- `HandleHealthRequest(w, r)` - Handle flexible health URL patterns
- `TimeSeriesHandler(component)` - Generate time series analysis handler

**HTTP Handlers:**
- `HealthHandler()` - Standard /health endpoint
- `StatusHandler()` - Simple UP/DOWN status check
- `TimeSeriesHandler(component)` - sar-style time series queries

## Tutorial

For a complete step-by-step tutorial showing how to build a real application with health metrics, see **[docs/TUTORIAL.md](docs/TUTORIAL.md)**. 

The tutorial walks through building "Doug's Diner" - a restaurant management system that demonstrates:
- Project structure and file organization
- Component-based metrics implementation
- HTTP endpoints with authentication
- Testing with standard library
- Production deployment considerations

Perfect for developers new to Go who want to learn both application architecture and metrics integration.

## Design Philosophy

- **Memory-first performance** - Zero performance impact on applications
- **Component-based organization** - Structure metrics by application components
- **Flexible routing** - Compatible with reverse proxies and load balancers
- **Development-friendly** - No database setup required for development
- **Production-ready persistence** - Optional SQLite with background sync
- **Go idioms** - Separate methods for type safety (no complex variadic parameters)
- **Progressive enhancement** - Add features without breaking simplicity
- **Zero breaking changes** - Existing code continues to work unchanged
