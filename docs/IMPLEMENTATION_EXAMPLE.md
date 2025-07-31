# Implementation Example: Doug's Diner Restaurant Management System

This document demonstrates how to integrate the `github.com/thisdougb/health` metrics package into a real-world application. We'll use Doug's Diner, a restaurant management system, as our example to show practical metrics implementation across different system components.

## System Overview

Doug's Diner is a Go-based restaurant management application with components for:
- **Kitchen Operations**: Order processing and meal preparation
- **Reservations**: Table booking and customer management  
- **Stock Management**: Inventory tracking and supplier integration
- **Customer Service**: Feedback collection and review management

## Architecture Pattern

The metrics implementation follows a component-based approach where each system area maintains its own metric categories while using a shared global metrics instance.

### Global Metrics State
```go
// Single metrics instance across the application
var globalMetrics *health.Metrics

func init() {
    globalMetrics = health.NewMetrics("dougs-diner", 10) // 10-sample rolling window
}
```

### HTTP Endpoint Integration
```go
func setupMetricsEndpoint() {
    http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
        // Admin authentication
        sessionID := r.Header.Get("Session-Id")
        if !validateAdminAccess(sessionID) {
            http.NotFound(w, r)
            return
        }
        
        // Return JSON metrics
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(globalMetrics.GetAllMetrics())
    })
}
```

## Component Implementation Examples

### Kitchen Component - Order Processing Pipeline

The kitchen component tracks order processing from receipt through completion:

```go
func ProcessOrder(orderID string, items []MenuItem) error {
    // Increment total orders counter
    globalMetrics.IncrementCounter("kitchen", "orders-total")
    
    startTime := time.Now()
    
    // Process each menu item
    globalMetrics.UpdateRolling("kitchen", "items-per-order", float64(len(items)))
    
    err := prepareOrder(orderID, items)
    if err != nil {
        globalMetrics.IncrementCounter("kitchen", "preparation-errors")
        return err
    }
    
    // Track successful completion
    globalMetrics.IncrementCounter("kitchen", "orders-completed")
    
    // Track processing time
    processingTime := time.Since(startTime).Milliseconds()
    globalMetrics.UpdateRolling("kitchen", "preparation-time-ms", float64(processingTime))
    
    return nil
}
```

### Customer Service Component - Review Management

The customer service component implements review collection with admin management capabilities:

```go
func CreateReview(customerID, content string) error {
    globalMetrics.IncrementCounter("service", "reviews-created")
    
    startTime := time.Now()
    
    // Create review in database
    err := storeReview(customerID, content)
    if err != nil {
        globalMetrics.IncrementCounter("service", "database-errors")
        return err
    }
    
    // Track performance
    creationTime := time.Since(startTime).Milliseconds()
    globalMetrics.UpdateRolling("service", "review-creation-time-ms", float64(creationTime))
    
    return nil
}

func CreateAdminResponse(reviewID, response string) error {
    globalMetrics.IncrementCounter("service", "admin-responses-sent")
    globalMetrics.IncrementCounter("service", "ai-integrations") // If using AI assistance
    
    startTime := time.Now()
    
    err := storeAdminResponse(reviewID, response)
    if err != nil {
        return err
    }
    
    responseTime := time.Since(startTime).Milliseconds()
    globalMetrics.UpdateRolling("service", "admin-response-time-ms", float64(responseTime))
    
    return nil
}
```

## Metrics Naming Strategy

### Business-Aligned Naming
Rather than exposing technical implementation details, metrics reflect business processes:

| Technical Function | Business Metric Name | Reasoning |
|-------------------|---------------------|-----------|
| `ProcessOrderItems()` | `orders-completed` | Tracks successful business outcome |
| `UpdateInventoryDatabase()` | `stock-updated` | Reflects inventory management process |
| `SendSupplierRequest()` | `supplier-orders-placed` | Business relationship tracking |
| `ValidateReservation()` | `bookings-confirmed` | Customer-facing process completion |

### Component Categories

#### Kitchen Operations
- **Counters**: `orders-total`, `orders-completed`, `preparation-errors`  
- **Rolling**: `preparation-time-ms`, `items-per-order`, `quality-score`

#### Customer Service  
- **Counters**: `reviews-created`, `admin-responses-sent`, `ai-integrations`
- **Rolling**: `admin-response-time-ms`, `reviews-per-customer`, `satisfaction-rating`

#### Reservations
- **Counters**: `bookings-requested`, `bookings-confirmed`, `cancellations`
- **Rolling**: `booking-time-ms`, `tables-per-booking`, `advance-booking-days`

