# Health Package Enhancement Plan

**Last Updated:** January 15, 2025  
**Target:** Enhanced monitoring capabilities with Claude-based analysis  
**Repository:** https://github.com/thisdougb/health

## Executive Summary

This document outlines the transition of the health package from a good idea to a production-grade metrics application. The current package provides basic functionality, but production requirements demand rigorous engineering practice.

The enhanced package will serve two primary use cases:

1. **Real-time monitoring endpoints** for external monitoring services to detect system issues
2. **Claude-optimized analysis endpoints** providing rich contextual data for AI-powered interpretation

This transition maintains backward compatibility while adding SQLite persistence, event-driven backups, configurable alerting, and component-specific health monitoring - transforming a simple library into a robust production monitoring solution.

---

## Current Health Package Analysis

### Existing Strengths
- ✅ Simple `IncrMetric()` API for counter metrics
- ✅ Thread-safe operations via mutex protection
- ✅ JSON export via `Dump()` method
- ✅ Lightweight design with no external dependencies
- ✅ HTTP `/health` endpoint for basic health checks

### Enhancement Requirements
- **Persistent Storage**: SQLite backend for historical metrics
- **Real-time Monitoring**: Endpoints for external monitoring services
- **Alert Management**: Configurable thresholds and alert states
- **Component Health**: Individual component monitoring (database, providers, etc.)
- **Claude Integration**: Rich contextual data for AI analysis
- **Backup Integration**: Event-driven backups for data persistence

---

## Feature Specifications

### 1. Real-Time Monitoring Endpoints

#### Core Health Endpoints
```
GET /health/service/status          - Simple UP/DOWN (200/503 responses)
GET /health/alerts/status          - Current alert level: ok|warning|critical  
GET /health/alerts/summary         - Structured alert data for monitoring services
GET /health/alerts/components      - Per-component health status
```

#### Component-Specific Health Checks
```
GET /health/database/status        - Database operations health
GET /health/backup/status          - Backup system health
GET /health/performance/status     - Performance-based alerts
GET /health/errors/status          - Error rate monitoring
GET /health/capacity/status        - System capacity monitoring
```

#### Alert Data Structure
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "alert_level": "warning|critical|ok",
  "alerts": [
    {
      "component": "database",
      "severity": "warning", 
      "reason": "slow query performance",
      "metric": "query_duration_ms",
      "current_value": 1500,
      "threshold": 1000,
      "duration_minutes": 15,
      "first_detected": "2025-01-15T10:15:00Z"
    }
  ],
  "healthy_components": ["backup", "performance", "errors"],
  "system_status": "degraded"
}
```

### 2. Claude-Optimized Analysis Endpoints

#### Claude-Specific Endpoints
```
GET /health/claude/analyze                    - Full system analysis
GET /health/claude/diagnose/{component}       - Component-specific diagnosis  
GET /health/claude/recommendations           - Suggested actions based on alerts
GET /health/claude/incident-report           - Detailed incident context
GET /health/claude/trends?period=24h         - Trend analysis data
GET /health/claude/compare?baseline=7d&current=24h - Comparative analysis
```

#### Claude Context Data Structure
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "service_health": {
    "status": "healthy|degraded|critical",
    "issues": ["high error rate detected", "backup failures detected"],
    "backup_status": {
      "last_backup": "2025-01-15T10:00:00Z",
      "backup_success_rate": 98.5,
      "recovery_tested": "2025-01-14T00:00:00Z"
    },
    "key_metrics": {
      "active_connections": 45,
      "error_rate_percent": 2.1,
      "avg_response_time_ms": 150
    }
  },
  "business_metrics": {
    "daily_signups": 12,
    "daily_active_users": 34,
    "processed_requests": 89
  },
  "current_alerts": [...],
  "recent_changes": [
    "Large data batch processed 15 minutes ago",
    "New user signup spike detected",
    "External service experienced connection issues"
  ],
  "system_context": {
    "active_users": 45,
    "recent_request_volume": 1250,
    "service_health": {"database": "ok", "backup": "ok", "external": "degraded"},
    "time_of_day": "peak_usage"
  },
  "suggested_actions": [
    "Monitor external service connections",
    "Check if email service is experiencing delivery delays",
    "Consider scaling processing resources"
  ]
}
```

