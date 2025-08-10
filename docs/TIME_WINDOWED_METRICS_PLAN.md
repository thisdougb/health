# Time-Windowed Metrics with Move-and-Flush Architecture

## Summary

This document outlines the plan to replace the current cumulative counter system with a time-windowed metrics approach that provides richer statistical data while solving server restart data loss issues.

## Current Problems

1. **Server Restart Data Loss**: Cumulative counters reset to 0 on restart
2. **Storage Inefficiency**: Each increment creates a separate database row
3. **Limited Statistics**: Only cumulative totals, no rate/trend analysis
4. **High Lock Contention**: Every metric increment immediately queues to SQLite

## Proposed Solution: Time-Windowed Metrics

### Core Concept

- Collect raw metric values in 60-second time windows
- Calculate min/max/avg/count statistics during flush to database  
- Use "move-and-flush" architecture to minimize lock contention
- Single unified system for both counter and value metrics

### Pseudo Code

```go
// Data Structures
type StateImpl struct {
    SampledMetrics map[component][timekey][metric][]float64  // Active collection
    FlushQueue     map[component][timekey][metric][]float64  // Ready for DB write
    collectMutex   sync.RWMutex                             // Protects active collection
    flushMutex     sync.Mutex                               // Protects flush queue
}

// Collection (hot path)
func IncrMetric(name string) {
    timekey := getCurrentTimeKey()  // "202508191015" (60s windows)
    collectMutex.Lock()
    SampledMetrics["Global"][timekey][name].append(1.0)
    collectMutex.Unlock()
}

func AddMetric(name string, value float64) {
    timekey := getCurrentTimeKey() 
    collectMutex.Lock()
    SampledMetrics["Global"][timekey][name].append(value)
    collectMutex.Unlock()
}

// Background flush (every 60s)
func moveAndFlush() {
    currentTimeKey := getCurrentTimeKey()
    
    // Move completed windows to flush queue (minimal lock time)
    collectMutex.Lock()
    for component, timeWindows := range SampledMetrics {
        for timekey, metrics := range timeWindows {
            if timekey != currentTimeKey {
                FlushQueue[component][timekey] = metrics
                delete(SampledMetrics[component], timekey)
            }
        }
    }
    collectMutex.Unlock()
    
    // Process flush queue (no collection locks needed)
    flushMutex.Lock()
    for component, timeWindows := range FlushQueue {
        for timekey, metrics := range timeWindows {
            for metric, values := range metrics {
                min, max, avg, count := calculateStats(values)
                writeToSQL(timekey, component, metric, min, max, avg, count)
            }
        }
    }
    clear(FlushQueue)
    flushMutex.Unlock()
}

func calculateStats(values []float64) (min, max, avg float64, count int) {
    if len(values) == 0 {
        return 0, 0, 0, 0
    }
    
    min, max = values[0], values[0]
    var sum float64
    
    for _, v := range values {
        if v < min { min = v }
        if v > max { max = v }
        sum += v
    }
    
    return min, max, sum/float64(len(values)), len(values)
}
```

### SQL Schema

```sql
CREATE TABLE time_series_metrics (
    time_window_start INTEGER,  -- Unix timestamp of window start
    component TEXT,            -- "Global", "database", "webserver", etc.
    metric TEXT,              -- "web_requests", "response_time", etc.
    min_value REAL,           -- Minimum value in window
    max_value REAL,           -- Maximum value in window  
    avg_value REAL,           -- Average value in window
    count INTEGER,            -- Number of events in window
    PRIMARY KEY (time_window_start, component, metric)
);
```

## Storage Examples

### In-Memory Collection (60s window)
```go
// Counter metrics: each increment appends 1.0
SampledMetrics["Global"]["202508191014"]["web_requests"] = [1.0, 1.0, 1.0, ..., 1.0] // 150 values

// Value metrics: each measurement appends actual value  
SampledMetrics["Global"]["202508191014"]["response_time"] = [145.2, 152.1, 138.7, 167.3, ...] // response times
SampledMetrics["database"]["202508191014"]["query_time"] = [23.1, 45.2, 12.8, 89.4, ...] // query times
```

