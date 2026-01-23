package discovery

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/security"
)

// ScontrolDiscovery discovers SLURM clusters by parsing scontrol ping output
type ScontrolDiscovery struct {
	scontrolPath string
	timeout      time.Duration
	defaultPort  int
}

// ScontrolResult represents the parsed result from scontrol ping
type ScontrolResult struct {
	Hostname string
	Role     string // "primary" or "backup"
	Status   string // "UP" or "DOWN"
}

// NewScontrolDiscovery creates a new ScontrolDiscovery instance
func NewScontrolDiscovery() DiscoveryMethod {
	// Validate scontrol path, fall back to original if validation fails
	scontrolPath := "scontrol"
	if validated, err := security.ValidateAndResolveCommand(scontrolPath, "slurm"); err == nil {
		scontrolPath = validated
	}

	return &ScontrolDiscovery{
		scontrolPath: scontrolPath,
		timeout:      10 * time.Second,
		defaultPort:  6820,
	}
}

// NewScontrolDiscoveryWithConfig creates a new ScontrolDiscovery with custom configuration
func NewScontrolDiscoveryWithConfig(scontrolPath string, timeout time.Duration, defaultPort int) DiscoveryMethod {
	// Validate scontrol path, fall back to original if validation fails
	if scontrolPath == "" {
		scontrolPath = "scontrol"
	}
	if validated, err := security.ValidateAndResolveCommand(scontrolPath, "slurm"); err == nil {
		scontrolPath = validated
	}

	sd := &ScontrolDiscovery{
		scontrolPath: scontrolPath,
		timeout:      timeout,
		defaultPort:  defaultPort,
	}
	if sd.timeout == 0 {
		sd.timeout = 10 * time.Second
	}
	if sd.defaultPort == 0 {
		sd.defaultPort = 6820
	}
	return sd
}

// Name returns the name of this discovery method
func (sd *ScontrolDiscovery) Name() string {
	return "scontrol"
}

// Priority returns the priority of this discovery method (lower = higher priority)
func (sd *ScontrolDiscovery) Priority() int {
	return 5 // Higher priority than network scan, lower than config file
}

// Discover discovers SLURM clusters using scontrol ping
func (sd *ScontrolDiscovery) Discover(ctx context.Context) ([]*DiscoveredCluster, error) {
	debug.Logger.Printf("Starting scontrol discovery using path: %s", sd.scontrolPath)

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, sd.timeout)
	defer cancel()

	// Execute scontrol ping
	// nolint:gosec // G204: Command path is validated during initialization via security.ValidateAndResolveCommand
	cmd := exec.CommandContext(ctxWithTimeout, sd.scontrolPath, "ping")
	output, err := cmd.Output()
	if err != nil {
		debug.Logger.Printf("scontrol ping failed: %v", err)
		return nil, fmt.Errorf("scontrol ping failed: %w", err)
	}

	// Parse the output
	results, err := sd.parsePingOutput(string(output))
	if err != nil {
		debug.Logger.Printf("Failed to parse scontrol output: %v", err)
		return nil, fmt.Errorf("failed to parse scontrol output: %w", err)
	}

	if len(results) == 0 {
		debug.Logger.Printf("No controllers found in scontrol ping output")
		return nil, nil
	}

	// Convert results to DiscoveredCluster
	var clusters []*DiscoveredCluster
	for _, result := range results {
		cluster := sd.resultToCluster(result)
		if cluster != nil {
			clusters = append(clusters, cluster)
		}
	}

	debug.Logger.Printf("scontrol discovery found %d clusters", len(clusters))
	return clusters, nil
}

// parsePingOutput parses the output of scontrol ping
// Expected format: "Slurmctld(primary) at hostname is UP"
// Alternative format: "Slurmctld(primary/backup) at hostname is UP/DOWN"
func (sd *ScontrolDiscovery) parsePingOutput(output string) ([]ScontrolResult, error) {
	var results []ScontrolResult

	// Regex patterns for different scontrol ping output formats
	// Format 1: "Slurmctld(primary) at hostname is UP"
	// Format 2: "Slurmctld(backup) at hostname is DOWN"
	// Format 3: "Slurmctld at hostname is UP"
	patterns := []*regexp.Regexp{
		// Primary format with role in parentheses
		regexp.MustCompile(`Slurmctld\((\w+)\)\s+at\s+(\S+)\s+is\s+(\w+)`),
		// Alternative format without role
		regexp.MustCompile(`Slurmctld\s+at\s+(\S+)\s+is\s+(\w+)`),
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var matched bool
		for i, pattern := range patterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 0 {
				var result ScontrolResult
				if i == 0 && len(matches) >= 4 {
					// Format with role
					result = ScontrolResult{
						Role:     strings.ToLower(matches[1]),
						Hostname: matches[2],
						Status:   strings.ToUpper(matches[3]),
					}
				} else if i == 1 && len(matches) >= 3 {
					// Format without role
					result = ScontrolResult{
						Role:     "primary",
						Hostname: matches[1],
						Status:   strings.ToUpper(matches[2]),
					}
				}
				if result.Hostname != "" {
					results = append(results, result)
					matched = true
					break
				}
			}
		}

		if !matched {
			debug.Logger.Printf("Could not parse scontrol line: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading scontrol output: %w", err)
	}

	return results, nil
}

// resultToCluster converts a ScontrolResult to a DiscoveredCluster
func (sd *ScontrolDiscovery) resultToCluster(result ScontrolResult) *DiscoveredCluster {
	// Only return clusters that are UP
	if result.Status != "UP" {
		debug.Logger.Printf("Skipping controller %s (status: %s)", result.Hostname, result.Status)
		return nil
	}

	// Build the REST endpoint URL
	endpoint := fmt.Sprintf("http://%s:%d", result.Hostname, sd.defaultPort)

	// Calculate confidence based on role
	confidence := 0.85
	if result.Role == "primary" {
		confidence = 0.9
	}

	cluster := &DiscoveredCluster{
		Name:          fmt.Sprintf("scontrol-%s", result.Hostname),
		Host:          result.Hostname,
		Port:          sd.defaultPort,
		RestEndpoints: []string{endpoint},
		Confidence:    confidence,
		DetectionMethods: []string{
			"scontrol-ping",
		},
		Metadata: map[string]string{
			"source":          "scontrol",
			"controller_role": result.Role,
			"status":          result.Status,
		},
	}

	return cluster
}

// ParseScontrolPingOutput is a utility function to parse scontrol ping output
// This can be used by other components that need to parse scontrol output
func ParseScontrolPingOutput(output string) ([]ScontrolResult, error) {
	sd := &ScontrolDiscovery{}
	return sd.parsePingOutput(output)
}

// GetControllerHostname extracts the primary controller hostname from scontrol ping output
// Returns the hostname and whether it was found
func GetControllerHostname(output string) (string, bool) {
	results, err := ParseScontrolPingOutput(output)
	if err != nil || len(results) == 0 {
		return "", false
	}

	// Prefer primary controller
	for _, result := range results {
		if result.Role == "primary" && result.Status == "UP" {
			return result.Hostname, true
		}
	}

	// Fall back to any UP controller
	for _, result := range results {
		if result.Status == "UP" {
			return result.Hostname, true
		}
	}

	return "", false
}
