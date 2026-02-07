# ADR 005: Request Validation Strategy

## Status
Accepted

## Context
External API endpoints need protection against malicious queries that could expose sensitive data, consume excessive resources, or exploit system vulnerabilities. The validation must be both secure and performant.

## Decision
Implement multi-layered request validation:

1. **Query Complexity Analysis**
   - Algorithmic scoring based on query operations
   - Configurable complexity limits
   - Prevention of resource-intensive queries

2. **Metric Pattern Filtering**
   - Regex-based allowed/blocked patterns
   - Protection against sensitive metric exposure
   - Support for environment-specific rules

3. **Time Range Validation**
   - Limits on historical query ranges
   - Prevention of excessive data retrieval
   - Configurable maximum lookback periods

4. **General Request Validation**
   - HTTP method validation
   - Content-Length limits
   - Parameter sanitization

## Alternatives Considered
1. **Query parsing and AST analysis**: Rejected due to complexity and performance overhead
2. **Simple string matching**: Rejected as insufficient for complex attack vectors
3. **External validation service**: Rejected to avoid additional dependencies

## Consequences

### Positive
- Protection against resource exhaustion attacks
- Prevention of sensitive data exposure
- Fast validation with regex caching
- Configurable security policies
- Comprehensive attack surface coverage

### Negative
- False positives for legitimate complex queries
- Regex compilation overhead on startup
- Configuration complexity for custom environments

## Implementation Notes
- Complexity scoring weights: basic operations (1), aggregations (10-25)
- Default complexity limit: 100 points
- Time range limit: 24 hours maximum
- Request body limit: 1MB maximum
- Blocked patterns: none by default (empty array), configurable per deployment