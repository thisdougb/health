# Health Package Architecture

## Overview

The health package provides a simple, thread-safe metrics collection system designed for containerized applications. The package is organized by core capabilities:

1. **Data Methods** - Core metrics recording (global and component-based)
2. **Data Access** - Web request handling with flexible URL patterns
3. **Storage Models** - Memory-only or SQLite persistence with background sync
4. **Data Management** - Retention policies, backup integration, and automated cleanup

## Core Components

### 1. Data Methods (Metrics Recording)

#### State Struct (`internal/core/state.go`)

The `StateImpl` struct is the internal implementation for metrics collection, supporting both global and component-based metrics with persistence:

```go
type StateImpl struct {
    Identity    string                    // Instance identifier
    Started     int64                     // Unix timestamp of initialization
    Metrics     map[string]map[string]int // Component-based counter metrics
    persistence *storage.Manager          // Persistence coordination
    mu          sync.Mutex               // Writer lock for thread safety
}
```

#### Metric Recording Methods

**Counter Metrics (stored in memory + persisted):**
```go
func (s *State) IncrMetric(name string)                       // Global counters
func (s *State) IncrComponentMetric(component, name string)   // Component counters
```

**Raw Value Metrics (persisted to storage backend):**
```go
func (s *State) AddMetric(name string, value float64)                    // Global values
func (s *State) AddComponentMetric(component, name string, value float64) // Component values
```

**Key Design Decisions**:
- **Dual storage model**: Counter metrics in memory for real-time, raw values persisted for analysis
- **Component organization**: Metrics organized by application component for complex systems  
- **API design rationale**: Separate methods for different metric types (counters vs raw values)
- **Backward compatibility**: Existing `IncrMetric()` unchanged, new methods are additive
- **Async persistence**: Raw values persisted asynchronously to avoid blocking metric collection
- **Thread-safe operations**: All write operations protected by mutex
- **Zero-value initialization**: Maps are created lazily on first use
- **Client-side aggregation**: Raw values stored in backend for flexible analysis by clients

### 2. Data Access (Web Request Handling)

#### HandleHealthRequest Method

```go
func (s *State) HandleHealthRequest(w http.ResponseWriter, r *http.Request)
```

**URL Pattern Processing:**
- Searches for `/health/` pattern in URL path
- Processes everything after `/health/` regardless of prefix
- Routes to component-specific data or overall health status
- Enables external router compatibility (nginx, Kubernetes ingress)

**Supported URL Patterns:**
- `{prefix}/health/` → All metrics (JSON dump)
- `{prefix}/health/status` → Overall health status (200/503)
- `{prefix}/health/{component}` → Component-specific metrics
- `{prefix}/health/{component}/status` → Component health status

### 3. Storage Models

#### Memory-Only Model (Current)
- Fast performance with no I/O overhead
- Ideal for development (no database setup required)
- Data lost on application restart
- Clean state for testing

#### SQLite Persistence Model (Enhanced)
- Memory-first approach for zero performance impact
- Background Go routine syncs every ~60 seconds
- No blocking I/O on metric recording operations
- Single-file deployment simplicity
- Historical metrics for analysis

### Persistence Layer (`internal/storage/`)

The persistence layer provides pluggable storage backends for historical metric data:

```go
type Backend interface {
    WriteMetrics(metrics []MetricEntry) error
    ReadMetrics(component string, start, end time.Time) ([]MetricEntry, error)
    ListComponents() ([]string, error)
    Close() error
}
```

**Implementations**:
- **Memory Backend**: Fast in-memory storage for testing and development
- **SQLite Backend**: Production-ready persistence with async write queue
- **Async Processing**: Background goroutine batches writes to prevent blocking

## Thread Safety Model

### Global Mutex Protection

```go
var mu sync.Mutex // writer lock
```

**Locking Strategy**:
- **Write operations only**: `IncrMetric()` and `IncrComponentMetric()` use mutex
- **Read operations unprotected**: `Dump()` reads without locking for performance
- **Persistence calls**: Raw value persistence happens outside critical sections
- **Rationale**: Health monitoring scenarios prioritize availability over perfect consistency

### Concurrency Considerations

- **Reader-writer trade-off**: Accepts potential inconsistent reads during writes for better performance
- **Short critical sections**: Minimal work performed while holding locks
- **No deadlock risk**: Single global mutex eliminates complex locking hierarchies

## Data Flow Architecture

### Initialization Flow

1. **Instance Creation**: `State` struct created with persistence manager
2. **Configuration**: `SetConfig(identity)` sets instance parameters
3. **Timestamp Recording**: `Started` field set to current Unix timestamp
4. **Persistence Setup**: Storage backend initialized from environment variables
5. **Default Handling**: Empty identity uses sensible defaults

### Metric Collection Flow

#### Counter Metrics
```
Application Event → IncrMetric(name) → Mutex Lock → Map Update → Mutex Unlock → Async Persist
```

#### Raw Value Metrics
```
Application Event → AddMetric(name, value) → Async Persist (no blocking)
```

### Export Flow

```
HTTP Request → Dump() → JSON Marshal (counter metrics only) → HTTP Response
```

**No Locking**: Export operates without mutex for maximum availability during health checks.
**Counter Metrics Only**: Raw values are not included in JSON output - they're stored in backend for analysis.

## Memory Management

### Lazy Initialization Pattern

