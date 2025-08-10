# Health Package Decision Log

## Purpose and Maintenance

This document records the reasoning behind significant architectural decisions, feature implementations, and strategic changes made during health package development. It serves as a historical record of what happened and why, capturing both successful decisions and rejected alternatives.

**When to Update**: Add entries for:
- Major architectural changes or refactors
- Significant features implemented or removed
- Technology choices and trade-offs
- Performance optimisations
- Security decisions
- User experience pivots
- Integration decisions (external services, APIs)
- Development process changes

**Entry Format**: Each entry should include the date, a brief context-setting summary, and specific decision points with rationale.

**Chronological Order**: Entries are ordered with most recent first - years in descending order (2025, 2024), and months within each year in descending order (December to January).

**Git History Reference**: Use the last updated timestamp from git log as the baseline when checking git history for changes since this document was last updated.

---

## 2025-08-10: Phase 2 - Move-and-Flush Architecture for Time-Windowed Metrics Implementation

**Decision**: Implemented move-and-flush architecture replacing cumulative counters with time-windowed collection system

**Context**:
- Phase 2 of time-windowed metrics plan required move-and-flush architecture for minimal lock contention
- Need to solve server restart data loss issues and provide richer statistical data
- Must achieve ~99% storage reduction while maintaining sub-microsecond performance
- Required background processing system for automatic time window aggregation
- Time window keys needed human-readable format for debugging and analysis

**Decision**:
- **Move-and-Flush Architecture**: Implemented dual-map system with separate mutexes
  - SampledMetrics (active collection) - protected by collectMutex (RWMutex)
  - FlushQueue (ready for DB write) - protected by flushMutex (Mutex)
  - Minimal lock contention: hot collection path separate from flush operations
- **Time Window Format**: Enhanced time keys to YYYYMMDDHHMMSS with zero-padding
  - 60-second window: 20250810103300 (zeros out seconds)
  - 1-hour window: 20250810100000 (zeros out minutes and seconds)
  - Human-readable and sortable for database queries
- **Statistical Aggregation**: Added calculateStats() function for min/max/avg/count
  - Aggregation happens outside collection locks for non-blocking performance
  - Single aggregated row replaces hundreds of individual metric entries
- **Database Schema**: Created time_series_metrics table with proper indexing
  - PRIMARY KEY (time_window_key, component, metric) for efficient lookups
  - Indexes on component and time_window_key for query performance
- **Backend Support**: Extended both SQLite and Memory backends
  - New WriteTimeSeriesMetrics() methods for aggregated data storage
  - Maintains existing WriteMetrics() for backward compatibility
- **Background Processing**: Added automatic flush goroutine
  - Runs every HEALTH_SAMPLE_RATE seconds (configurable, default 60s)
  - Graceful shutdown with context cancellation and final flush

**Technical Implementation**:
- Comprehensive test suite with 8 new Phase 2 test functions
- Performance validated: maintains sub-microsecond collection times
- Storage efficiency: ~99% reduction (150 rows → 1 aggregated row per time window)
- Thread-safe operations with separate read/write mutexes
- Background processing with proper lifecycle management
- All existing tests continue to pass - full backward compatibility

**Performance Results**:
- Collection operations: <100ns per metric (unchanged from Phase 1)
- Storage reduction: 150 individual entries → 1 aggregated entry per window
- Lock contention minimized: collection and flush operate on separate data structures
- Memory efficiency: completed windows moved to flush queue, then cleared
- Database efficiency: batch writes with proper transaction handling

**Consequences**:
- Solves server restart data loss - all data persisted in time windows
- Provides rich statistical data (min/max/avg/count) for each metric per time window
- Enables efficient historical analysis and trending
- Foundation for advanced monitoring dashboards and alerting
- Maintains API compatibility - no changes to existing metric collection methods
- Supports both development (memory) and production (SQLite) workflows
- Creates foundation for Phase 3 integration and migration features

---

## 2025-07-31: Phase 6 - Event-Driven Backup Integration Implementation

**Decision**: Implemented event-driven backup system following known patterns with State lifecycle integration

**Context**:
- Phase 6 of persistent storage plan required backup functionality following established patterns
- Need event-driven backups (not scheduled) triggered by application lifecycle events
- Must avoid SQLite file locking issues by using existing database connection
- Required seamless integration with existing State system without breaking API compatibility
- Backup functionality should be optional and configurable via environment variables