### 3. SQLite Storage with Event-Driven Backups

#### Database Schema
```sql
-- Core metrics storage
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_name TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value INTEGER NOT NULL,
    metric_type TEXT NOT NULL DEFAULT 'counter', -- counter, gauge, timer
    component TEXT,                              -- tripengine, providers, etc.
    context TEXT,                               -- JSON context data
    timestamp INTEGER NOT NULL,
    UNIQUE(node_name, metric_name, timestamp)
);

-- Alert state management  
CREATE TABLE alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    component TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    severity TEXT NOT NULL,                     -- warning, critical
    reason TEXT NOT NULL,
    current_value REAL NOT NULL,
    threshold_value REAL NOT NULL,
    first_detected INTEGER NOT NULL,
    last_updated INTEGER NOT NULL,
    resolved_at INTEGER,                        -- NULL if active
    status TEXT NOT NULL DEFAULT 'active'      -- active, resolved
);

-- Schema version tracking
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

-- Indexes for performance
CREATE INDEX idx_metrics_node_time ON metrics(node_name, timestamp);
CREATE INDEX idx_metrics_component ON metrics(component, timestamp);
CREATE INDEX idx_alerts_active ON alerts(status, component) WHERE status = 'active';
```

#### Backup Integration Functions
```go
// New functions following established patterns
func InitHealthWithBackup(dbPath string, backupConfig BackupConfig) error
func PersistMetricsWithBackup() error
func BackupHealthDatabase() error
func RestoreHealthFromBackup(date string) error
func CleanupHealthBackups() error
func GetBackupStatus() BackupStatus
```

#### Event-Driven Backup Triggers
- **Daily metric rollover**: Backup after daily aggregation
- **Alert state changes**: Backup after critical alerts resolved
- **Bulk metric ingestion**: Backup after large metric batches
- **Configuration changes**: Backup after threshold updates

### 4. Alert Management System

#### Configurable Alert Thresholds
```go
type AlertThresholds struct {
    // Performance alerts  
    DayProcessingMaxMs         int     `default:"3000"`
    APIResponseTimeMaxMs       int     `default:"1000"`
    DatabaseQueryMaxMs         int     `default:"500"`
    BackupDurationMaxMs        int     `default:"30000"`
    
    // Error rate alerts (percentages)
    EmailFailureRateMax        float64 `default:"5.0"`
    TripDetectionErrorMax      float64 `default:"2.0"`
    ProviderErrorRateMax       float64 `default:"10.0"`
    BackupFailureRateMax       float64 `default:"1.0"`
    
    // Capacity alerts
    ActiveUsersMax             int     `default:"1000"`
    ConcurrentRequestsMax      int     `default:"100"`
    DatabaseConnectionsMax     int     `default:"50"`
    
    // External service alerts
    MapboxRateLimitBuffer      int     `default:"1000"`  // API calls remaining
    StripeWebhookDelayMaxMin   int     `default:"15"`    // Processing delay
    SendGridQuotaBufferPct     float64 `default:"10.0"`  // Quota remaining %
}
```

#### Alert State Management
```go
type AlertState struct {
    Component      string    `json:"component"`
    MetricName     string    `json:"metric_name"`
    Severity       string    `json:"severity"`      // warning, critical
    Reason         string    `json:"reason"`
    CurrentValue   float64   `json:"current_value"`
    ThresholdValue float64   `json:"threshold_value"`
    FirstDetected  time.Time `json:"first_detected"`
    LastUpdated    time.Time `json:"last_updated"`
    DurationMinutes int      `json:"duration_minutes"`
    Status         string    `json:"status"`        // active, resolved
}

// Alert management functions
func TriggerAlert(component, severity, reason string, currentValue, threshold float64)
func ResolveAlert(component, metricName string)
func EscalateAlert(alertID int, newSeverity string)
func GetActiveAlerts() []AlertState
func GetAlertHistory(component string, since time.Time) []AlertState
```