#### Stock Management
- **Counters**: `stock-updated`, `supplier-orders-placed`, `inventory-alerts`
- **Rolling**: `reorder-frequency-days`, `stock-levels`, `supplier-delivery-time-hours`

## Production Metrics Query Example

Here's how an administrator might query the system metrics in production:

### Query Request

The raw metrics are available via an endpoint call:

```bash
curl -H "Session-Id: some super secret secret" \
     https://dougsdiner.com/internal/metrics
```

```json
{
    "Identity": "dougs-diner",
    "Started": 1753961879,
    "RollingDataSize": 10,
    "Metrics": {
        "service": {
            "admin-response-count": 1,
            "admin-responses-sent": 1,
            "admin-reviews-listed": 1,
            "admin-titles-updated": 1,
            "ai-integrations": 1,
            "reviews-created": 1,
            "reviews-retrieved": 2
        }
    },
    "RollingMetrics": {
        "service": {
            "admin-count-time-ms": 0,
            "admin-list-time-ms": 0.3,
            "admin-response-time-ms": 1.2,
            "admin-reviews-per-page": 0.2,
            "review-creation-time-ms": 1,
            "review-retrieval-time-ms": 0.4,
            "reviews-per-customer": 0.3,
            "total-reviews": 0.2
        }
    }
}
```

### Ask Claude

But the real power of this is just asking Claude:

```
> ok, we just implemented metrics. see the doc for how to query metrics in production.  can you give me a summary

**Summary Analysis:**

**System Status:**
- Service: `dougs-diner` 
- Started: 1753961879 (Unix timestamp)
- Rolling data window: 10 samples

**Customer Service Activity:**
- **Admin Operations**: 1 response sent, 1 review listed, 1 title updated
- **AI Integration**: 1 AI-assisted interaction (likely automated review response)
- **Customer Activity**: 1 review created, 2 reviews retrieved

**Performance Metrics (Rolling Averages):**
- **Admin Operations**: Excellent performance (0-1.2ms response times)
- **Review Operations**: Sub-millisecond performance (0.4-1ms)
- **System Efficiency**: 0.2 reviews per page average, 0.3 reviews per customer

The metrics show the customer service system is performing well with fast response times and recent admin activity, indicating healthy review management operations.
```

## Key Implementation Decisions

### 1. Component-Based Organization
**Decision**: Organize metrics by business component rather than technical layer.
**Reasoning**: Makes metrics more meaningful to business stakeholders and easier to correlate with user experience.

### 2. Dual Metric Types
**Decision**: Use counters for events and rolling averages for performance.
**Reasoning**: Counters track business volume, rolling averages smooth out performance variations for operational insights.

### 3. Business-Aligned Naming
**Decision**: Use descriptive business process names instead of technical function names.
**Reasoning**: Metrics become documentation of business processes and are understandable to non-technical stakeholders.

### 4. Admin Authentication
**Decision**: Protect metrics endpoint with admin authentication headers.
**Reasoning**: Metrics may reveal business intelligence; access should be controlled while maintaining programmatic access for monitoring tools.

### 5. Global Metrics Instance
**Decision**: Single metrics instance shared across all components.
**Reasoning**: Simplifies metric aggregation and provides consistent system-wide view while allowing component-specific categorization.

## Testing Strategy

### Unit Test Integration
```go
func TestOrderProcessing(t *testing.T) {
    // Reset metrics for test isolation
    testMetrics := health.NewMetrics("test-diner", 5)
    
    // Process test order
    err := ProcessOrder("test-123", []MenuItem{
        {Name: "Burger", Price: 12.99},
        {Name: "Fries", Price: 4.99},
    })
    
    assert.NoError(t, err)
    
    // Verify metrics were recorded
    metrics := testMetrics.GetAllMetrics()
    assert.Equal(t, 1, metrics.Metrics["kitchen"]["orders-completed"])
    assert.Equal(t, 2.0, metrics.RollingMetrics["kitchen"]["items-per-order"])
}
```

### Integration Testing
Test metric collection under realistic load conditions and verify performance overhead remains acceptable.

## Monitoring and Alerting Guidelines

### Key Performance Indicators
- **Success Rate**: `orders-completed` / `orders-total` × 100
- **Error Rate**: `preparation-errors` / `orders-total` × 100
- **Customer Satisfaction**: `satisfaction-rating` rolling average

### Operational Thresholds  
- **High Error Rate**: Error rate > 5% indicates kitchen operational issues
- **Slow Service**: `preparation-time-ms` > 15 minutes suggests capacity problems
- **Low Satisfaction**: `satisfaction-rating` < 4.0 requires immediate attention

This implementation demonstrates how metrics become a natural part of business process tracking, providing operational visibility while maintaining clean separation of concerns in the application architecture.