**Decision**:
- **Event-Driven Architecture**: Following known patterns with backups triggered by:
  - State shutdown (Close() method) - automatic backup on graceful shutdown
  - Manual backup creation via Manager.CreateBackup() method
  - No scheduled/timer-based backups (event-driven only)
- **SQLite VACUUM INTO Implementation**: Using atomic backup creation with same database connection
  - Avoids file locking issues by cascading DB connection through backend
  - SQLiteBackend.CreateBackup() method uses existing connection
  - Memory backends skip backup (temporary storage, no persistence needed)
- **Configuration Integration**: Extended existing Config system with BackupConfig
  - HEALTH_BACKUP_ENABLED - Enable/disable backup functionality
  - HEALTH_BACKUP_DIR - Backup directory path (/data/backups/health default)  
  - HEALTH_BACKUP_RETENTION_DAYS - Retention policy (30 days default)
  - HEALTH_BACKUP_INTERVAL - Not used (event-driven, not scheduled)
- **Manager Integration**: Storage Manager coordinates backup operations
  - Manager.CreateBackup() - Manual backup creation
  - Manager.ListBackups() - List available backup files
  - Manager.RestoreFromBackup() - Database restoration capability
  - Manager.GetBackupInfo() - Configuration introspection

**Technical Implementation**:
- Following known backup.go patterns with dated backup files (health_YYYYMMDD.db)
- Comprehensive test suite with 13 test functions covering all scenarios
- Event-driven backup on State.Close() - automatic backup during graceful shutdown
- Backup directory creation with proper permissions (0755)
- Retention policy cleanup with configurable retention days
- Error handling with graceful degradation (backup failures don't break shutdown)
- Zero API changes - existing State.Close() unchanged, backup happens transparently

**Performance Results**:
- Event-driven backup creation: ~110-120ms for typical database sizes
- No performance impact on normal operations (only during backup events)
- All integration tests passing with SQLite and memory backends
- Backup operations are atomic using SQLite VACUUM INTO
- File locking issues resolved through proper connection management

**Consequences**:
- Production-ready backup system following established patterns
- Event-driven architecture aligns with application lifecycle management
- Configurable backup retention prevents disk space issues
- Zero breaking changes - full backward compatibility maintained
- Foundation for operational monitoring and data recovery workflows
- Seamless integration with existing SQLite persistence backend

---

## 2025-07-31: Phase 5 - Claude Data Extraction Functions Implementation

**Decision**: Implemented specialized administrative functions for extracting historical metrics data optimized for Claude analysis

**Context**:
- Phase 5 of persistent storage plan required data extraction functions for AI/Claude processing
- Need efficient access to historical metrics with time-based filtering
- Required component-based organization and statistical aggregation
- Must maintain excellent performance for interactive analysis workflows
- JSON output must be optimized for programmatic consumption and AI interpretation

**Decision**:
- **Admin Functions Implementation**: Created `internal/handlers/admin.go` with four core functions:
  - `ExtractMetricsByTimeRange()` - Component-specific metric extraction with time filtering
  - `ExportAllMetrics()` - Complete data export in structured JSON format
  - `ListAvailableComponents()` - Component discovery for filtering operations
  - `GetHealthSummary()` - Statistical health analysis with aggregation
- **Interface Design**: Created `AdminInterface` abstraction requiring `GetStorageManager()` method
  - Enables dependency injection and testing
  - StateImpl implements interface through new `GetStorageManager()` method
  - Clean separation between admin operations and core metrics functionality
- **JSON Output Optimization**: Designed Claude-friendly output structure
  - Component-organized data for easy programmatic access
  - Time-series data with proper chronological ordering
  - Statistical aggregation (min/max/avg) for value metrics
  - Counter summaries with totals and occurrence counts
  - System health indicators based on resource usage patterns
  - Pretty-printed JSON with consistent indentation

**Technical Implementation**:
- Comprehensive test suite with 9 test functions including benchmarks
- Performance optimized: ExtractMetricsByTimeRange ~706 ns/op, GetHealthSummary ~9.5 μs/op
- Graceful error handling when persistence is disabled
- Type-safe value conversion supporting multiple numeric types
- Memory-efficient operations with zero allocations for core functions

**Performance Results**:
- Sub-microsecond response times for component extraction
- Microsecond-level performance for complex statistical aggregation
- Zero memory growth validated through automated testing
- All tests passing with comprehensive edge case coverage

**Consequences**:
- Enables efficient Claude analysis of historical metrics data
- Provides foundation for advanced health monitoring and alerting
- Maintains backward compatibility with existing State interface
- Supports both development (memory backend) and production (SQLite) workflows
- Facilitates programmatic health analysis and reporting workflows

---

## 2025-07-31: Phase 4 - Background System Metrics Collection Implementation

**Decision**: Implemented automatic system metrics collection with always-on background monitoring

**Context**:
- Phase 4 of persistent storage plan required automatic system metrics collection
- Need operational visibility into application resource usage and health
- Must maintain zero performance impact on core metric operations
- System metrics should complement existing application metrics

**Decision**:
- **SystemCollector Implementation**: Created `internal/metrics/system.go` with background collection
  - CPU utilization percentage (simplified estimation based on GC activity)
  - Memory allocation in bytes (runtime.MemStats)
  - Health data size estimation for memory usage monitoring
  - Goroutine count for concurrency monitoring
  - Application uptime in seconds for operational tracking
- **Always-On Architecture**: System metrics automatically enabled with every State instance
  - Background goroutine runs every minute (configurable interval)
  - Graceful shutdown when State.Close() is called
  - Non-blocking design with zero impact on application performance
- **Storage Integration**: System metrics stored as raw values in persistence backend
  - Not included in JSON output (counters only in memory)
  - Full historical data available through storage queries
  - Organized under "system" component for easy filtering

**Technical Implementation**:
- Background collection using context.Context for cancellation
- StateInterface abstraction for dependency injection and testing
- Comprehensive test suite with concurrent access testing
- Performance benchmarks validating <100ns per core operation
- Memory usage validation preventing leaks

**Performance Results**:
- Core operations maintain sub-microsecond performance:
  - IncrMetric: ~50-76 ns/op
  - AddMetric: ~16-18 ns/op
  - Dump: ~1500-2600 ns/op
- System metrics collection: ~100ms per collection cycle
- Zero memory growth verified through automated testing

**Consequences**:
- Automatic operational monitoring without configuration
- Historical system metrics available for analysis and alerting
- Foundation for advanced system health monitoring
- Maintains backward compatibility and performance characteristics
- Enables proactive monitoring of application resource usage

---

## 2025-07-31: Component-Based Metrics Architecture

**Decision**: Introduced component-based metrics API alongside existing global metrics

**Context**:
- Health package needed better organization for complex applications with multiple components
- External router compatibility (nginx, Kubernetes ingress) required flexible URL patterns
- Go's variadic parameter limitations prevented clean mixed parameter APIs
- Need to maintain backward compatibility with existing `IncrMetric()` usage

**Decision**:
- **API Design**: Added separate methods for component-based metrics:
  - `IncrComponentMetric(component, name string)` for component counters
  - `UpdateComponentRollingMetric(component, name string, value float64)` for component averages
- **URL Pattern Support**: Implemented `HandleHealthRequest()` with flexible routing:
  - `{prefix}/health/` → All metrics (any prefix supported)
  - `{prefix}/health/{component}` → Component-specific metrics
  - `{prefix}/health/{component}/status` → Component health status
- **Storage Models**: Defined three approaches:
  - Memory-only (current) - fast, ideal for development
  - SQLite persistence - background sync every ~60 seconds, zero performance impact
  - Configuration via environment variables for deployment flexibility
- **Data Management**: Added retention policies, backup integration, automated cleanup

**Technical Implementation**:
- Separate methods vs variadic parameters due to Go limitations
- External router compatibility for microservice architectures  
- Memory-first approach with optional background persistence
- Component organization in JSON output structure

**Consequences**:
- Enables component-based organization for complex systems
- Maintains backward compatibility (existing code unchanged)
- Supports modern microservice deployment patterns
- Zero performance impact with memory-first design
- Clear development-to-production workflow (env var configuration)

---

## 2025-07-30: Project Documentation Structure

**Decision**: Created docs directory and DECISION_LOG.md to track architectural decisions

**Context**: 
- Need structured documentation approach following known patterns
- Important to document the "why" behind implementation decisions
- CLAUDE.md was becoming the single source of truth but needed separation of concerns

**Decision**: 
- Created `docs/` directory for project documentation
- Added `DECISION_LOG.md` to track decision rationale over time
- CLAUDE.md remains focused on development workflow and package architecture
- Documentation references added to CLAUDE.md for discoverability

**Consequences**:
- Future architectural decisions will be documented with context and rationale
- Clearer separation between development guidance (CLAUDE.md) and decision history (docs/)
- Follows established patterns from known project structure

---

## 2025-07-30: Dynamic Git Attribution in Commits

**Decision**: Use dynamic git config values for commit co-authorship attribution

**Context**:
- Far Oeuf projects use mandatory collaborative attribution in all commits
- Previously hardcoded "Doug" in commit templates
- Need flexible attribution that works across different developers

**Decision**:
- Updated commit standards to use `$(git config --get user.name)` and `$(git config --get user.email)`
- Maintains collaborative attribution while being developer-agnostic
- Preserves Claude co-authorship requirement

**Consequences**:
- Commit attribution automatically adapts to different git configurations
- Maintains consistency with collaborative development patterns
- Reduces manual maintenance of commit templates

---

## 2025-07-30: CLAUDE.md Creation and Workflow Integration

**Decision**: Created comprehensive CLAUDE.md with established workflow patterns

**Context**:
- Need guidance file for future Claude Code instances working on this repository
- Health package needed consistent development patterns and workflows
- Required clear architectural documentation and development standards

**Decision**:
- Created CLAUDE.md with project overview, architecture, and development commands
- Adopted branch management patterns (`{component}_{description}`)
- Integrated test-first development approach
- Added comprehensive testing requirements
- Included collaborative commit attribution standards

**Consequences**:
- Future development will follow consistent patterns
- Clear guidance for Claude Code instances on package architecture
- Established testing and workflow standards for the health package

---

## 2020-05-16: Go Report Card Integration

**Decision**: Added Go Report Card badge to README for code quality visibility

**Context**:
- Package reached stable state with comprehensive tests and documentation
- Need external validation of code quality standards
- Go Report Card provides automated analysis of Go code quality

**Decision**:
- Added Go Report Card badge to README.md
- Fixed identified typos and formatting issues
- Integrated quality reporting into project presentation

**Consequences**:
- External code quality validation visible to users
- Continuous quality monitoring and improvement feedback
- Professional appearance for open source package

---

## 2020-05-11: Thread Safety Implementation

**Decision**: Added mutex protection around all write operations to metrics

**Context**:
- Package designed for use in concurrent environments (web servers, containers)
- Race conditions possible with multiple goroutines updating metrics simultaneously
- Need thread-safe operations without compromising performance

**Decision**:
- Implemented global mutex (`var mu sync.Mutex`) for write protection
- Protected `IncrMetric()` and `UpdateRollingMetric()` operations
- Left `Dump()` unprotected for maximum availability during health checks
- Used short critical sections to minimize lock contention

**Consequences**:
- Thread-safe metric updates in concurrent environments
- Maintains high availability for health check endpoints
- Established clear concurrency model for the package

---

## 2020-05-10: CI/CD Integration

**Decision**: Implemented CircleCI for continuous integration and testing

**Context**:
- Need automated testing for pull requests and commits
- Want consistent test execution across different environments
- Public visibility of build status important for open source adoption

**Decision**:
- Added CircleCI configuration with Go testing pipeline
- Integrated status badge in README for build visibility
- Established automated testing workflow for contributions

**Consequences**:
- Automated quality assurance for all code changes
- Public confidence through visible build status
- Foundation for future deployment automation

---

## 2020-05-10: Project Foundation and Architecture

**Decision**: Initial implementation of health metrics package with dual metric types

**Context**:
- Need simple, reliable health monitoring for containerized Go applications
- Existing solutions either too complex or lacking key features
- Focus on JSON output for modern monitoring dashboards

**Decision**:
- **Core Architecture**: State struct with simple counters and rolling averages
- **Thread Model**: Single global mutex for write operations, unprotected reads
- **Data Structures**: Maps for counters, circular buffers for rolling metrics
- **Output Format**: JSON serialization for HTTP health endpoints
- **Testing Strategy**: Comprehensive unit tests with race condition testing

**Key Implementation Choices**:
- **Simplicity First**: Two metric types cover majority of use cases
- **Container Focus**: Designed for Kubernetes health check patterns
- **Standard Library Only**: No external dependencies for reliability
- **Defensive Programming**: Invalid inputs ignored rather than causing errors

**Initial File Structure**:
- `health.go` (98 lines): Core State struct and metric operations
- `rolling_metric.go` (26 lines): Circular buffer implementation
- `health_test.go` (128 lines): Comprehensive unit tests
- `rolling_metric_test.go` (37 lines): Rolling average tests
- `doc.go` (54 lines): Package documentation and usage examples

**Consequences**:
- Simple, focused package suitable for immediate production use
- Clear architectural foundation for future enhancements
- Comprehensive test coverage from initial implementation
- Standard Go conventions and idioms throughout