### 5. Enhanced Metric Types

#### Metric Type Definitions
```go
type MetricType string

const (
    MetricTypeCounter MetricType = "counter"  // Existing - monotonic counters
    MetricTypeGauge   MetricType = "gauge"    // Current values (memory, connections)
    MetricTypeTimer   MetricType = "timer"    // Duration measurements
    MetricTypeRate    MetricType = "rate"     // Calculated rates (per second, per minute)
)

type Metric struct {
    Name      string                 `json:"name"`
    Type      MetricType            `json:"type"`
    Value     float64               `json:"value"`
    Component string                `json:"component"`
    Context   map[string]interface{} `json:"context,omitempty"`
    Timestamp time.Time             `json:"timestamp"`
}
```

#### Enhanced API Methods
```go
// Existing method (maintain compatibility)
func (s *State) IncrMetric(name string)

// New enhanced methods
func (s *State) IncrMetricWithContext(name, component string, context map[string]interface{})
func (s *State) SetGauge(name string, value float64, component string)
func (s *State) RecordTimer(name string, duration time.Duration, component string)
func (s *State) UpdateRate(name string, count int, component string)

// Bulk operations
func (s *State) RecordMetrics(metrics []Metric)
func (s *State) FlushMetrics() error

// Query methods
func (s *State) GetMetric(name string, timeRange string) ([]Metric, error)
func (s *State) GetComponentMetrics(component string, since time.Time) ([]Metric, error)
func (s *State) GetMetricSummary(timeRange string) (MetricSummary, error)
```

---

## Implementation Roadmap

### Phase 1: Core Infrastructure (Week 1)
**Priority: P0 - Critical for Launch**

1. **SQLite Integration** 
   - [ ] Database schema implementation
   - [ ] Migration system for schema updates
   - [ ] Basic CRUD operations for metrics
   - [ ] Connection management and error handling

2. **Enhanced Metric Types**
   - [ ] Extend State struct for new metric types
   - [ ] Implement gauge, timer, rate metric methods
   - [ ] Maintain backward compatibility with existing `IncrMetric()`
   - [ ] Add component and context support

3. **Basic Persistence**
   - [ ] Background persistence goroutine
   - [ ] Configurable flush intervals
   - [ ] Batch metric insertions for performance
   - [ ] Error handling and retry logic

4. **Core Health Endpoints**
   - [ ] `/health/service/status` - Simple UP/DOWN
   - [ ] `/health/alerts/status` - Current alert level
   - [ ] Basic HTTP status code responses (200/503)
   - [ ] JSON response formatting

### Phase 2: Alert System and Component Health (Week 2)  
**Priority: P0 - Critical for Launch**

5. **Alert Management**
   - [ ] Alert threshold configuration system
   - [ ] Alert state tracking in SQLite
   - [ ] Alert triggering and resolution logic
   - [ ] Alert escalation based on duration

6. **Component Health Checks**
   - [ ] `/health/tripengine/status` - Trip processing health
   - [ ] `/health/providers/status` - Provider integration health
   - [ ] `/health/external/status` - External services health
   - [ ] `/health/database/status` - Database operations health

7. **Real-time Monitoring**
   - [ ] Structured alert data endpoints
   - [ ] Component-specific health status
   - [ ] Performance threshold monitoring
   - [ ] Error rate calculations and alerts

8. **Configuration Management**
   - [ ] Environment variable configuration
   - [ ] Runtime threshold updates
   - [ ] Component enable/disable flags
   - [ ] Alert sensitivity levels

### Phase 3: Claude Integration and Advanced Features (Week 3)
**Priority: P1 - Important for Operations**

9. **Claude-Optimized Endpoints**
   - [ ] `/health/claude/analyze` - Full system analysis
   - [ ] `/health/claude/diagnose/{component}` - Component diagnosis
   - [ ] `/health/claude/recommendations` - Suggested actions
   - [ ] Rich contextual data structures

10. **Backup Integration**
    - [ ] Event-driven backup triggers
    - [ ] Backup status monitoring
    - [ ] Recovery capability testing
    - [ ] Integration with existing backup patterns

