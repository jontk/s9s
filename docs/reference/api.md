# API Documentation

Complete REST API reference for S9S, enabling programmatic access to cluster management, job operations, and system information.

## API Overview

The S9S REST API provides:
- RESTful endpoints for all S9S functionality
- JSON request/response format
- Authentication via tokens, OAuth2, or certificates
- Rate limiting and caching
- WebSocket support for real-time updates
- OpenAPI 3.0 specification

### Base URL
```
https://your-s9s-instance.com/api/v1
```

### API Versions
- **v1**: Current stable version
- **v2**: Beta version (if available)

## Authentication

### Token Authentication
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     https://api.example.com/v1/jobs
```

### API Key Authentication
```bash
curl -H "X-API-Key: YOUR_API_KEY" \
     https://api.example.com/v1/jobs
```

### OAuth 2.0
```bash
# Get access token
curl -X POST https://api.example.com/oauth/token \
     -H "Content-Type: application/json" \
     -d '{
       "client_id": "your_client_id",
       "client_secret": "your_client_secret",
       "grant_type": "client_credentials"
     }'

# Use access token
curl -H "Authorization: Bearer ACCESS_TOKEN" \
     https://api.example.com/v1/jobs
```

## Jobs API

### List Jobs
```http
GET /v1/jobs
```

**Parameters:**
- `state` (string): Filter by job state
- `user` (string): Filter by username
- `partition` (string): Filter by partition
- `limit` (integer): Maximum results (default: 100)
- `offset` (integer): Pagination offset
- `sort` (string): Sort field (default: submit_time)
- `order` (string): Sort order - asc/desc (default: desc)

**Example Request:**
```bash
curl "https://api.example.com/v1/jobs?state=RUNNING&user=alice&limit=50"
```

**Example Response:**
```json
{
  "jobs": [
    {
      "job_id": "12345",
      "job_name": "simulation_run",
      "user": "alice",
      "account": "research",
      "partition": "gpu",
      "state": "RUNNING",
      "submit_time": "2023-12-15T10:30:00Z",
      "start_time": "2023-12-15T10:32:15Z",
      "end_time": null,
      "elapsed_time": "2:15:30",
      "time_limit": "4:00:00",
      "nodes": ["node001", "node002"],
      "node_count": 2,
      "cpu_count": 32,
      "memory": "128GB",
      "gpus": 4,
      "priority": 1000,
      "qos": "normal",
      "working_directory": "/home/alice/simulations",
      "standard_output": "/home/alice/logs/job_12345.out",
      "standard_error": "/home/alice/logs/job_12345.err",
      "exit_code": null,
      "reason": null
    }
  ],
  "total": 156,
  "limit": 50,
  "offset": 0,
  "has_more": true
}
```

### Get Job Details
```http
GET /v1/jobs/{job_id}
```

**Example Request:**
```bash
curl "https://api.example.com/v1/jobs/12345"
```

**Example Response:**
```json
{
  "job_id": "12345",
  "job_name": "simulation_run",
  "user": "alice",
  "account": "research",
  "partition": "gpu",
  "state": "RUNNING",
  "submit_time": "2023-12-15T10:30:00Z",
  "start_time": "2023-12-15T10:32:15Z",
  "end_time": null,
  "elapsed_time": "2:15:30",
  "time_limit": "4:00:00",
  "nodes": ["node001", "node002"],
  "node_count": 2,
  "cpu_count": 32,
  "memory": "128GB",
  "gpus": 4,
  "priority": 1000,
  "qos": "normal",
  "working_directory": "/home/alice/simulations",
  "command": "/home/alice/scripts/run_simulation.sh",
  "environment": {
    "PATH": "/usr/local/bin:/usr/bin:/bin",
    "CUDA_VISIBLE_DEVICES": "0,1,2,3"
  },
  "resources": {
    "cpu_efficiency": 0.85,
    "memory_efficiency": 0.67,
    "gpu_utilization": 0.92,
    "max_memory_used": "89GB",
    "max_cpu_time": "127.5h"
  },
  "dependencies": [],
  "array_job_id": null,
  "array_task_id": null
}
```

### Submit Job
```http
POST /v1/jobs
```

**Request Body:**
```json
{
  "job_name": "my_simulation",
  "script": "/home/alice/scripts/simulation.sh",
  "partition": "gpu",
  "nodes": 2,
  "cpus_per_node": 16,
  "memory": "64GB",
  "gpus": 2,
  "time_limit": "4:00:00",
  "account": "research",
  "qos": "normal",
  "working_directory": "/home/alice/simulations",
  "standard_output": "/home/alice/logs/%j.out",
  "standard_error": "/home/alice/logs/%j.err",
  "environment": {
    "CUDA_VISIBLE_DEVICES": "0,1"
  },
  "dependencies": [
    {
      "type": "afterok",
      "job_id": "12344"
    }
  ]
}
```

**Example Request:**
```bash
curl -X POST "https://api.example.com/v1/jobs" \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer TOKEN" \
     -d '{
       "job_name": "test_job",
       "script": "/home/alice/test.sh",
       "nodes": 1,
       "cpus_per_node": 4,
       "time_limit": "1:00:00"
     }'
