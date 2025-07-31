# Tutorial: Building Doug's Diner with Health Metrics

This tutorial shows you how to build a complete Go application with integrated health metrics. We'll create Doug's Diner, a restaurant management system, step by step. This is perfect for developers new to Go who want to learn both application structure and metrics integration.

## What You'll Build

By the end of this tutorial, you'll have a working restaurant management system that tracks business metrics like order processing times, customer reviews, and system performance.

**Features we'll implement:**
- Order processing with kitchen metrics
- Customer review system with response tracking
- Admin dashboard with real-time metrics
- HTTP endpoints for external monitoring

## Project Structure

Here's how we'll organize our code (similar to the tripkist app pattern):

```
dougs-diner/
├── main.go                 # Application entry point and HTTP server
├── metrics/
│   └── metrics.go         # Global metrics setup and management
├── kitchen/
│   └── kitchen.go         # Order processing with metrics
├── service/
│   └── service.go         # Customer reviews with metrics
└── go.mod                 # Go module dependencies
```

This structure separates concerns while keeping related code together - each component has its own package but shares the same metrics system.

## Understanding the Code Structure

The following examples show the exact code structure used in this implementation. Each code block includes the file path where you would place this code in your project.

### Global Metrics State
**File: `metrics/metrics.go`**

```go
// File: metrics/metrics.go
// This file sets up our application-wide metrics system

// Single metrics instance across the application
// This variable is shared by all components (kitchen, service, etc.)
var globalMetrics *health.Metrics

func init() {
    // This runs automatically when the package loads
    // Creates metrics with app name "dougs-diner" and 10-sample rolling window
    globalMetrics = health.NewMetrics("dougs-diner", 10) // 10-sample rolling window
}
```

**What this code does:**
- Creates one shared metrics instance for the entire application
- The `init()` function runs automatically when your program starts
- "dougs-diner" identifies your app in the metrics output
- `10` means rolling averages use the last 10 data points

### HTTP Endpoint Integration
**File: `main.go`**

```go
// File: main.go
// This sets up the web server and metrics endpoint

func setupMetricsEndpoint() {
    // Create an HTTP endpoint at /internal/metrics
    http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
        // Admin authentication - check for valid session
        sessionID := r.Header.Get("Session-Id")
        if !validateAdminAccess(sessionID) {
            // Hide the endpoint if not authenticated (return 404, not 401)
            http.NotFound(w, r)
            return
        }
        
        // Return JSON metrics to authenticated users
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(globalMetrics.Dump()))
    })
}
```

**What this code does:**
- Creates a protected `/internal/metrics` endpoint for viewing metrics
- Checks authentication using a session ID from HTTP headers
- Returns 404 (not found) instead of 401 (unauthorized) to hide the endpoint
- Sends metrics as JSON when authentication succeeds

## Component Implementation Examples

### Kitchen Component - Order Processing Pipeline
**File: `kitchen/kitchen.go`**

The kitchen component tracks order processing from receipt through completion:

```go
// File: kitchen/kitchen.go
// This handles restaurant order processing with integrated metrics

func ProcessOrder(orderID string, items []MenuItem) error {
    // Increment total orders counter - tracks business volume
    globalMetrics.IncrementCounter("kitchen", "orders-total")
    
    // Start timing how long order processing takes
    startTime := time.Now()
    
    // Track average items per order - helps with kitchen planning
    globalMetrics.UpdateRolling("kitchen", "items-per-order", float64(len(items)))
    
    // Do the actual work of preparing the order
    err := prepareOrder(orderID, items)
    if err != nil {
        // Count errors so we can track kitchen problems
        globalMetrics.IncrementCounter("kitchen", "preparation-errors")
        return err
    }
    
    // Track successful completion - measures success rate
    globalMetrics.IncrementCounter("kitchen", "orders-completed")
    
    // Calculate and record how long this order took to process
    processingTime := time.Since(startTime).Milliseconds()
    globalMetrics.UpdateRolling("kitchen", "preparation-time-ms", float64(processingTime))
    
    return nil
}
```

**What this code does:**
- `IncrementCounter()` adds 1 to a metric (like counting events)
- `UpdateRolling()` records a value for averaging (like response times)
- `time.Now()` and `time.Since()` measure how long operations take
- Component name "kitchen" groups all kitchen-related metrics together

### Customer Service Component - Review Management
**File: `service/service.go`**

The customer service component implements review collection with admin management capabilities:

