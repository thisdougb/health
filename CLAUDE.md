# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go package for tracking and reporting metrics in containerized applications. The package provides a simple health monitoring system designed for service architectures, particularly in Kubernetes environments where containers expose `/health` HTTP handlers.

## Documentation

- **`./docs/ARCHITECTURE.md`** - Detailed system architecture and component design
- **`./docs/DECISION_LOG.md`** - Historical record of architectural decisions and strategic changes

## Core Architecture

The package is organized by core capabilities:

### 1. Data Methods (Core Metrics Recording)
- **Global metrics**: `IncrMetric()`, `UpdateRollingMetric()` - system-wide counters and averages
- **Component metrics**: `IncrComponentMetric()`, `UpdateComponentRollingMetric()` - organized by application component
- Thread-safe operations using mutex protection around writes

### 2. Data Access (Web Request Handling)  
- `HandleHealthRequest()` - flexible URL pattern handling with external router compatibility
- Component-specific endpoints: `/health/{component}`, `/health/{component}/status`
- JSON serializable output via `Dump()` method

### 3. Storage Models
- **Memory-only** (current) - fast, volatile, ideal for development
- **SQLite persistence** - background sync every ~60 seconds, zero performance impact
- Environment variable configuration for deployment flexibility

### 4. System Metrics (Automatic)
- **Always-on collection** - CPU, memory, goroutines, uptime, health data size
- **Background monitoring** - runs every minute automatically  
- **Zero performance impact** - sub-microsecond operation times maintained
- **Historical storage** - all system metrics persisted for analysis

### 5. Data Management
- **Retention policies** - configurable data lifecycle  
- **Backup integration** - event-driven backups following established patterns
- **Automated cleanup** - background maintenance processes

Key design principles:
- Memory-first approach for zero performance impact
- Component-based organization for complex systems
- External router compatibility (nginx, Kubernetes ingress)
- Go's idiomatic patterns with separate methods for type safety

## Development Commands

This is a standard Go module using Go 1.14+:

```bash
# Run tests
go test

# Run specific test
go test -run TestFunctionName

# Run tests with verbose output
go test -v

# Get dependencies
go mod tidy

# Format code
go fmt ./...

# Build with SQLite persistence (requires CGO)
CGO_ENABLED=1 go build

# Build memory-only version (no CGO required)  
CGO_ENABLED=0 go build

# Cross-compile for Linux from macOS (requires zig)
CC="zig cc -target x86_64-linux" CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build
```

## Testing

The package includes comprehensive tests:
- `health_test.go`: Tests for main State functionality
- `rolling_metric_test.go`: Tests for rolling average calculations

No external testing frameworks are used - standard Go testing only.

## Package Usage Pattern

### Basic Usage (Memory-Only)
```go
// Initialize
state := health.NewState()
state.SetConfig("my-app")

// Counter metrics (stored in memory + persisted)
state.IncrMetric("requests")
state.IncrComponentMetric("webserver", "requests")
state.IncrComponentMetric("database", "queries")

// Raw value metrics (persisted to storage for analysis)
state.AddMetric("response_time", 145.2)
state.AddComponentMetric("cache", "hit_rate", 0.85)

// System metrics are collected automatically in background:
// - cpu_percent, memory_bytes, health_data_size, goroutines, uptime_seconds
// - Runs every minute, stored in persistence backend under "system" component

// Export JSON - shows counter metrics only (raw values go to storage)
json := state.Dump()

// Always close gracefully to flush pending data
defer state.Close()
```

### JSON Output Structure

The package outputs counter metrics in a component-organized structure designed for easy programmatic consumption:

```json
{
    "Identity": "my-app",
    "Started": 1753959967,
    "Metrics": {
        "Global": {
            "requests": 150
        },
        "webserver": {
            "requests": 100
        },
        "database": {
            "queries": 250
        }
    }
}
```

**Key Features:**
- **Component grouping**: Counter metrics are organized by component for easy filtering
- **Global as component**: Global metrics are grouped under "Global" for consistency
- **Real-time counters**: Shows current counter values for immediate status
- **Raw values separate**: Raw metric values are persisted to storage backend for historical analysis
- **Computer-friendly**: Optimized for consumption by tools like `jq` and monitoring systems

### Web Request Handling
```go
// Flexible URL pattern support
http.HandleFunc("/health/", func(w http.ResponseWriter, r *http.Request) {
    if !authenticated(r) {
        http.Error(w, "Unauthorized", 401)
        return
    }
    state.HandleHealthRequest(w, r) // Handles all /health/* patterns
})
```

### Production with Persistence and Backup

**Important: SQLite requires CGO compilation**

SQLite persistence uses CGO, which requires a C compiler. For cross-compilation (especially Linux targets from macOS), use the zig compiler:

```bash
# Install zig compiler (macOS with Homebrew)
brew install zig

# Cross-compile for Linux from macOS
CC="zig cc -target x86_64-linux" CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/myapp main.go

# Standard compilation (same platform)
CGO_ENABLED=1 go build -o bin/myapp main.go

# Memory-only mode (no CGO required)
CGO_ENABLED=0 go build -o bin/myapp main.go
```