```

**Example Response:**
```json
{
  "job_id": "12346",
  "state": "PENDING",
  "message": "Job submitted successfully"
}
```

### Cancel Job
```http
DELETE /v1/jobs/{job_id}
```

**Parameters:**
- `signal` (string): Signal to send (default: SIGTERM)
- `force` (boolean): Force cancellation

**Example Request:**
```bash
curl -X DELETE "https://api.example.com/v1/jobs/12345?force=true"
```

**Example Response:**
```json
{
  "job_id": "12345",
  "state": "CANCELLED",
  "message": "Job cancelled successfully"
}
```

### Hold/Release Job
```http
PUT /v1/jobs/{job_id}/hold
PUT /v1/jobs/{job_id}/release
```

**Request Body (Hold):**
```json
{
  "reason": "Debugging required"
}
```

**Example Request:**
```bash
curl -X PUT "https://api.example.com/v1/jobs/12345/hold" \
     -H "Content-Type: application/json" \
     -d '{"reason": "Need to debug"}'
```

### Update Job Priority
```http
PUT /v1/jobs/{job_id}/priority
```

**Request Body:**
```json
{
  "priority": 1500
}
```

### Get Job Output
```http
GET /v1/jobs/{job_id}/output
GET /v1/jobs/{job_id}/error
```

**Parameters:**
- `lines` (integer): Number of lines to retrieve
- `follow` (boolean): Follow output (WebSocket)

## Nodes API

### List Nodes
```http
GET /v1/nodes
```

**Parameters:**
- `state` (string): Filter by node state
- `partition` (string): Filter by partition
- `features` (string): Filter by features
- `available` (boolean): Filter available nodes only

**Example Response:**
```json
{
  "nodes": [
    {
      "node_name": "node001",
      "state": "MIXED",
      "partitions": ["gpu", "normal"],
      "cpu_total": 32,
      "cpu_allocated": 16,
      "cpu_load": 14.2,
      "memory_total": "128GB",
      "memory_allocated": "64GB",
      "memory_free": "58GB",
      "gpus_total": 4,
      "gpus_allocated": 2,
      "features": ["gpu", "cuda", "infiniband"],
      "architecture": "x86_64",
      "os": "Linux 5.4.0-74-generic",
      "boot_time": "2023-12-01T08:30:00Z",
      "last_busy": "2023-12-15T12:45:30Z",
      "jobs": ["12345", "12347"],
      "reason": null
    }
  ],
  "total": 256
}
```

### Get Node Details
```http
GET /v1/nodes/{node_name}
```

### Drain/Resume Node
```http
PUT /v1/nodes/{node_name}/drain
PUT /v1/nodes/{node_name}/resume
```

**Request Body (Drain):**
```json
{
  "reason": "Hardware maintenance",
  "timeout": "2h"
}
```

### Update Node State
```http
PUT /v1/nodes/{node_name}/state
```

**Request Body:**
```json
{
  "state": "DOWN",
  "reason": "Hardware failure"
}
```

## Users API

### List Users
```http
GET /v1/users
```

**Example Response:**
```json
{
  "users": [
    {
      "username": "alice",
      "default_account": "research",
      "admin_level": "none",
      "jobs_running": 15,
      "jobs_pending": 3,
      "cpu_time_used": "1247h",
      "associations": [
        {
          "account": "research",
          "partition": "gpu",
          "qos": ["normal", "high"]
        }
      ]
    }
  ]
}
```

### Get User Details
```http
GET /v1/users/{username}
```

## Partitions API

### List Partitions
```http
GET /v1/partitions
```

**Example Response:**
```json
{
  "partitions": [
    {
      "name": "gpu",
      "state": "UP",
      "nodes_total": 32,
      "nodes_idle": 8,
      "nodes_allocated": 20,
      "nodes_mixed": 4,
      "cpus_total": 1024,
      "cpus_allocated": 756,
      "default_time": "2:00:00",
      "max_time": "7-00:00:00",
      "max_nodes": 16,
      "priority": 1000,
      "qos": "normal",
      "features": ["gpu", "cuda"]
    }
  ]
}
```

## Metrics API

### Cluster Statistics
```http
GET /v1/metrics/cluster
```

**Example Response:**
```json
{
  "timestamp": "2023-12-15T14:30:00Z",
  "cluster_utilization": {
    "cpu_percent": 78.5,
    "memory_percent": 65.2,
    "gpu_percent": 84.7
  },
  "job_counts": {
    "RUNNING": 1247,
    "PENDING": 156,
    "COMPLETED": 45672,
    "FAILED": 234,
    "CANCELLED": 89
  },
  "node_counts": {
    "IDLE": 45,
    "MIXED": 89,
    "ALLOCATED": 112,
    "DOWN": 3,
    "DRAIN": 2
  },
  "queue_stats": {
    "avg_wait_time": "12m 34s",
    "max_wait_time": "2h 15m",
    "throughput": 145.7
  }
}
```

### Performance Metrics
```http
GET /v1/metrics/performance
```

**Parameters:**
- `start_time` (string): Start time (ISO 8601)
- `end_time` (string): End time (ISO 8601)
- `interval` (string): Aggregation interval (1m, 5m, 1h, 1d)

**Example Response:**
```json
{
  "metrics": [
    {
      "timestamp": "2023-12-15T14:00:00Z",
      "cpu_utilization": 76.8,
      "memory_utilization": 63.5,
      "gpu_utilization": 82.1,
      "jobs_submitted": 45,
      "jobs_completed": 38,
      "avg_queue_time": 720
    }
  ],
  "interval": "1h",
  "count": 24
}
```

### Job Efficiency
```http
GET /v1/metrics/efficiency
```

**Parameters:**
- `user` (string): Filter by user
- `partition` (string): Filter by partition
- `min_runtime` (string): Minimum runtime threshold

## Search API

### Global Search
```http
GET /v1/search
```

**Parameters:**
- `q` (string): Search query
- `type` (string): Resource type (jobs, nodes, users)
- `limit` (integer): Maximum results

**Example Request:**
```bash
curl "https://api.example.com/v1/search?q=alice%20gpu&limit=20"
```

**Example Response:**
```json
{
  "results": [
    {
      "type": "job",
      "id": "12345",
      "title": "alice's GPU simulation",
      "description": "Running on gpu partition",
      "url": "/v1/jobs/12345",
      "score": 0.95
    },
    {
      "type": "node",
      "id": "gpu001",
      "title": "GPU Node 001",
      "description": "4x NVIDIA A100 GPUs",
      "url": "/v1/nodes/gpu001",
      "score": 0.82
    }
  ],
  "total": 15,
  "query_time": 0.045
}
```

## Export API

### Export Data
```http
POST /v1/export
```

**Request Body:**
```json
{
  "type": "jobs",
  "format": "csv",
  "filter": {
    "state": "COMPLETED",
    "user": "alice",
    "start_time": "2023-12-01T00:00:00Z",
    "end_time": "2023-12-31T23:59:59Z"
  },
  "fields": ["job_id", "job_name", "user", "state", "runtime"],
  "options": {
    "include_headers": true,
    "delimiter": ",",
    "filename": "alice_december_jobs.csv"
  }
}
```

**Example Response:**
```json
{
  "export_id": "exp_abc123",
  "status": "processing",
  "download_url": null,
  "created_at": "2023-12-15T14:30:00Z",
  "expires_at": "2023-12-16T14:30:00Z"
}
```

### Get Export Status
```http
GET /v1/exports/{export_id}
```

**Example Response:**
```json
{
  "export_id": "exp_abc123",
  "status": "completed",
  "download_url": "https://api.example.com/v1/exports/exp_abc123/download",
  "file_size": 2048576,
  "record_count": 1247,
  "created_at": "2023-12-15T14:30:00Z",
  "completed_at": "2023-12-15T14:32:15Z",
  "expires_at": "2023-12-16T14:30:00Z"
}
```

### Download Export
```http
GET /v1/exports/{export_id}/download
```

## Reports API

### Generate Report
```http
POST /v1/reports
```

**Request Body:**
```json
{
  "type": "utilization",
  "format": "pdf",
  "parameters": {
    "period": "month",
    "start_date": "2023-12-01",
    "end_date": "2023-12-31",
    "partitions": ["gpu", "cpu"],
    "include_charts": true
  },
  "delivery": {
    "method": "email",
    "email": "admin@example.com"
  }
}
```

### List Reports
```http
GET /v1/reports
```

## WebSocket API

### Real-time Updates
```javascript
const ws = new WebSocket('wss://api.example.com/v1/ws');

