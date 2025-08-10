#!/bin/bash

echo "Testing Observability Plugin External API"
echo "========================================="

# Health check
echo -e "\n1. Health Check:"
curl -s http://localhost:8091/health | jq .

# Status endpoint
echo -e "\n2. Status Endpoint:"
curl -s http://localhost:8091/api/v1/status | jq .

# Simple metric query
echo -e "\n3. Simple Metric Query (up):"
curl -s "http://localhost:8091/api/v1/metrics/query?query=up" | jq '.data | length'

# Node CPU query
echo -e "\n4. Node CPU Usage Query:"
curl -s "http://localhost:8091/api/v1/metrics/query?query=node_cpu_seconds_total" | jq '.status'

# Test rate limiting
echo -e "\n5. Testing Rate Limiting (10 rapid requests):"
for i in {1..10}; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8091/api/v1/metrics/query?query=up")
    echo "Request $i: HTTP $STATUS"
done

# Test validation - blocked metric
echo -e "\n6. Testing Validation - Blocked Metric:"
curl -s "http://localhost:8091/api/v1/metrics/query?query=api_secret_key" | jq .

# Test validation - complex query
echo -e "\n7. Testing Complex Query:"
COMPLEX_QUERY="rate(node_cpu_seconds_total[5m]) + rate(node_memory_MemTotal_bytes[5m])"
curl -s "http://localhost:8091/api/v1/metrics/query?query=$COMPLEX_QUERY" | jq '.status'

# Check audit log
echo -e "\n8. Checking Audit Log:"
if [ -f audit.log ]; then
    echo "Last 5 audit entries:"
    tail -5 audit.log | jq -c '{event_type, path, sensitive}'
else
    echo "Audit log not found"
fi