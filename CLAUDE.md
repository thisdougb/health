# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go package for tracking and reporting metrics in containerized applications. The package provides a simple health monitoring system designed for service architectures, particularly in Kubernetes environments where containers expose `/health` HTTP handlers.

## Documentation

- **`./docs/ARCHITECTURE.md`** - Detailed system architecture and component design
- **`./docs/DECISION_LOG.md`** - Historical record of architectural decisions and strategic changes

## Core Architecture

The package consists of two main components:

1. **State struct** (`health.go`): The main metrics container that holds:
   - Simple counter metrics (incremented via `IncrMetric()`)
   - Rolling average metrics (updated via `UpdateRollingMetric()`)
   - Thread-safe operations using mutex locks around writes

2. **rollingMetric struct** (`rolling_metric.go`): Implements circular buffer logic for calculating rolling averages over a configurable window size

Key design principles:
- Thread-safe metric updates with mutex protection around writes
- JSON serializable output via `Dump()` method
- No rate calculations (consumers handle rates over time)
- Configurable rolling window sizes for average calculations

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

Typical usage involves:
1. Initialize with `Info(identity, rollingDataSize)`
2. Increment counters with `IncrMetric(name)`
3. Update rolling metrics with `UpdateRollingMetric(name, value)`
4. Export JSON with `Dump()` for HTTP health endpoints

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