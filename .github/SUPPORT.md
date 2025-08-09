# Support

Welcome to s9s support! We're here to help you get the most out of s9s for managing your SLURM clusters.

## Getting Help

### 1. Documentation
Start with our comprehensive documentation:
- [README](../README.md) - Quick start and overview
- [Configuration Guide](../docs/CONFIGURATION.md) - Detailed configuration options
- [User Guide](../docs/USER_GUIDE.md) - Complete usage instructions
- [API Documentation](../docs/API.md) - API reference
- [Architecture Guide](../docs/ARCHITECTURE.md) - Technical architecture
- [Development Guide](../docs/DEVELOPMENT.md) - Contributing and development
- [Performance Analysis](../docs/PERFORMANCE_ANALYSIS.md) - Performance optimization
- [Streaming Guide](../docs/STREAMING_GUIDE.md) - Real-time log streaming

### 2. Community Support
- **GitHub Discussions**: Ask questions, share ideas, and get community help
- **GitHub Issues**: Report bugs or request features using our issue templates
- **Stack Overflow**: Tag your questions with `s9s` and `slurm`

### 3. Self-Help Resources

#### Quick Troubleshooting
1. **Update to the latest version**: `s9s --version` to check your version
2. **Check configuration**: `s9s --validate-config` to verify your config
3. **Enable debug logging**: Run with `--debug` flag for verbose output
4. **Try mock mode**: Use `--mock` to test s9s without a real SLURM cluster
5. **Check SLURM connectivity**: Verify you can access SLURM REST API directly

#### Common Issues

##### Connection Problems
- Verify SLURM REST API is running and accessible
- Check firewall rules and network connectivity
- Validate SSL certificates if using HTTPS
- Confirm API version compatibility

##### Authentication Issues
- Verify SLURM credentials are correct
- Check token expiration if using JWT tokens
- Ensure proper permissions in SLURM cluster
- Review SLURM authentication configuration

##### Performance Issues
- Check system resources (CPU, memory)
- Adjust refresh intervals in configuration
- Enable performance monitoring with `--profile`
- Review [Performance Analysis](../docs/PERFORMANCE_ANALYSIS.md) guide

### 4. Professional Support

#### Community Support (Free)
- GitHub Issues and Discussions
- Community-driven documentation improvements
- Best-effort response times
- No SLA guarantees

#### Enterprise Support (Paid)
For organizations requiring dedicated support:
- **Priority Support**: Guaranteed response times
- **Professional Services**: Custom integrations and features
- **Training Programs**: Comprehensive user and admin training
- **24/7 Support**: Round-the-clock technical support
- **Dedicated Success Manager**: Assigned customer success representative

Contact: enterprise-support@s9s.dev

## Reporting Issues

### Before Reporting
1. Search existing issues to avoid duplicates
2. Update to the latest version
3. Review relevant documentation
4. Collect debug information

### Issue Types

#### Bug Reports
Use the [Bug Report template](.github/ISSUE_TEMPLATE/bug_report.yml)
- Include s9s version, OS, and SLURM version
- Provide clear reproduction steps
- Include relevant logs and configuration (redact sensitive data)

#### Feature Requests
Use the [Feature Request template](.github/ISSUE_TEMPLATE/feature_request.yml)
- Describe the problem you're trying to solve
- Explain your proposed solution
- Include use cases and examples

#### Questions
Use the [Question template](.github/ISSUE_TEMPLATE/question.yml)
- Check documentation first
- Provide context about your environment
- Describe what you've already tried

### Information to Include
Always include:
- s9s version: `s9s --version`
- Operating system and version
- SLURM version and configuration
- Relevant configuration files (redacted)
- Error messages and logs
- Steps to reproduce the issue

## Response Times

### Community Support
- **Bug Reports**: We aim to respond within 3-5 business days
- **Feature Requests**: Initial response within 1 week
- **Questions**: Community-driven, typically within 24-48 hours

### Enterprise Support
- **Critical Issues**: 4 hours
- **High Priority**: 24 hours
- **Medium Priority**: 72 hours
- **Low Priority**: 1 week

## Contributing Back

Help improve s9s for everyone:
- **Documentation**: Improve guides and examples
- **Bug Fixes**: Submit pull requests for issues
- **Features**: Implement new functionality
- **Testing**: Help test new releases and features
- **Community**: Answer questions and help other users

See our [Contributing Guide](../CONTRIBUTING.md) for details.

## Security Issues

**Do not report security vulnerabilities through public issues.**

For security-related issues, please:
- Email: security@s9s.dev
- Use GitHub Security Advisories if available
- Follow our [Security Policy](.github/SECURITY.md)

## Resources

### Official Resources
- **GitHub Repository**: https://github.com/jontk/s9s
- **Documentation**: https://github.com/jontk/s9s/tree/main/docs
- **Releases**: https://github.com/jontk/s9s/releases

### Community Resources
- **Examples Repository**: Community-contributed examples and configurations
- **Plugin Registry**: Directory of available s9s plugins
- **User Guides**: Community-written tutorials and guides

### Related Projects
- **SLURM**: https://slurm.schedmd.com/
- **SLURM REST API**: https://slurm.schedmd.com/rest.html
- **Go Terminal UI**: https://github.com/rivo/tview

## Feedback

We value your feedback! Help us improve s9s:
- **Feature Suggestions**: What would make s9s more useful for you?
- **Documentation Improvements**: What's unclear or missing?
- **User Experience**: How can we make s9s easier to use?
- **Performance**: What performance issues have you encountered?

Share feedback through:
- GitHub Discussions
- Feature requests
- Direct email: feedback@s9s.dev

---

Thank you for using s9s! We're committed to providing excellent support and continuously improving the tool based on your needs and feedback.