```go
// Enable SQLite persistence and event-driven backups via environment variables
// HEALTH_PERSISTENCE_ENABLED=true
// HEALTH_DB_PATH="/data/health.db"
// HEALTH_FLUSH_INTERVAL="60s"  
// HEALTH_BATCH_SIZE="100"
// HEALTH_BACKUP_ENABLED=true
// HEALTH_BACKUP_DIR="/data/backups/health"
// HEALTH_BACKUP_RETENTION_DAYS="30"

state := health.NewState() // Automatically uses env config
state.SetConfig("production-app")

// Counter metrics - stored in memory + persisted
state.IncrMetric("requests")
state.IncrComponentMetric("api", "requests")

// Raw values - persisted only (for historical analysis)
state.AddMetric("response_time", 142.5)
state.AddComponentMetric("database", "query_time", 23.1)

// System metrics automatically collected and persisted every minute
// Available in storage backend under "system" component for monitoring

// Event-driven backup - automatically happens on graceful shutdown
// Also available manually: state.GetStorageManager().CreateBackup()

// Always close gracefully in production (triggers backup if enabled)
defer state.Close()
```

### Administrative Data Extraction (Claude Analysis)
```go
import "github.com/thisdougb/health/internal/handlers"

// Initialize state with persistence enabled
state := health.NewState()
state.SetConfig("production-app")

// ... collect metrics over time ...

// Define time range for analysis
start := time.Now().Add(-24 * time.Hour) // Last 24 hours
end := time.Now()

// 1. List all available components
components, err := handlers.ListAvailableComponents(state)
if err != nil {
    log.Printf("Error listing components: %v", err)
}
// Output: ["Global", "database", "system", "webserver"]

// 2. Extract metrics for specific component
webserverData, err := handlers.ExtractMetricsByTimeRange(state, "webserver", start, end)
if err != nil {
    log.Printf("Error extracting webserver metrics: %v", err)
}
// Returns JSON with time-series data for webserver component

// 3. Export all metrics for comprehensive analysis
allMetrics, err := handlers.ExportAllMetrics(state, start, end, "json")
if err != nil {
    log.Printf("Error exporting metrics: %v", err)
}
// Returns complete dataset with all components and time ranges

// 4. Get statistical health summary
healthSummary, err := handlers.GetHealthSummary(state, start, end)
if err != nil {
    log.Printf("Error getting health summary: %v", err)
}
// Returns aggregated statistics with min/max/avg and health indicators

defer state.Close()
```

**Admin Functions Output Structure**:
```json
{
  "start_time": "2025-07-30T23:41:14Z",
  "end_time": "2025-07-31T23:41:14Z",
  "components": [
    {
      "component": "webserver",
      "metric_count": 15,
      "counters": {
        "http_requests": {"count": 10, "total": 1500}
      },
      "values": {
        "response_time": {"count": 10, "min": 120.5, "max": 180.2, "avg": 145.8}
      }
    }
  ],
  "system_metrics": {
    "cpu_percent": {"count": 24, "min": 15.2, "max": 78.5, "avg": 42.3},
    "memory_bytes": {"count": 24, "min": 1048576, "max": 2097152, "avg": 1572864}
  },
  "overall_summary": {
    "time_span_hours": 24,
    "total_components": 4,
    "total_metrics": 156,
    "system_healthy": true
  }
}
```

**Claude Analysis Benefits**:
- ✅ **Time-based filtering**: Focus analysis on specific time periods
- ✅ **Component isolation**: Analyze individual system components  
- ✅ **Statistical aggregation**: Ready-made min/max/avg calculations
- ✅ **Health indicators**: Automated system health assessment
- ✅ **Performance optimized**: Sub-microsecond to microsecond response times
- ✅ **Structured JSON**: Perfect for programmatic analysis and AI processing

## Development Workflow

### Branch Management
- Branch naming: `{component}_{description}` (e.g., `health_add_metric_validation`)
- Always create feature branches - never commit directly to master
- Use descriptive branch names that indicate the component being modified

### Development Process
1. **Plan**: Create implementation plan and get approval before coding
2. **Branch**: Create new feature branch with descriptive name
3. **Test-First**: Write tests before implementing functionality
4. **Implement**: Write code to make tests pass
5. **Verify**: Run full test suite to ensure no regressions
6. **Commit**: Include collaborative attribution in commit messages

### Testing Requirements
- Write comprehensive tests for all new functionality
- Use table-driven tests for multiple scenarios
- Test both success and error conditions
- Ensure thread safety testing for concurrent access

### Commit Standards
All commits must include collaborative attribution using git config user.name and user.email:
```
Brief description of changes

- Key implementation details
- Any breaking changes

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: $(git config --get user.name) <$(git config --get user.email)>
Co-Authored-By: Claude <noreply@anthropic.com>
```