// Subscribe to job updates
ws.send(JSON.stringify({
  "action": "subscribe",
  "channel": "jobs",
  "filter": {
    "user": "alice",
    "state": "RUNNING"
  }
}));

// Receive updates
ws.onmessage = function(event) {
  const update = JSON.parse(event.data);
  console.log('Job update:', update);
};
```

**Update Message Format:**
```json
{
  "channel": "jobs",
  "action": "update",
  "resource_id": "12345",
  "timestamp": "2023-12-15T14:35:00Z",
  "data": {
    "job_id": "12345",
    "state": "COMPLETED",
    "end_time": "2023-12-15T14:34:45Z",
    "exit_code": 0
  }
}
```

### Available Channels
- `jobs` - Job state changes
- `nodes` - Node state changes
- `metrics` - Performance metrics
- `alerts` - System alerts
- `queue` - Queue statistics

## Batch Operations API

### Bulk Job Operations
```http
POST /v1/jobs/bulk
```

**Request Body:**
```json
{
  "action": "cancel",
  "job_ids": ["12345", "12346", "12347"],
  "options": {
    "force": true,
    "reason": "Emergency maintenance"
  }
}
```

**Example Response:**
```json
{
  "batch_id": "batch_xyz789",
  "results": [
    {
      "job_id": "12345",
      "status": "success",
      "message": "Job cancelled"
    },
    {
      "job_id": "12346",
      "status": "error",
      "message": "Job already completed"
    }
  ],
  "summary": {
    "total": 3,
    "successful": 2,
    "failed": 1
  }
}
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Job 99999 not found",
    "details": {
      "resource_type": "job",
      "resource_id": "99999"
    },
    "request_id": "req_abc123",
    "timestamp": "2023-12-15T14:30:00Z"
  }
}
```

### HTTP Status Codes
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service unavailable

### Common Error Codes
- `INVALID_REQUEST` - Malformed request
- `AUTHENTICATION_FAILED` - Invalid credentials
- `PERMISSION_DENIED` - Insufficient permissions
- `RESOURCE_NOT_FOUND` - Resource doesn't exist
- `RESOURCE_CONFLICT` - Resource state conflict
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `VALIDATION_ERROR` - Request validation failed
- `SLURM_ERROR` - SLURM backend error

## Rate Limiting

### Rate Limit Headers
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 897
X-RateLimit-Reset: 1640995200
X-RateLimit-Window: 3600
```