11. **Advanced Analytics**
    - [ ] Trend analysis over time periods
    - [ ] Comparative analysis (baseline vs current)
    - [ ] Anomaly detection algorithms
    - [ ] Historical context for alerts

12. **Documentation and Testing**
    - [ ] Comprehensive API documentation
    - [ ] Integration examples
    - [ ] Unit test coverage
    - [ ] Performance benchmarking

---

## API Specifications

### Monitoring Service Integration

#### Simple Health Check
```bash
# Returns HTTP 200 (healthy) or 503 (unhealthy)
curl -I https://example.com/health/service/status

# Expected responses:
# HTTP/1.1 200 OK - System healthy
# HTTP/1.1 503 Service Unavailable - System degraded/critical
```

#### Alert Status Check
```bash
# Returns current alert level
curl https://example.com/health/alerts/status

# Response:
{
  "alert_level": "warning",
  "active_alert_count": 2,
  "critical_alert_count": 0,
  "system_status": "degraded"
}
```

#### Detailed Alert Information
```bash
# Returns full alert details
curl https://example.com/health/alerts/summary

# Response:
{
  "timestamp": "2025-01-15T10:30:00Z",
  "alert_level": "warning",
  "alerts": [
    {
      "component": "tripengine",
      "severity": "warning",
      "reason": "slow day summary update",
      "metric": "day_processing_duration_ms",
      "current_value": 5500,
      "threshold": 3000,
      "duration_minutes": 15
    }
  ],
  "healthy_components": ["wahoo", "strava", "mapbox"],
  "system_status": "degraded"
}
```

#### Component-Specific Health
```bash
# Check specific component health
curl https://example.com/health/database/status

# Response:
{
  "component": "database",
  "status": "warning",
  "metrics": {
    "query_processing_avg_ms": 800,
    "connection_success_rate": 98.5,
    "query_success_rate": 95.2
  },
  "alerts": [
    {
      "severity": "warning",
      "reason": "processing time above threshold",
      "duration_minutes": 15
    }
  ]
}
```

### Claude Analysis Integration

#### Full System Analysis
```bash
# Complete system health analysis for Claude
curl https://example.com/health/claude/analyze

# Response includes:
# - Current alerts with full context
# - Recent system changes and events
# - Performance trends and anomalies
# - Business metrics and user impact
# - Suggested remediation actions
# - Historical context for decision making
```

#### Component Diagnosis
```bash
# Detailed component analysis
curl https://example.com/health/claude/diagnose/database

# Response includes:
# - Component-specific metrics and trends
# - Related system dependencies
# - Recent changes affecting the component
# - Performance analysis and bottlenecks
# - Recommended optimizations
```

#### Trend Analysis
```bash
# Comparative trend analysis
curl "https://example.com/health/claude/trends?period=24h"
curl "https://example.com/health/claude/compare?baseline=7d&current=24h"

# Response includes:
# - Performance trends over specified periods
# - Anomaly detection results
# - Seasonal patterns and variations  
# - Growth metrics and capacity planning
# - Predictive analysis for potential issues
```

---

## Configuration and Deployment

### Environment Variables

#### Core Configuration
```bash
# Database configuration
HEALTH_DB_PATH="/data/health.db"
HEALTH_BACKUP_ENABLED="true"
HEALTH_BACKUP_RETENTION_DAYS="30"

# Monitoring configuration  
HEALTH_METRICS_FLUSH_INTERVAL="60s"
HEALTH_ALERT_CHECK_INTERVAL="30s" 
HEALTH_NODE_NAME="app-main"

# Alert thresholds
HEALTH_DAY_PROCESSING_MAX_MS="3000"
HEALTH_API_RESPONSE_MAX_MS="1000"
HEALTH_EMAIL_FAILURE_RATE_MAX="5.0"
HEALTH_PROVIDER_ERROR_RATE_MAX="10.0"

# External service monitoring
HEALTH_MAPBOX_RATE_LIMIT_BUFFER="1000"
HEALTH_SENDGRID_QUOTA_BUFFER_PCT="10.0"
HEALTH_STRIPE_WEBHOOK_DELAY_MAX_MIN="15"
```

