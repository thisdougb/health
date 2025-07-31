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

### 4. Data Management
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

# Build (check compilation)
go build

# Get dependencies
go mod tidy

# Format code
go fmt ./...
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
var state health.State
state.Info("my-app", 10)

// Global metrics
state.IncrMetric("requests")
state.UpdateRollingMetric("response_time", 145.2)

// Component-based metrics  
state.IncrComponentMetric("webserver", "requests")
state.IncrComponentMetric("database", "queries")
state.UpdateComponentRollingMetric("cache", "hit_rate", 0.85)

// Export JSON
json := state.Dump()
```

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

### Production with Persistence
```go
// Enable SQLite persistence via environment variable
// HEALTH_PERSISTENCE_ENABLED=true
// HEALTH_DB_PATH="/data/health.db"
// HEALTH_FLUSH_INTERVAL="60s"
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