### Rate Limits
- **Free Tier**: 1,000 requests/hour
- **Pro Tier**: 10,000 requests/hour
- **Enterprise**: Custom limits

## OpenAPI Specification

### Download Specification
```http
GET /v1/openapi.json
GET /v1/openapi.yaml
```

### Interactive Documentation
- Swagger UI: `https://api.example.com/docs`
- ReDoc: `https://api.example.com/redoc`

## SDK Examples

### Python SDK
```python
from s9s_client import S9SClient

# Initialize client
client = S9SClient(
    base_url="https://api.example.com",
    token="your_token_here"
)

# List jobs
jobs = client.jobs.list(state="RUNNING", user="alice")

# Submit job
job = client.jobs.submit(
    name="test_job",
    script="/home/alice/test.sh",
    nodes=2,
    cpus_per_node=16
)

# Get job details
job_details = client.jobs.get(job.job_id)

# Cancel job
client.jobs.cancel(job.job_id)
```

### JavaScript SDK
```javascript
import { S9SClient } from '@s9s/client';

const client = new S9SClient({
  baseUrl: 'https://api.example.com',
  token: 'your_token_here'
});

// List jobs
const jobs = await client.jobs.list({
  state: 'RUNNING',
  user: 'alice'
});

// Submit job
const job = await client.jobs.submit({
  name: 'test_job',
  script: '/home/alice/test.sh',
  nodes: 2,
  cpusPerNode: 16
});
```

### Go SDK
```go
package main

import (
    "github.com/s9s/go-client"
)

func main() {
    client := s9s.NewClient(&s9s.Config{
        BaseURL: "https://api.example.com",
        Token:   "your_token_here",
    })

    // List jobs
    jobs, err := client.Jobs.List(&s9s.JobListOptions{
        State: "RUNNING",
        User:  "alice",
    })

    // Submit job
    job, err := client.Jobs.Submit(&s9s.JobSubmitRequest{
        Name:        "test_job",
        Script:      "/home/alice/test.sh",
        Nodes:       2,
        CPUsPerNode: 16,
    })
}
```

## Testing

### Test Environment
```bash
# Base URL for testing
export S9S_API_URL="https://api-test.example.com"
export S9S_API_TOKEN="test_token_123"
```

### cURL Examples
```bash
# Test authentication
curl -H "Authorization: Bearer $S9S_API_TOKEN" \
     $S9S_API_URL/v1/status

# List jobs
curl -H "Authorization: Bearer $S9S_API_TOKEN" \
     "$S9S_API_URL/v1/jobs?limit=10"

# Submit test job
curl -X POST \
     -H "Authorization: Bearer $S9S_API_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"job_name":"test","script":"/bin/sleep 60","nodes":1}' \
     "$S9S_API_URL/v1/jobs"
```

## Next Steps

- Download SDKs from [GitHub](https://github.com/jontk/s9s-clients)
- Try the interactive API explorer
- Read integration guides for popular tools
- Review [command reference](./commands.md)
- Check [configuration options](./configuration.md)
