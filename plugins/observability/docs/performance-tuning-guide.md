# Performance Tuning Guide

This guide provides recommendations for optimizing the performance of the observability plugin across different deployment scenarios.

## Prometheus Client Optimization

### Connection Settings
```yaml
prometheus:
  timeout: 30s          # Increase for slow networks
  retry:
    maxRetries: 3       # Balance between reliability and latency
    initialDelay: 1s
    maxDelay: 10s
    multiplier: 2.0
```

### TLS Configuration
```yaml
prometheus:
  tls:
    enabled: true
    insecureSkipVerify: false  # Set to true only for testing
```

**Impact**: TLS verification adds ~5-10ms per request but provides security.

### Circuit Breaker Tuning
```go
// For high-availability environments
circuitConfig := CircuitBreakerConfig{
    FailureThreshold: 5,      // Failures before opening
    RecoveryTimeout: 10s,     // Time before retry
    RequestTimeout: 30s,      // Individual request timeout
    MaxConcurrentRequests: 100, // Concurrent request limit
}
```

## Caching Optimization

### Cache Configuration
```yaml
cache:
  enabled: true
  defaultTTL: 30s        # Balance between freshness and performance
  maxSize: 1000          # Adjust based on available memory
  cleanupInterval: 5m    # More frequent cleanup for memory-constrained environments
```

### Cache Key Optimization
- **Default Strategy**: Uses query hash + time window
- **Memory Usage**: ~100 bytes per cache entry
- **Hit Rate Target**: >80% for optimal performance

### Memory Considerations
```bash
# Monitor cache memory usage
curl localhost:8080/api/v1/status | jq '.data.cache'

# Expected memory per 1000 entries: ~100KB
```

## Security Performance Impact

### Rate Limiting
```yaml
security:
  api:
    rateLimit:
      requestsPerMinute: 1000        # ✓ Implemented - Higher for high-traffic environments
      burstCapacity: 100             # ⚠ NOT YET IMPLEMENTED - Planned feature
      globalRequestsPerMinute: 5000  # ✓ Implemented
      cleanupInterval: 5m            # ⚠ NOT YET IMPLEMENTED - Planned feature
```

**Performance Impact**: ~0.1ms per request

**Note**: Only `requestsPerMinute` and `globalRequestsPerMinute` are currently parsed. `burstCapacity` and `cleanupInterval` are planned features.

### Request Validation
```yaml
security:
  api:
    validation:
      enabled: true                  # ✓ Implemented
      maxQueryLength: 2048           # ✓ Implemented - Reduce for memory-constrained environments
      maxComplexityScore: 200        # ⚠ NOT YET IMPLEMENTED - Planned feature
      maxTimeRangeDays: 7            # ⚠ NOT YET IMPLEMENTED - Planned feature
```

**Performance Impact**: ~0.5ms per request for regex validation

**Note**: Only `enabled` and `maxQueryLength` are currently parsed. Complexity scoring and time range validation are planned features.

### Audit Logging
```yaml
security:
  api:
    audit:
      enabled: true               # ✓ Implemented
      logFile: "/var/log/obs.log" # ✓ Implemented - Path to audit log file
      logLevel: "warn"            # ⚠ NOT YET IMPLEMENTED - Planned feature
      sensitiveOnly: true         # ⚠ NOT YET IMPLEMENTED - Planned feature
      includeBodies: false        # ⚠ NOT YET IMPLEMENTED - Planned feature
      maxFileSizeMB: 100          # ⚠ NOT YET IMPLEMENTED - Planned feature
```

**Performance Impact**:
- File I/O: ~1-2ms per logged request

**Note**: Only `enabled` and `logFile` are currently parsed. Log level filtering, selective logging, and rotation settings are planned features.
- Memory logging: ~0.1ms per request

## Historical Data Collection

### Collection Intervals
```yaml
# Balance between data granularity and system load
historical:
  collectInterval: 5m      # More frequent = more data + higher load
  retention: 720h          # 30 days default
  maxDataPoints: 10000     # Limit memory usage
```

### Query Optimization
```yaml
# Use efficient queries for collection
historical:
  queries:
    cpu_usage: "avg(rate(cpu_seconds_total[5m])) by (instance)"
    memory_usage: "avg(memory_usage_bytes / memory_total_bytes) by (instance)"
```

**Performance Impact**: 
- 5-minute intervals: ~10 metrics/second system load
- 1-minute intervals: ~50 metrics/second system load

## Database and Storage

