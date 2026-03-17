# Enterprise Features

s9s is designed as a terminal UI interface for SLURM. For enterprise requirements, s9s leverages SLURM's native enterprise capabilities rather than reimplementing them.

## Authentication

s9s relies on SLURM's native authentication mechanisms. All authentication and authorization is handled by SLURM directly — s9s respects whatever auth configuration your cluster uses.

See [Configuration Guide](../getting-started/configuration.md) for connection setup.

## Enterprise Capabilities via SLURM

For enterprise requirements beyond authentication, s9s relies on SLURM's native capabilities:

### Security & Access Control

**SLURM provides:**
- Multi-Factor Authentication (MFA) via PAM integration
- Pluggable Authentication Modules (PAM)
- Account-based access control
- Job submission policies and limits
- Resource access restrictions

**s9s integration:**
- s9s respects SLURM's authentication and authorization
- All operations are subject to SLURM's security policies
- User permissions are enforced by SLURM

### High Availability & Scalability

**SLURM provides:**
- Controller failover and redundancy
- Distributed architecture
- Multi-cluster federation
- Database redundancy (MySQL, MariaDB with replication)

**s9s integration:**
- Supports connections to highly available SLURM endpoints
- Can be configured with multiple cluster contexts
- No single point of failure when SLURM is configured for HA

### Multi-Tenancy & Resource Management

**SLURM provides:**
- Account hierarchies for organizational structure
- Fair-share scheduling across accounts/users
- Resource quotas and limits per account
- QoS (Quality of Service) policies
- Partition-based resource isolation

**s9s integration:**
- Full visibility into account hierarchies
- QoS and partition management views
- User and account resource tracking
- Reservation management

### Audit & Compliance

**SLURM provides:**
- Complete job accounting database
- Detailed audit logs
- Resource usage tracking
- Job history and provenance

**s9s integration:**
- Export job data to CSV/JSON for compliance reporting
- Real-time monitoring of resource usage
- Historical job data access

### Monitoring & Observability

**s9s provides:**
- Real-time cluster monitoring
- Job and node status visibility
- Resource utilization metrics
- Optional observability plugin for Prometheus integration

**See:** [Observability Plugin](../plugins/observability.md)

## Configuration for Enterprise Environments

### Multiple Clusters

Configure multiple SLURM clusters:

```yaml
defaultCluster: production

clusters:
  - name: production
    cluster:
      endpoint: "https://prod-slurm.example.com:6820"
      token: "${SLURM_JWT}"
      apiVersion: v0.0.44

  - name: development
    cluster:
      endpoint: "https://dev-slurm.example.com:6820"
      token: "${SLURM_DEV_JWT}"
      apiVersion: v0.0.43

  - name: research
    cluster:
      endpoint: "https://research-slurm.example.com:6820"
      token: "${SLURM_RESEARCH_JWT}"
      apiVersion: v0.0.44
```

Switch between clusters:

```bash
s9s --cluster production
s9s --cluster development
```

### TLS Configuration

For secure communication with SLURM REST API:

```yaml
clusters:
  - name: production
    cluster:
      endpoint: "https://slurm.example.com:6820"
      insecure: false  # Enforce TLS certificate validation
      timeout: 30s
```

## Deployment Considerations

### Container Deployment

s9s can be deployed in containerized environments:

```dockerfile
FROM alpine:latest
COPY s9s /usr/local/bin/s9s
RUN chmod +x /usr/local/bin/s9s

# s9s is an interactive TUI application
ENTRYPOINT ["/usr/local/bin/s9s"]
```

> **Note:** s9s is a terminal UI application and does not have a non-interactive `jobs --format json` mode. For non-interactive SLURM data access, use `scontrol` or `sacct` directly, or the SLURM REST API.

### SSH Integration

For direct node access in enterprise environments:

```yaml
ssh:
  enabled: true
  multiplexing: true
  control_path: "/tmp/s9s-ssh-%r@%h:%p"
```

See [SSH Integration Guide](../guides/ssh-integration.md) for details.

## Future Development

Additional enterprise features are under consideration:

- Advanced backup and recovery capabilities
- Extended API integrations
- Enhanced multi-cluster management

For feature requests or to discuss enterprise requirements, please [open a discussion](https://github.com/jontk/s9s/discussions) or [file an issue](https://github.com/jontk/s9s/issues).

## Support

- **Community Support**: [GitHub Discussions](https://github.com/jontk/s9s/discussions)
- **Bug Reports**: [GitHub Issues](https://github.com/jontk/s9s/issues)
- **Contributing**: [Development Guide](../development/contributing.md)

## Resources

- [SLURM Security Guide](https://slurm.schedmd.com/quickstart_admin.html#security)
- [SLURM High Availability](https://slurm.schedmd.com/quickstart_admin.html#HA)
- [SLURM Accounting](https://slurm.schedmd.com/accounting.html)
- [SLURM Multi-Cluster](https://slurm.schedmd.com/multi_cluster.html)
