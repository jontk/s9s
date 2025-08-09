# Security Policy

## Supported Versions

We actively support the following versions of s9s with security updates:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| latest release | :white_check_mark: |
| < latest | :x: |

## Reporting a Vulnerability

We take the security of s9s seriously. If you believe you have found a security vulnerability in s9s, please report it to us as described below.

### Please do NOT report security vulnerabilities through public GitHub issues.

Instead, please report them via email to: **security@s9s.dev** (or through GitHub Security Advisories if available).

Please include the following information in your report:
- Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

## Response Process

1. **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours.

2. **Investigation**: We will investigate the vulnerability and determine its impact and severity.

3. **Fix Development**: If the vulnerability is confirmed, we will develop a fix.

4. **Disclosure Timeline**: We aim to:
   - Provide an initial response within 48 hours
   - Provide a detailed response within 7 days
   - Release a fix within 30 days for high/critical severity issues
   - Release a fix within 90 days for medium/low severity issues

5. **Credit**: If you would like, we will credit you in our security advisory and release notes.

## Security Best Practices

When using s9s, please follow these security best practices:

### Configuration Security
- Store configuration files with appropriate permissions (600 or 640)
- Use environment variables or secure credential management for sensitive data
- Regularly rotate API tokens and passwords
- Enable TLS for all SLURM connections

### Network Security
- Use HTTPS/TLS for all connections to SLURM REST API
- Consider using VPN or private networks for cluster access
- Implement proper firewall rules
- Monitor network traffic for suspicious activity

### Authentication & Authorization
- Use strong, unique passwords for SLURM accounts
- Implement proper RBAC in your SLURM cluster
- Consider using SSO/LDAP integration
- Regularly audit user access and permissions

### System Security
- Keep s9s updated to the latest version
- Run s9s with minimal required privileges
- Use dedicated service accounts where possible
- Monitor logs for suspicious activity
- Regularly update underlying system packages

### Data Protection
- Be cautious with exported data containing sensitive information
- Use encryption for data at rest and in transit
- Implement proper backup and disaster recovery procedures
- Follow data retention policies

## Known Security Considerations

### Authentication
- s9s inherits the authentication model of your SLURM cluster
- API tokens and passwords are stored in configuration files - protect these files
- Consider the security implications of storing credentials vs. prompting for them

### Network Communication
- All communication with SLURM happens over the network
- Ensure TLS is properly configured and certificates are valid
- Consider the implications of connecting to SLURM clusters over untrusted networks

### Plugin System
- Plugins run with the same privileges as the main s9s application
- Only load plugins from trusted sources
- Review plugin code before deployment in production environments

### Export Functionality
- Exported data may contain sensitive information about your cluster and jobs
- Be careful about where exported files are stored and who has access to them
- Consider encrypting exported data if it contains sensitive information

### Logging and Debugging
- Debug logs may contain sensitive information
- Secure log files and rotate them regularly
- Be careful about sharing debug output when reporting issues

## Security Updates

Security updates will be released as patch versions and communicated through:
- GitHub Security Advisories
- Release notes
- Documentation updates

Subscribe to our releases or watch the repository to stay informed about security updates.

## Contact

For security-related questions or concerns that are not vulnerabilities, you can:
- Open a discussion on GitHub Discussions
- Contact us at security@s9s.dev
- Create a support issue (for general security questions)

## Acknowledgments

We would like to thank the following individuals for responsibly disclosing security vulnerabilities:
- (This section will be updated as we receive reports)

---

This security policy is subject to change. Please check back regularly for updates.