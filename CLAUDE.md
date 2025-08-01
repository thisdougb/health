# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go package for tracking and reporting application metrics designed for AI-first problem resolution. The package provides a simple health monitoring system for service architectures, with compatibility for environments such as containerized deployments where applications expose `/health` HTTP handlers.

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
- **Time Series Queries** - sar-style analysis with intuitive parameters:
  - `?window=5m&lookback=2h` - 5-minute averages looking back 2 hours
  - `?window=1m&lookahead=30m&date=2025-01-15&time=14:30:00` - forward analysis from specific time
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
- External router compatibility with reverse proxies and load balancers
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

### Time Series Analysis (sar-style Queries)

The package provides intuitive time series analysis with sar-style parameters for historical and predictive analysis:

```go
// Set up time series endpoints for different components
http.HandleFunc("/health/webserver/timeseries", state.TimeSeriesHandler("webserver"))
http.HandleFunc("/health/database/timeseries", state.TimeSeriesHandler("database"))
http.HandleFunc("/health/system/timeseries", state.TimeSeriesHandler("system"))
```

**Query Examples:**
```bash
# Look back 2 hours with 5-minute aggregation windows
curl "http://localhost:8080/health/webserver/timeseries?window=5m&lookback=2h"

# Analyze specific time period (great for incident analysis)
curl "http://localhost:8080/health/database/timeseries?window=1m&lookback=1h&date=2025-01-15&time=14:30:00"

# Forward-looking analysis (prediction scenarios)
curl "http://localhost:8080/health/system/timeseries?window=10s&lookahead=30m"

# Different time formats supported
curl "http://localhost:8080/health/api/timeseries?window=30s&lookback=4h&time=09:15"
```

**Response Format:**
```json
{
  "component": "webserver",
  "window": "5m0s",
  "direction": "lookback",
  "duration": "2h0m0s", 
  "start_time": "2025-01-15T08:00:00Z",
  "end_time": "2025-01-15T10:00:00Z",
  "reference_time": "2025-01-15T10:00:00Z",
  "metrics": {
    "requests_per_sec": {
      "08:00:00": 125.5,
      "08:05:00": 142.3,
      "08:10:00": 156.7,
      "08:15:00": 148.2
    },
    "response_time": {
      "08:00:00": 45.2,
      "08:05:00": 52.1,
      "08:10:00": 48.7,
      "08:15:00": 51.3
    }
  }
}
```

**Key Parameters:**
- `window` - Aggregation period (e.g., "5m", "1h", "30s") 
- `lookback` - Look back from reference time (mutually exclusive with lookahead)
- `lookahead` - Look forward from reference time (mutually exclusive with lookback)
- `date` - Reference date in YYYY-MM-DD format (defaults to today)
- `time` - Reference time in HH:MM:SS format (defaults to now)

**Use Cases:**
- **Incident analysis**: `lookback` from specific incident time
- **Performance trending**: Regular `lookback` queries for dashboards  
- **Capacity planning**: `lookahead` queries for prediction scenarios
- **System monitoring**: Automated queries with different window sizes

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

## Configuration Guide for Junior Developers

### Environment Variables Reference

The health package uses environment variables for configuration to enable zero-code deployment changes. This section provides detailed explanations for junior developers.

#### Basic Persistence Configuration
```bash
# Enable SQLite persistence (default: false - uses memory only)
# When false: All data is stored in memory and lost on restart
# When true: Raw values are persisted to SQLite, counters remain in memory
export HEALTH_PERSISTENCE_ENABLED=true

# SQLite database file path (required when persistence enabled)
# This file will be created automatically if it doesn't exist
# Use absolute paths in production to avoid issues with working directory changes
export HEALTH_DB_PATH=/data/health.db

# How often to flush data from memory queue to SQLite (default: 60s)
# Shorter intervals: Less data loss if application crashes, but more I/O
# Longer intervals: Better performance, but potential data loss on crash
export HEALTH_FLUSH_INTERVAL=30s

# How many metrics to batch together before writing to SQLite (default: 100)
# Higher values: Better write performance, uses more memory
# Lower values: Less memory usage, more frequent disk writes
export HEALTH_BATCH_SIZE=50
```

