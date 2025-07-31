# health

[![Go Report](https://goreportcard.com/badge/github.com/thisdougb/health)](https://goreportcard.com/badge/github.com/thisdougb/health)

⚠️  **WARNING: BREAKING CHANGES IN PROGRESS**  
This package is currently undergoing a major refactor to improve the API design. Breaking changes will occur without notice. **Do not use in production** until this warning is removed.

---

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

- **Component-based organization** - Organize metrics by application component
- **Zero dependencies** - Pure in-memory metrics by default  
- **Schema-less metrics** - Create metrics on-demand with any name
- **Thread-safe** - Safe for concurrent use across goroutines
- **External router compatible** - Works with nginx, Kubernetes ingress
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
    
    // Configure with unique service identifier and rolling window size
    // "my-service" helps Claude identify this instance in multi-service environments
    // 10 is the sample size for calculating rolling averages
    metrics.SetConfig("my-service", 10)
}

func main() {
    http.HandleFunc("/api", handleAPI)
    http.HandleFunc("/health", handleHealth)
    http.ListenAndServe(":8080", nil)
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
    // Global metrics (system-wide)
    metrics.IncrMetric("total-requests") 
    metrics.UpdateRollingMetric("memory-usage", 1024.5)
    
    // Component-based metrics (organized by application component)
    metrics.IncrComponentMetric("webserver", "requests")
    metrics.IncrComponentMetric("database", "queries")
    metrics.UpdateComponentRollingMetric("webserver", "response-time", 245.0)
    metrics.UpdateComponentRollingMetric("cache", "hit-rate", 0.85)
    
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

## AI-Powered Analysis

Enable persistent storage for Claude Code to analyze historical patterns:

```bash
# Enable SQLite storage for AI queries
export HEALTH_STORAGE=sqlite
export HEALTH_SQLITE_FILE=./metrics.db

# Your app runs unchanged - Claude can now query historical data
go run main.go
```

**External Router Compatibility:**
- nginx: Route `/serviceA/health/` to different backend clusters
- Kubernetes ingress: Different paths to different services  
- Load balancers: Component-specific health routing

## Storage Models

| Model | Performance | Persistence | Use Case |
|-------|-------------|-------------|----------|
| Memory-Only | Fastest | None | Development, testing |
| SQLite Background | Fast | Full | Production with history |

**Memory-First Design**: All metric operations use memory for speed. Optional SQLite persistence runs in background Go routines with zero performance impact on your application.

## Configuration

All configuration via environment variables with sensible defaults:

```bash
# Persistence (default: memory-only)
HEALTH_PERSISTENCE_ENABLED=true       # Enable SQLite persistence
HEALTH_DB_PATH="./health.db"          # Database file path
HEALTH_FLUSH_INTERVAL="60s"           # Background sync interval

# Data management
HEALTH_RETENTION_DAYS=30              # Keep data for 30 days
HEALTH_BACKUP_ENABLED=true            # Enable backups
HEALTH_BACKUP_DIR="./backups"         # Backup directory
```

## API Reference

### Core Methods

**Initialization:**
- `NewState()` - Create new health state instance
- `SetConfig(identity, rollingSize)` - Configure metrics instance

**Global Metrics:**
- `IncrMetric(name)` - Increment global counter
- `UpdateRollingMetric(name, value)` - Update global rolling average

**Component-Based Metrics:**
- `IncrComponentMetric(component, name)` - Increment component counter
- `UpdateComponentRollingMetric(component, name, value)` - Update component rolling average

**Data Access:**
- `Dump()` - Export current state as JSON
- `HandleHealthRequest(w, r)` - Handle flexible health URL patterns

## Design Philosophy

- **Memory-first performance** - Zero performance impact on applications
- **Component-based organization** - Structure metrics by application components
- **External router compatible** - Works with nginx, Kubernetes ingress routing
- **Development-friendly** - No database setup required for development
- **Production-ready persistence** - Optional SQLite with background sync
- **Go idioms** - Separate methods for type safety (no complex variadic parameters)
- **Progressive enhancement** - Add features without breaking simplicity
- **Zero breaking changes** - Existing code continues to work unchanged