#### Alert Sensitivity Levels
```bash
# Alert sensitivity: strict|moderate|relaxed
HEALTH_ALERT_SENSITIVITY="moderate"

# Component monitoring enable/disable
HEALTH_MONITOR_TRIPENGINE="true"
HEALTH_MONITOR_PROVIDERS="true"
HEALTH_MONITOR_EXTERNAL="true"
HEALTH_MONITOR_DATABASE="true"
HEALTH_MONITOR_BACKUP="true"
```

### Backup Integration Configuration

#### Following Established Patterns
```go
// Event-driven backup configuration
type BackupConfig struct {
    Enabled           bool   `default:"true"`
    RetentionDays     int    `default:"30"`
    BackupDir         string `default:"/data/backups/health"`
    TriggerThreshold  int    `default:"1000"` // Metrics count
    AutoCleanup       bool   `default:"true"`
}

// Backup trigger events (following established patterns)
// - After daily metric aggregation
// - After critical alert resolution  
// - After configuration changes
// - After bulk metric ingestion
```

### Application Integration Pattern

#### Data Processing Integration
```go
// In application processing logic
err = dataEngine.ProcessData(ctx, db, userId, beforeIngestionTime, beforeIngestionTime)
if err != nil {
    // Track processing error in health system
    healthState.IncrMetricWithContext("data_processing_errors", "processor", map[string]interface{}{
        "user_id": userId,
        "error": err.Error(),
    })
} else {
    // Track successful processing
    healthState.RecordTimer("data_processing_duration", processingDuration, "processor")
    healthState.IncrMetricWithContext("data_processing_success", "processor", map[string]interface{}{
        "user_id": userId,
        "records_processed": len(inData),
    })
}

// Trigger backup after successful data processing
err = database.BackupUserData(userId)
// Also trigger health metrics backup if threshold reached
healthState.PersistMetricsWithBackup()
```

#### External Service Integration Monitoring
```go
// In external service authentication flows
healthState.IncrMetricWithContext("auth_initiated", "external", map[string]interface{}{
    "service": "external_api",
    "user_id": userId,
})

// On authentication completion
healthState.IncrMetricWithContext("auth_completed", "external", map[string]interface{}{
    "service": "external_api", 
    "user_id": userId,
})

// On webhook processing
webhookStart := time.Now()
// ... process webhook ...
healthState.RecordTimer("webhook_processing_duration", time.Since(webhookStart), "external")
```

---

## Testing Strategy

### Unit Testing
```go
func TestHealthMetrics(t *testing.T) {
    // Test basic metric operations
    state := &State{}
    state.Init("test-node", 5)
    
    state.IncrMetric("test_counter")
    state.SetGauge("test_gauge", 42.0, "test")
    
    metrics := state.GetMetrics()
    assert.Equal(t, 1, metrics["test_counter"])
    assert.Equal(t, 42.0, metrics["test_gauge"])
}

func TestAlertManagement(t *testing.T) {
    // Test alert triggering and resolution
    state := &State{}
    state.InitWithBackup("test.db", BackupConfig{})
    
    // Trigger alert
    state.TriggerAlert("test_component", "warning", "test alert", 100, 50)
    alerts := state.GetActiveAlerts()
    assert.Len(t, alerts, 1)
    
    // Resolve alert  
    state.ResolveAlert("test_component", "test_metric")
    alerts = state.GetActiveAlerts()
    assert.Len(t, alerts, 0)
}
```

### Integration Testing
```go
func TestApplicationIntegration(t *testing.T) {
    // Test integration with application patterns
    state := &State{}
    state.InitWithBackup("test.db", BackupConfig{Enabled: true})
    
    // Simulate data processing
    start := time.Now()
    state.IncrMetricWithContext("records_processed", "processor", map[string]interface{}{
        "user_id": "test-user",
        "count": 10,
    })
    state.RecordTimer("processing_duration", time.Since(start), "processor")
    
    // Verify metrics stored
    metrics := state.GetComponentMetrics("processor", time.Now().Add(-1*time.Hour))
    assert.NotEmpty(t, metrics)
    
    // Verify backup triggered
    backupStatus := state.GetBackupStatus()
    assert.True(t, backupStatus.LastBackup.After(start))
}
```