### Historical Data Storage
```bash
# Optimize storage directory location
mkdir -p /fast-ssd/s9s-observability/historical
```

### Cleanup and Retention
```yaml
historical:
  retention: 720h          # Automatic cleanup after 30 days
  maxDataPoints: 10000     # Prevent unbounded growth
```

## Memory Optimization

### Go Runtime Tuning
```bash
# Set garbage collection target (default: 100)
export GOGC=50              # More aggressive GC for memory-constrained environments
export GOMEMLIMIT=512MiB    # Hard memory limit
```

### Component Memory Usage (Approximate)
- **Prometheus Client**: 10-50MB (depends on query cache)
- **Historical Collector**: 20-100MB (depends on retention)
- **Security Components**: 5-20MB (depends on rate limiting clients)
- **Subscription Manager**: 5-15MB (depends on active subscriptions)

## Network Optimization

### Connection Pooling
```go
// HTTP client configuration for Prometheus
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### Request Batching
```yaml
# Enable request batching for multiple queries
prometheus:
  enableBatching: true
  batchSize: 10
  batchTimeout: 5s
```

## Deployment Scenarios

### High-Traffic Environment
```yaml
prometheus:
  timeout: 60s
cache:
  maxSize: 5000
  defaultTTL: 60s
security:
  api:
    rateLimit:
      requestsPerMinute: 5000
      globalRequestsPerMinute: 20000
historical:
  collectInterval: 10m  # Reduce collection frequency
```

### Memory-Constrained Environment
```yaml
cache:
  maxSize: 200
  defaultTTL: 120s
security:
  api:
    audit:
      logLevel: "error"
      sensitiveOnly: true
historical:
  maxDataPoints: 2000
  retention: 168h  # 7 days
```

### Security-Focused Environment
```yaml
security:
  api:
    rateLimit:
      requestsPerMinute: 100
    validation:
      maxComplexityScore: 50
      maxTimeRangeDays: 1
    audit:
      logLevel: "info"
      sensitiveOnly: false
      includeBodies: false
```

## Monitoring Performance

### Key Metrics to Monitor
1. **Request Latency**: p95 latency < 100ms for cached queries
2. **Cache Hit Rate**: Should be > 80%
3. **Circuit Breaker State**: Should remain closed > 99% of time
4. **Memory Usage**: Should remain stable over time
5. **Error Rates**: Should be < 1% for all endpoints

### Performance Debugging
```bash
# Check plugin performance metrics
curl localhost:8080/api/v1/status | jq '.data'

# Monitor system resources
top -p $(pgrep s9s)

# Check log files for performance issues
tail -f /var/log/s9s-observability-audit.log | grep -E "(error|timeout|slow)"
```

## Troubleshooting Common Issues

### High Memory Usage
1. Reduce cache size and TTL
2. Decrease historical data retention
3. Enable more aggressive garbage collection
4. Check for memory leaks in rate limiter client cleanup

### High CPU Usage
1. Increase cache TTL to reduce Prometheus queries
2. Optimize Prometheus queries for efficiency
3. Reduce historical collection frequency
4. Disable non-essential security features in development

### High Network Latency
1. Increase connection pool sizes
2. Enable request batching
3. Optimize Prometheus query patterns
4. Use connection keep-alive

### Security Performance Issues
1. Use "warn" or "error" log levels instead of "info"
2. Enable `sensitiveOnly` audit logging
3. Increase rate limiting thresholds for trusted clients
4. Optimize validation regex patterns

## Benchmarking

### Load Testing
```bash
# Test API endpoints under load
ab -n 1000 -c 10 -H "Authorization: Bearer test-token" \
   "http://localhost:8080/api/v1/metrics/query?query=cpu_usage"
```

### Performance Profiling
```bash
# Enable Go profiling
export ENABLE_PPROF=true

# Access profiling endpoints
go tool pprof http://localhost:6060/debug/pprof/profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

## Recommended Configurations

### Production Environment
- Cache TTL: 60-120 seconds
- Rate limiting: 1000 requests/minute per client
- Audit logging: "warn" level, sensitive only
- Historical retention: 30 days
- Collection interval: 5-10 minutes

### Development Environment
- Cache TTL: 10-30 seconds
- Rate limiting: 10000 requests/minute per client
- Audit logging: "info" level, all requests
- Historical retention: 7 days
- Collection interval: 1-2 minutes

### Testing Environment
- Cache TTL: 5 seconds
- Rate limiting: disabled
- Audit logging: disabled
- Historical retention: 1 day
- Collection interval: 30 seconds