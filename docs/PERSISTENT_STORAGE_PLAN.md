# Persistent Storage Implementation Plan

## Overview

This document outlines the phased implementation of persistent storage for the health metrics package. The implementation follows a layered architecture designed to provide eventual consistency without blocking metric collection operations.

## Goals

- **Non-blocking metric collection**: Application code paths must never be blocked by persistence operations
- **Eventual consistency**: Metrics persist to storage asynchronously with acceptable data loss window
- **Component-based organization**: Maintain existing component/metric structure in persistent storage
- **Claude integration**: Enable time-based metric extraction for analysis
- **Production readiness**: Include backup, retention, and system monitoring capabilities

## Architecture Overview

### 3-Layer Design

**Layer 1 (Public API)**: Application interface - existing methods remain unchanged
**Layer 2 (Business Logic)**: Bulk operations, queuing, and data extraction logic
**Layer 3 (Storage Backends)**: SQLite (production) and Memory (development/testing) implementations

### Data Flow

```
App Code → IncrMetric() → Memory Update → Async Queue → Background Sync → SQLite
                     ↓
Claude ← Extract Functions ← Query Layer ← SQLite Storage
```

## Implementation Phases

### Phase 1: Foundation - Storage Interface & Memory Backend

**Goal**: Establish storage interface and in-memory implementation for testing

**Files to Create**:
- `/internal/storage/interface.go` - Storage backend interface
- `/internal/storage/memory.go` - In-memory storage implementation  
- `/internal/storage/memory_test.go` - Unit tests

**Interface Design**:
```go
type Backend interface {
    WriteMetrics(metrics []MetricEntry) error
    ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error)
    ListComponents() ([]string, error)
    Close() error
}

type MetricEntry struct {
    Timestamp time.Time
    Component string
    Name      string
    Value     interface{} // int for counters, float64 for rolling
    Type      string      // "counter" or "rolling"
}
```

**Success Criteria**:
- Interface defines all required operations
- Memory backend passes all unit tests
- Consistent behavior with existing in-memory metrics

### Phase 2: SQLite Backend & Async Processing

**Goal**: Implement production SQLite backend with async write queue

**Files to Create**:
- `/internal/storage/sqlite.go` - SQLite storage implementation
- `/internal/storage/sqlite_test.go` - SQLite-specific tests
- `/internal/storage/queue.go` - Async write queue management

**Database Schema**:
```sql
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    component TEXT NOT NULL,
    name TEXT NOT NULL,
    value REAL NOT NULL,
    type TEXT NOT NULL,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX idx_metrics_time_component ON metrics(timestamp, component);
CREATE INDEX idx_metrics_component_name ON metrics(component, name);
```

**Key Features**:
- Async write queue with configurable batch size
- Background goroutine for periodic flushes
- Connection pooling and error recovery
- Environment variable configuration

**Success Criteria**:
- SQLite backend implements full interface
- Async writes don't block application threads
- Data persists correctly across application restarts
- Performance benchmarks meet requirements

### Phase 3: Integration with Existing State System

**Goal**: Integrate persistence manager with current State implementation

**Files to Modify**:
- `/internal/core/state.go` - Add persistence manager
- `/api.go` - Expose any new public methods if needed

**Files to Create**:
- `/internal/storage/manager.go` - Persistence coordination
- `/internal/storage/config.go` - Configuration management

**Integration Points**:
- `IncrMetric()` and `IncrComponentMetric()` trigger async writes
- `UpdateRollingMetric()` methods include persistence
- Configuration determines storage backend (memory vs SQLite)
- Graceful shutdown ensures data flush

**Configuration Variables**:
```bash
HEALTH_PERSISTENCE_ENABLED=true
HEALTH_DB_PATH="/data/health.db"
HEALTH_FLUSH_INTERVAL="60s"
HEALTH_BATCH_SIZE="100"
```

**Success Criteria**:
- Existing API unchanged - backward compatibility maintained
- Metrics persist without performance degradation
- Configuration enables/disables persistence cleanly
- Integration tests pass with both backends

### Phase 4: Background System Metrics Collection ✅ COMPLETED

**Goal**: Automatic system metrics collection and persistence

**Files to Create**:
- `/internal/metrics/system.go` - System metrics collector
- `/internal/metrics/system_test.go` - System metrics tests

**System Metrics**:
```go
// Collected every minute under "system" component
"cpu_percent"        // float64 - CPU utilization percentage  
"memory_bytes"       // int - Current memory usage
"health_data_size"   // int - Size of in-memory metrics data
"goroutines"         // int - Number of active goroutines
"uptime_seconds"     // int - Application uptime
```

**Implementation**:
- Background goroutine runs every minute
- Uses system calls for CPU/memory info
- Stores metrics using existing component-based API
- Configurable collection interval

**Success Criteria**: ✅ ALL COMPLETED
- ✅ System metrics collect automatically (always-on background collection)
- ✅ Data appears in persistent storage (system metrics stored as raw values)
- ✅ Minimal performance impact on application (<100ns per operation maintained)
- ✅ Metrics useful for operational monitoring (CPU, memory, goroutines, uptime, health data size)