### Performance Testing
```go
func BenchmarkMetricRecording(b *testing.B) {
    state := &State{}
    state.Init("bench-node", 100)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        state.IncrMetric(fmt.Sprintf("metric_%d", i%10))
    }
}

func BenchmarkConcurrentMetrics(b *testing.B) {
    state := &State{}
    state.Init("bench-node", 100)
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            state.IncrMetric(fmt.Sprintf("metric_%d", i%10))
            i++
        }
    })
}
```

---

## Migration from Current Health Package

### Backward Compatibility
- ✅ Existing `IncrMetric()` method unchanged
- ✅ Existing `Dump()` method unchanged  
- ✅ Existing `/health` endpoint unchanged
- ✅ Existing `State` struct extended, not replaced

### Migration Steps
1. **Add new dependencies**: SQLite driver, backup packages
2. **Extend State struct**: Add new fields for enhanced functionality
3. **Add new methods**: Enhanced metric methods alongside existing ones
4. **Add new endpoints**: New monitoring endpoints alongside existing `/health`
5. **Add configuration**: Environment variables for new features
6. **Optional features**: All new features are opt-in via configuration

### Example Migration
```go
// Before (existing usage)
var s health.State
s.Info("node-name", 5)
s.IncrMetric("requests")
fmt.Println(s.Dump())

// After (enhanced usage - existing code still works)
var s health.State
s.InitWithBackup("health.db", health.BackupConfig{Enabled: true}) // Enhanced init
s.IncrMetric("requests")                                         // Still works
s.SetGauge("memory_usage", 1024, "system")                      // New feature
fmt.Println(s.Dump())                                           // Still works
```

---

## Success Criteria

### Launch Readiness Checklist
- [ ] **External Monitoring Integration**: Simple endpoints return 200/503 status codes
- [ ] **Alert Detection**: System detects and reports critical issues within 60 seconds  
- [ ] **Component Health**: Individual component health checks functional
- [ ] **Claude Analysis**: Rich contextual data available for AI interpretation
- [ ] **Backup Integration**: Health metrics backed up following established patterns
- [ ] **Performance Impact**: <5ms overhead for metric recording operations
- [ ] **Reliability**: 99.9% uptime for monitoring endpoints
- [ ] **Configuration**: All thresholds configurable via environment variables

### Operational Metrics
- **Alert Accuracy**: <5% false positive rate for critical alerts
- **Response Time**: Monitoring endpoints respond within 100ms
- **Storage Efficiency**: SQLite database growth <10MB/month for typical usage
- **Backup Success**: 99.5% backup success rate with automatic recovery
- **API Reliability**: 99.9% availability for monitoring endpoints

---

## Future Enhancements (Post-Launch)

### Advanced Analytics
- Machine learning-based anomaly detection
- Predictive alerting based on trend analysis
- Capacity planning recommendations
- Seasonal pattern recognition

### Extended Integration
- Webhook notifications for critical alerts
- Integration with external incident management systems
- Custom alert rules engine
- Multi-tenant monitoring for service providers

### Performance Optimization
- Time-series database optimization
- Metric aggregation and downsampling
- Distributed monitoring for multi-instance deployments
- Real-time streaming analytics

---

## Conclusion

This enhancement plan transforms the lightweight health package into a comprehensive monitoring solution tailored for modern application architecture and operational needs. The phased approach ensures minimal disruption while delivering critical monitoring capabilities for launch readiness.

The combination of real-time monitoring endpoints and Claude-optimized analysis provides both automated alerting and intelligent system insights, enabling proactive issue resolution and informed operational decisions.

**Key Success Factors:**
- Maintain simplicity and performance of original package
- Follow proven architectural patterns (SQLite, event-driven backups)
- Provide structured data for both machines and AI analysis
- Enable external monitoring integration with minimal configuration
- Support growth from MVP launch through production scale operations