#### Backup Configuration
```bash
# Enable automatic backups (default: false)
# Backups are event-driven (not scheduled) - triggered by graceful shutdown
export HEALTH_BACKUP_ENABLED=true

# Directory where backup files are stored (default: ./backups)
# Directory will be created automatically if it doesn't exist
# Use absolute paths in production environments
export HEALTH_BACKUP_DIR=/data/backups/health

# How many days to keep backup files (default: 30)
# Older backups are automatically deleted to prevent disk space issues
# Note: You should also manage external backup storage and archival as needed
export HEALTH_BACKUP_RETENTION_DAYS=7
```

### Configuration Examples by Environment

#### Development Environment (Local Machine)
```bash
# Minimal configuration for development
# Uses memory-only storage for fast iteration and clean state between runs

# No environment variables needed - memory backend is default
# Simply use: state := NewState()

# For testing persistence features during development:
export HEALTH_PERSISTENCE_ENABLED=true
export HEALTH_DB_PATH=/tmp/health_dev.db
export HEALTH_FLUSH_INTERVAL=1s  # Fast flush for immediate feedback
```

#### Testing Environment (CI/CD)
```bash
# Configuration for automated testing
# Uses temporary files that are cleaned up automatically

export HEALTH_PERSISTENCE_ENABLED=true
export HEALTH_DB_PATH=/tmp/health_test_${BUILD_NUMBER}.db
export HEALTH_FLUSH_INTERVAL=100ms  # Very fast for test speed
export HEALTH_BATCH_SIZE=10         # Small batches for test predictability

# Backup testing
export HEALTH_BACKUP_ENABLED=true
export HEALTH_BACKUP_DIR=/tmp/health_test_backups_${BUILD_NUMBER}
export HEALTH_BACKUP_RETENTION_DAYS=1  # Short retention for testing
```

#### Staging Environment (Pre-Production)
```bash
# Configuration that mirrors production but with more aggressive settings
# for testing production scenarios

export HEALTH_PERSISTENCE_ENABLED=true
export HEALTH_DB_PATH=/var/lib/health/staging.db
export HEALTH_FLUSH_INTERVAL=30s    # More frequent than production
export HEALTH_BATCH_SIZE=50         # Smaller than production for faster feedback

# Backup configuration
export HEALTH_BACKUP_ENABLED=true
export HEALTH_BACKUP_DIR=/var/lib/health/backups
export HEALTH_BACKUP_RETENTION_DAYS=7  # Shorter retention than production
```

#### Production Environment (Live System)
```bash
# Production configuration optimized for performance and reliability

export HEALTH_PERSISTENCE_ENABLED=true
export HEALTH_DB_PATH=/data/health/production.db
export HEALTH_FLUSH_INTERVAL=60s    # Standard interval balances performance and data safety
export HEALTH_BATCH_SIZE=100        # Larger batches for better write performance

# Production backup configuration
export HEALTH_BACKUP_ENABLED=true
export HEALTH_BACKUP_DIR=/data/backups/health
export HEALTH_BACKUP_RETENTION_DAYS=30  # Keep monthly backups

# Additional production considerations:
# - Ensure /data/health directory has proper permissions
# - Set up monitoring for disk space in backup directory
# - Consider backup directory on separate disk/mount for safety
```

### Docker Configuration Examples

#### Docker Compose for Development
```yaml
version: '3.8'
services:
  myapp:
    build: .
    environment:
      # Memory-only for development
      HEALTH_PERSISTENCE_ENABLED: "false"
    volumes:
      - ./data:/data  # Optional: for testing persistence
```

#### Docker Compose for Production
```yaml
version: '3.8'
services:
  myapp:
    build: .
    environment:
      HEALTH_PERSISTENCE_ENABLED: "true"
      HEALTH_DB_PATH: "/data/health.db"
      HEALTH_FLUSH_INTERVAL: "60s"
      HEALTH_BATCH_SIZE: "100"
      HEALTH_BACKUP_ENABLED: "true"
      HEALTH_BACKUP_DIR: "/data/backups"
      HEALTH_BACKUP_RETENTION_DAYS: "30"
    volumes:
      - health_data:/data
      - health_backups:/data/backups
    restart: unless-stopped

volumes:
  health_data:
    driver: local
  health_backups:
    driver: local
```

### Kubernetes Configuration Examples

#### Development Namespace
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-dev
spec:
  replicas: 1
  selector:
    matchLabels:
      app: myapp-dev
  template:
    metadata:
      labels:
        app: myapp-dev
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        env:
        # Memory-only for development
        - name: HEALTH_PERSISTENCE_ENABLED
          value: "false"