### SQL Output After Flush
```
time_window_start | component | metric        | min   | max   | avg   | count
1692454440       | Global    | web_requests  | 1.0   | 1.0   | 1.0   | 150
1692454440       | Global    | response_time | 138.7 | 167.3 | 150.8 | 45  
1692454440       | database  | query_time    | 12.8  | 89.4  | 42.6  | 23
```

## Implementation Plan

### Phase 1: Core Time-Windowed Collection
1. **Add `SampledMetrics` map to `StateImpl`** with slice storage for raw values
2. **Modify `IncrMetric()` and `AddMetric()`** to append values to current time window
3. **Implement `getCurrentTimeKey()`** function for 60-second time windows
4. **Update `StateImpl` structure** to include new fields and mutexes

### Phase 2: Move-and-Flush Architecture  
5. **Add `FlushQueue` map and separate mutexes** to `StateImpl`
6. **Implement `moveToFlushQueue()`** logic in background goroutine
7. **Create `calculateStats()`** function for min/max/avg/count calculation
8. **Update SQL schema** and create new `time_series_metrics` table
9. **Modify storage backend** to write aggregated statistics

### Phase 3: Integration and Migration
10. **Replace immediate SQLite queuing** with time-windowed flush mechanism
11. **Update JSON output** to show current window statistics  
12. **Extend time series endpoints** to query new schema
13. **Add migration logic** for existing installations
14. **Update tests** for new time-windowed behavior

## Benefits

### Performance Benefits
- **~99% Storage Reduction**: 150 individual rows → 1 aggregated row per metric per window
- **Minimal Lock Contention**: Hot collection path uses brief locks, flush happens on moved data
- **Non-blocking Flush**: Statistics calculation doesn't block metric collection
- **Batch Efficiency**: Full time windows flushed as single database transactions

### Data Quality Benefits  
- **No Server Restart Data Loss**: All historical data persisted in time windows
- **Rich Statistics**: True min/max/avg for measured values, proper event counts
- **Flexible Aggregation**: UI can sum for totals, calculate rates, show trends
- **Zero Configuration**: 60-second default windows work for all metrics

### Storage Efficiency
- **Before**: 10,000 metric increments = 10,000 database rows
- **After**: 10,000 metric increments = 1 database row with count=10000
- **Estimated**: ~60MB/month vs current ~160MB/month (62% reduction)

## Migration Strategy

1. **Backward Compatibility**: Keep existing JSON API structure during transition
2. **Dual Write Period**: Write to both old and new schemas temporarily  
3. **Gradual Migration**: Migrate time series endpoints to new schema first
4. **Data Export**: Provide tools to migrate existing cumulative data to time windows
5. **Feature Flag**: Allow switching between old and new systems during testing

## Usage Examples

### Application Code (Unchanged)
```go
state.IncrMetric("web_requests")           // Same simple call
state.AddMetric("response_time", 145.2)    // Same simple call  
state.IncrComponentMetric("db", "queries") // Same simple call
```

### UI Queries (New Capabilities)
```sql
-- Total requests in last hour
SELECT SUM(count) FROM time_series_metrics 
WHERE metric='web_requests' AND time_window_start >= last_hour;

-- Peak request rate per minute  
SELECT MAX(count) FROM time_series_metrics 
WHERE metric='web_requests' GROUP BY time_window_start;

-- Response time trends
SELECT time_window_start, avg_value, min_value, max_value 
FROM time_series_metrics 
WHERE metric='response_time' ORDER BY time_window_start;
```

## Risk Mitigation

- **Memory Usage**: Monitor slice growth during high-traffic periods
- **Lock Performance**: Measure collection mutex contention under load  
- **Flush Latency**: Ensure background flush completes within 60-second windows
- **Migration Issues**: Comprehensive testing with existing production data
- **Rollback Plan**: Keep old system available during transition period