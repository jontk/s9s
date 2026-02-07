# ADR 004: Rate Limiting Design

## Status
Accepted

## Context
The external API needs protection against abuse and resource exhaustion attacks. The system must handle both legitimate high-traffic scenarios and malicious request patterns while maintaining fair access for all clients.

## Decision
Implement a dual-layer rate limiting system using the token bucket algorithm:

1. **Per-Client Rate Limiting**
   - Individual token buckets for each client (identified by IP + User-Agent)
   - Configurable requests per minute and burst capacity
   - Automatic client cleanup to prevent memory exhaustion

2. **Global Rate Limiting**
   - System-wide request limits to protect server resources
   - Shared bucket across all clients
   - Priority handling for authenticated vs anonymous requests

3. **Token Bucket Algorithm**
   - Smooth request distribution over time
   - Burst capacity for legitimate traffic spikes
   - Gradual token refill to maintain sustained rates

## Alternatives Considered
1. **Fixed window rate limiting**: Rejected due to traffic burst issues
2. **Sliding window log**: Rejected due to memory overhead
3. **Leaky bucket**: Rejected as it's too strict for API access patterns
4. **Redis-based rate limiting**: Rejected to avoid external dependencies

## Consequences

### Positive
- Effective protection against abuse
- Fair resource allocation among clients
- Smooth traffic handling with burst support
- Memory-efficient implementation
- No external dependencies

### Negative
- In-memory state loss on restart
- Complex configuration tuning
- Potential false positives for legitimate high-volume clients

## Implementation Notes
- Default: 100 requests/minute per client, 1000 global
- Burst capacity: 10 requests for smooth traffic handling
- Client cleanup runs every 10 minutes
- Rate limit headers included in HTTP responses