```go
// File: service/service.go
// This handles customer reviews and admin responses with metrics tracking

func CreateReview(customerID, content string) error {
    // Count every review created - tracks customer engagement
    globalMetrics.IncrementCounter("service", "reviews-created")
    
    // Start timing the database operation
    startTime := time.Now()
    
    // Create review in database - the actual business logic
    err := storeReview(customerID, content)
    if err != nil {
        // Track database problems separately from business logic errors
        globalMetrics.IncrementCounter("service", "database-errors")
        return err
    }
    
    // Track performance - how fast can we create reviews?
    creationTime := time.Since(startTime).Milliseconds()
    globalMetrics.UpdateRolling("service", "review-creation-time-ms", float64(creationTime))
    
    return nil
}

func CreateAdminResponse(reviewID, response string) error {
    // Count admin responses - tracks staff engagement with customers
    globalMetrics.IncrementCounter("service", "admin-responses-sent")
    
    // Track AI usage if you're using AI to help write responses
    globalMetrics.IncrementCounter("service", "ai-integrations") // If using AI assistance
    
    // Time the admin response creation
    startTime := time.Now()
    
    // Store the admin response in database
    err := storeAdminResponse(reviewID, response)
    if err != nil {
        return err
    }
    
    // Track how fast admins can respond to reviews
    responseTime := time.Since(startTime).Milliseconds()
    globalMetrics.UpdateRolling("service", "admin-response-time-ms", float64(responseTime))
    
    return nil
}
```

**What this code does:**
- Each function tracks both success events and performance timing
- Database errors are tracked separately from business logic errors
- "service" component groups all customer service metrics together
- AI usage tracking helps measure automation vs human work

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
**File: `kitchen/kitchen_test.go`**

```go
// File: kitchen/kitchen_test.go
// This shows how to test your functions while also testing the metrics

package kitchen

import (
    "encoding/json"
    "strings"
    "testing"
    "github.com/thisdougb/health"
)

func TestOrderProcessing(t *testing.T) {
    // Create separate metrics instance for testing - keeps tests isolated
    testMetrics := health.NewState()
    testMetrics.SetConfig("test-diner", 5)
    
    // Process a test order with 2 items
    err := ProcessOrder("test-123", []MenuItem{
        {Name: "Burger", Price: 12.99},
        {Name: "Fries", Price: 4.99},
    })
    
    // Verify the function worked without errors
    if err != nil {
        t.Fatalf("ProcessOrder failed: %v", err)
    }
    
    // Get metrics as JSON string
    metricsJSON := testMetrics.Dump()
    
    // Basic verification that metrics were recorded
    if !strings.Contains(metricsJSON, `"kitchen"`) {
        t.Error("Expected kitchen metrics to be recorded")
    }
    
    if !strings.Contains(metricsJSON, `"orders-completed"`) {
        t.Error("Expected orders-completed metric to be recorded")
    }
    
    // For more detailed testing, parse the JSON
    var metrics map[string]interface{}
    if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
        t.Fatalf("Failed to parse metrics JSON: %v", err)
    }
    
    // Verify structure exists (basic validation)
    if _, exists := metrics["Metrics"]; !exists {
        t.Error("Expected Metrics section in JSON output")
    }
}
```

**Testing best practices:**
- Use separate metrics instances for each test to avoid interference
- Test both your business logic AND that metrics are recorded correctly
- Use `strings.Contains()` for basic JSON content verification
- Parse JSON with `json.Unmarshal()` for detailed metric value testing
- Use `t.Fatalf()` for test setup failures, `t.Error()` for assertion failures

### Integration Testing
**File: `main_test.go`**

Test metric collection under realistic load conditions and verify performance overhead remains acceptable.

**What to test:**
- Multiple components recording metrics simultaneously
- High-volume metric recording doesn't slow down your application
- HTTP endpoints return proper JSON and respect authentication
- Rolling averages calculate correctly over time

## Monitoring and Alerting Guidelines

### Key Performance Indicators
- **Success Rate**: `orders-completed` / `orders-total` × 100
- **Error Rate**: `preparation-errors` / `orders-total` × 100
- **Customer Satisfaction**: `satisfaction-rating` rolling average

### Operational Thresholds  
- **High Error Rate**: Error rate > 5% indicates kitchen operational issues
- **Slow Service**: `preparation-time-ms` > 15 minutes suggests capacity problems
- **Low Satisfaction**: `satisfaction-rating` < 4.0 requires immediate attention

## Getting Started - Your Next Steps

**For beginners learning Go and metrics:**

1. **Start simple:** Begin with just one component (like kitchen) and a few basic counters
2. **Add timing gradually:** Once counters work, add `time.Now()` and `time.Since()` for performance metrics  
3. **Test everything:** Write tests for both your business logic and metric recording
4. **Build incrementally:** Add components one at a time rather than building everything at once

**Common mistakes to avoid:**
- Don't create a new metrics instance in every function - use the global one
- Don't forget to handle errors when recording metrics fails
- Don't record metrics inside tight loops - it can slow your application
- Don't expose metrics endpoints without authentication

**File structure recap:**
```
dougs-diner/
├── main.go                 # HTTP server setup and metrics endpoint
├── metrics/metrics.go      # Global metrics initialization  
├── kitchen/kitchen.go      # Business logic with kitchen metrics
├── service/service.go      # Business logic with service metrics
└── *_test.go              # Tests for each component
```

This implementation demonstrates how metrics become a natural part of business process tracking, providing operational visibility while maintaining clean separation of concerns in the application architecture.