**Implementation Results**:
- SystemCollector automatically starts with every State instance
- 5 system metrics collected every minute: cpu_percent, memory_bytes, health_data_size, goroutines, uptime_seconds
- Zero performance impact verified through benchmarks
- Comprehensive test suite with 100% coverage
- Integration tests validate persistence to SQLite backend
- Background collection using goroutines with graceful shutdown

### Phase 5: Claude Data Extraction Functions

**Goal**: Enable time-based metric extraction for Claude analysis

**Files to Create**:
- `/internal/handlers/admin.go` - Data extraction functions
- `/internal/handlers/admin_test.go` - Extraction function tests

**Public Functions**:
```go
// Extract metrics for specific component and time range
func ExtractMetricsByTimeRange(component string, start, end time.Time) (string, error)

// Export all metrics within time range in specified format
func ExportAllMetrics(start, end time.Time, format string) (string, error)

// List all available components for filtering
func ListAvailableComponents() ([]string, error)

// Get system health summary for time period
func GetHealthSummary(start, end time.Time) (string, error)
```

**Response Formats**:
- JSON output compatible with Claude processing
- Component-organized structure for easy parsing
- Time-series data with proper timestamps
- Aggregation options (min, max, avg for rolling metrics)

**Success Criteria**:
- Functions return properly formatted JSON
- Time range filtering works correctly
- Component filtering isolates relevant data
- Performance acceptable for typical time ranges
- Data format optimized for Claude analysis

### Phase 6: Event-Driven Backup Integration

**Goal**: Implement backup system following established patterns

**Files to Create**:
- `/internal/storage/backup.go` - Backup functionality following tripkist pattern
- `/internal/storage/backup_test.go` - Backup tests

**Backup Features**:
```go
// Create backup of health database
func BackupHealthDatabase(dbPath string) error

// Cleanup old backups based on retention policy
func CleanupHealthBackups() error

// List available backup files
func ListHealthBackups() ([]string, error)

// Restore from specific backup
func RestoreHealthDatabase(backupFile string) error
```

**Backup Strategy**:
- Uses SQLite `VACUUM INTO` for atomic backups
- Daily backups with configurable retention
- Event-driven triggers (startup, shutdown, periodic)
- Follows same patterns as tripkist/sqlitedb

**Configuration**:
```bash
HEALTH_BACKUP_ENABLED=true
HEALTH_BACKUP_DIR="/data/backups/health"
HEALTH_BACKUP_RETENTION_DAYS="30"
HEALTH_BACKUP_INTERVAL="24h"
```

**Success Criteria**:
- Backups created without service interruption
- Restore functionality verified
- Retention policy removes old backups
- Integration with existing backup infrastructure

### Phase 7: Testing & Documentation

**Goal**: Comprehensive testing and documentation completion

**Testing Requirements**:
- Unit tests for all new components (>90% coverage)
- Integration tests for full workflow
- Performance benchmarks for persistence operations
- Race condition testing with `-race` flag
- Error condition and recovery testing

**Documentation Updates**:
- Update `/docs/ARCHITECTURE.md` with persistence details
- Add configuration examples to CLAUDE.md
- Create operational runbook for backup/restore
- Performance characteristics documentation

**Files to Create/Update**:
- Integration test suite
- Benchmark tests
- Updated architecture documentation
- Configuration reference guide

**Success Criteria**:
- All tests pass consistently
- Performance meets requirements
- Documentation covers all new features
- Ready for production deployment

## Implementation Guidelines

### Development Principles

1. **Backward Compatibility**: Never break existing API
2. **Graceful Degradation**: System works with persistence disabled
3. **Error Isolation**: Persistence failures don't affect metric collection
4. **Configuration Driven**: Environment variables control behavior
5. **Test Coverage**: Each phase includes comprehensive tests

### Performance Requirements

- Metric collection latency: <1ms additional overhead
- Background sync: Complete within flush interval
- Memory usage: <10MB additional for persistence layer
- Database operations: <100ms for typical queries
- Startup time: <5s additional for persistence initialization

### Error Handling Strategy

- **Write Failures**: Log errors, continue operation, retry logic
- **Read Failures**: Return cached data or error, don't crash
- **Database Corruption**: Automatic recovery, backup restoration
- **Configuration Errors**: Use defaults, log warnings

## Risk Mitigation

### Potential Issues

1. **Performance Impact**: Mitigated by async design and benchmarks
2. **Data Loss**: Minimized by frequent flushes and graceful shutdown
3. **Database Lock Contention**: Avoided by read/write separation
4. **Disk Space**: Managed by retention policies and cleanup
5. **Migration Complexity**: Phased approach allows incremental testing

### Rollback Strategy

Each phase can be individually disabled via configuration, allowing rollback to previous functionality if issues arise.

## Timeline Estimates

- **Phase 1**: 2-3 days (foundation work)
- **Phase 2**: 3-4 days (SQLite complexity)  
- **Phase 3**: 2-3 days (integration testing)
- **Phase 4**: 1-2 days (system metrics)
- **Phase 5**: 2-3 days (extraction functions)
- **Phase 6**: 2-3 days (backup implementation)
- **Phase 7**: 3-4 days (testing and documentation)

**Total**: 15-22 days for complete implementation

## Success Metrics

- Zero performance regression in metric collection
- <5 second data loss window on application restart
- 100% API compatibility maintained
- All tests pass with race detection enabled
- Documentation complete and accurate
- Production deployment successful