Maps are created only when first needed:
- **Metrics map**: Created on first `IncrMetric()` call
- **Component maps**: Created per-component on first metric for that component
- **Benefit**: Zero memory overhead for unused components

### Predictable Memory Usage

- **Counter storage only**: Only counter metrics stored in memory
- **No unbounded growth**: Counter metrics are bounded by application usage
- **Predictable scaling**: Memory usage scales linearly with unique counter metric names
- **Raw values external**: Raw metric values stored in persistence backend, not memory

## JSON Serialization Design

### Output Format

```json
{
    "Identity": "node-ac3e6",
    "Started": 1589113356,
    "Metrics": {
        "Global": {
            "requests": 42
        },
        "webserver": {
            "requests": 100
        }
    }
}
```

### Design Decisions

- **Structured JSON**: Easy parsing by monitoring systems
- **Timestamped**: `Started` field enables uptime calculations
- **Self-describing**: `Identity` provides context for the instance
- **Component organized**: Counter metrics grouped by component for clarity
- **Real-time only**: Shows current counter state, not historical data

## Error Handling Philosophy

### Defensive Programming

- **Invalid inputs ignored**: Empty metric names silently ignored
- **No panics**: All operations designed to be safe
- **Default values**: Invalid configuration uses sensible defaults
- **Fatal on JSON errors**: JSON marshalling errors are fatal (system integrity issue)

### Rationale

Health monitoring must be extremely reliable - better to ignore invalid operations than crash the application.

## Performance Characteristics

### Time Complexity

- **IncrMetric()**: O(1) - simple map increment
- **UpdateRollingMetric()**: O(n) where n = RollingDataSize for average calculation
- **Dump()**: O(m) where m = total number of metrics for JSON marshalling

### Space Complexity

- **Per instance**: O(k + r×n) where k = unique counter metrics, r = unique rolling metrics, n = RollingDataSize
- **Typical usage**: Very low memory footprint for standard containerized applications

## Integration Patterns

### Container Health Endpoints

Typical HTTP handler integration:

```go
var healthState health.State

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, "%s\n", healthState.Dump())
}
```

### Kubernetes Readiness/Liveness

- **Readiness probes**: Include application-specific metrics
- **Liveness probes**: Basic structural health information
- **Monitoring dashboards**: JSON parsing for metrics visualization

## Design Philosophy

### Simplicity Over Features

- **Two metric types only**: Counters and rolling averages cover most use cases
- **No rate calculations**: Consumers handle temporal analysis
- **No persistence**: In-memory only for simplicity
- **No complex statistics**: Focus on basic monitoring needs

### Operational Focus

- **Container-friendly**: Designed for containerized deployment patterns
- **Dashboard integration**: Standard JSON output for easy consumption
- **Auto-discovery**: Identity field enables automatic service discovery
- **Production-ready**: Thread-safe, defensive, and reliable

### API Design Philosophy

**"Expose behavior, hide implementation"** - The package API should expose what users need to accomplish (incrementing metrics, getting health data) while hiding how it's accomplished (storage mechanisms, internal data structures, locking strategies).

#### Public API Principles
- **Behavior-focused methods**: `IncrMetric()`, `UpdateRollingMetric()`, `Dump()`
- **Simple configuration**: Clear initialization with sensible defaults
- **Stable interfaces**: Public API changes require major version bumps
- **Self-documenting**: Method names clearly indicate their purpose

#### Private Implementation Strategy
- **Lowercase naming**: All internal functions and types use lowercase names
- **Internal packages**: Complex subsystems isolated in `internal/` directories
- **Implementation flexibility**: Internal structures can change without breaking users
- **Encapsulated complexity**: Database operations, file I/O, and alert logic remain hidden

#### Go Best Practices for Privacy
```go
// Public API - exposes behavior
func (s *State) IncrMetric(name string)           // ✅ Public behavior
func (s *State) HealthHandler() http.HandlerFunc  // ✅ Public integration

// Private implementation - hides complexity  
func (s *State) persistMetrics() error            // ✅ Private implementation
type alertEngine struct { ... }                   // ✅ Private internal type
var connectionPool *sql.DB                        // ✅ Private shared resource
```

This approach enables the package to evolve internally while maintaining a stable, simple public interface that users can depend on.

## Extensibility Considerations

### Current Limitations

- **In-memory only**: No persistence across restarts
- **No metric deletion**: Metrics accumulate for application lifetime
- **Fixed rolling window**: Cannot change rolling size after initialization
- **Global mutex**: Single lock may become bottleneck under extreme load

### Future Extension Points

- **Metric types**: Additional metric types could be added with similar patterns
- **Persistence**: Storage backends could be added while maintaining JSON compatibility  
- **Configuration**: Runtime configuration changes could be supported
- **Fine-grained locking**: Per-metric locks could improve concurrency

## Testing Architecture

### Thread Safety Testing

Tests must verify concurrent access patterns:
- **Multiple goroutines**: Concurrent increment operations
- **Race condition detection**: Use `go test -race`
- **Consistency validation**: Verify final state after concurrent operations

### Rolling Average Testing

- **Buffer overflow**: Verify circular buffer behavior
- **Average calculation**: Test mathematical correctness
- **Edge cases**: Empty buffers, single values, buffer size boundaries

## Dependencies

### Standard Library Only

- **encoding/json**: JSON serialization
- **log**: Error logging for JSON failures
- **sync**: Mutex for thread safety
- **time**: Timestamp generation

**Rationale**: Minimal dependencies reduce deployment complexity and improve reliability.