```

#### Production Namespace
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-prod
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp-prod
  template:
    metadata:
      labels:
        app: myapp-prod
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        env:
        - name: HEALTH_PERSISTENCE_ENABLED
          value: "true"
        - name: HEALTH_DB_PATH
          value: "/data/health.db"
        - name: HEALTH_FLUSH_INTERVAL
          value: "60s"
        - name: HEALTH_BATCH_SIZE
          value: "100"
        - name: HEALTH_BACKUP_ENABLED
          value: "true"
        - name: HEALTH_BACKUP_DIR
          value: "/data/backups"
        - name: HEALTH_BACKUP_RETENTION_DAYS
          value: "30"
        volumeMounts:
        - name: health-data
          mountPath: /data
      volumes:
      - name: health-data
        persistentVolumeClaim:
          claimName: myapp-health-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myapp-health-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

### Common Configuration Mistakes and Solutions

#### Mistake 1: Forgetting CGO for SQLite
```bash
# Wrong: This will fail if SQLite persistence is enabled
CGO_ENABLED=0 go build -o myapp main.go

# Correct: Enable CGO for SQLite support
CGO_ENABLED=1 go build -o myapp main.go

# For cross-compilation (Linux from macOS):
CC="zig cc -target x86_64-linux" CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build
```

#### Mistake 2: Using Relative Paths in Production
```bash
# Wrong: Relative paths can cause issues when working directory changes
export HEALTH_DB_PATH=./data/health.db
export HEALTH_BACKUP_DIR=./backups

# Correct: Always use absolute paths in production
export HEALTH_DB_PATH=/data/health.db
export HEALTH_BACKUP_DIR=/data/backups
```

#### Mistake 3: Invalid Duration Formats
```bash
# Wrong: These will be ignored and default values used
export HEALTH_FLUSH_INTERVAL=60        # Missing unit
export HEALTH_FLUSH_INTERVAL="1 minute" # Invalid format

# Correct: Use Go duration format
export HEALTH_FLUSH_INTERVAL=60s       # 60 seconds
export HEALTH_FLUSH_INTERVAL=1m        # 1 minute
export HEALTH_FLUSH_INTERVAL=30s       # 30 seconds
```

#### Mistake 4: Insufficient Disk Space Planning
```bash
# Wrong: No monitoring of backup directory growth
export HEALTH_BACKUP_RETENTION_DAYS=365  # Too long for most use cases

# Correct: Plan backup retention based on disk space and requirements
export HEALTH_BACKUP_RETENTION_DAYS=30   # Monthly retention (30 days)
# Plus: Set up monitoring for backup directory disk usage
```

#### Mistake 5: Backup Configuration Parsing
```bash
# Wrong: Setting retention days to 0 (causes immediate cleanup)
export HEALTH_BACKUP_RETENTION_DAYS=0    # Backups deleted immediately!

# Correct: Set positive integer for days to retain backups
export HEALTH_BACKUP_RETENTION_DAYS=7    # Keep for 7 days
export HEALTH_BACKUP_RETENTION_DAYS=30   # Keep for 30 days

# Note: Value must be a positive integer (days), not a duration format
```

### Configuration Validation

#### How to Verify Your Configuration
```go
// Add this to your application startup to verify configuration
func main() {
    state := NewState()
    defer state.Close()
    
    state.SetConfig("myapp-production")
    
    // Test basic functionality
    state.IncrMetric("startup_test")
    state.AddMetric("config_test", 123.45)
    
    // Verify JSON output works
    json := state.Dump()
    if json == "" {
        log.Fatal("Health configuration failed - no JSON output")
    }
    
    log.Println("Health package configured successfully")
    log.Printf("Configuration: %s", json)
}
```

#### Environment Variable Debugging
```bash
# Print all health-related environment variables for debugging
env | grep HEALTH

# Expected output for production:
# HEALTH_PERSISTENCE_ENABLED=true
# HEALTH_DB_PATH=/data/health.db
# HEALTH_FLUSH_INTERVAL=60s
# HEALTH_BATCH_SIZE=100
# HEALTH_BACKUP_ENABLED=true
# HEALTH_BACKUP_DIR=/data/backups
# HEALTH_BACKUP_RETENTION_DAYS=30
```

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