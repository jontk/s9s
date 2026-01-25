package streaming

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jontk/s9s/internal/dao"
)

// PathResolver resolves SLURM job output file paths using SLURM API data
type PathResolver struct {
	slurmConfig *SlurmConfig
	client      dao.SlurmClient // Uses SLURM API to get job metadata including file paths
}

// NewPathResolver creates a new path resolver
func NewPathResolver(client dao.SlurmClient, config *SlurmConfig) *PathResolver {
	if config == nil {
		config = DefaultSlurmConfig()
	}

	return &PathResolver{
		slurmConfig: config,
		client:      client,
	}
}

// ResolveOutputPath determines the full path to job output file using SLURM API data
func (pr *PathResolver) ResolveOutputPath(jobID, outputType string) (string, bool, string, error) {
	// Get job information from SLURM API via slurm-client
	// This provides WorkingDir, StdOut, StdErr fields with file paths
	job, err := pr.client.Jobs().Get(jobID)
	if err != nil {
		return "", false, "", fmt.Errorf("failed to get job info from SLURM API: %w", err)
	}

	// Use file paths provided by SLURM API
	var fileName string
	if outputType == "stdout" {
		fileName = pr.resolveStdoutPath(job) // Uses job.StdOut from SLURM API
	} else {
		fileName = pr.resolveStderrPath(job) // Uses job.StdErr from SLURM API
	}

	// Determine if job is running on local or remote node
	isRemote := pr.isRemoteNode(job.NodeList)
	nodeID := pr.extractPrimaryNode(job.NodeList)

	return fileName, isRemote, nodeID, nil
}

// resolveStdoutPath resolves the stdout file path using SLURM API data
func (pr *PathResolver) resolveStdoutPath(job *dao.Job) string {
	// Use stdout path provided by SLURM API via slurm-client
	if job.StdOut != "" && job.StdOut != "/dev/null" {
		return job.StdOut // Direct path from SLURM API
	}

	// Use working directory from SLURM API if available
	if job.WorkingDir != "" {
		return filepath.Join(job.WorkingDir,
			fmt.Sprintf(pr.slurmConfig.FilePattern, job.ID))
	}

	// Fallback to SLURM spool directory
	return filepath.Join(pr.slurmConfig.OutputDir,
		fmt.Sprintf(pr.slurmConfig.FilePattern, job.ID))
}

// resolveStderrPath resolves the stderr file path using SLURM API data
func (pr *PathResolver) resolveStderrPath(job *dao.Job) string {
	// Use stderr path provided by SLURM API via slurm-client
	if job.StdErr != "" && job.StdErr != "/dev/null" {
		return job.StdErr // Direct path from SLURM API
	}

	// Use working directory from SLURM API if available
	if job.WorkingDir != "" {
		return filepath.Join(job.WorkingDir,
			fmt.Sprintf(pr.slurmConfig.ErrorPattern, job.ID))
	}

	// Fallback to SLURM spool directory
	return filepath.Join(pr.slurmConfig.ErrorDir,
		fmt.Sprintf(pr.slurmConfig.ErrorPattern, job.ID))
}

// isRemoteNode determines if the job is running on remote nodes
func (pr *PathResolver) isRemoteNode(nodeList string) bool {
	if !pr.slurmConfig.RemoteAccess || nodeList == "" {
		return false
	}

	// Extract nodes from node list (format: "node1,node2" or "node[1-3]")
	nodes := pr.parseNodeList(nodeList)

	// Check if any node is not in the local nodes list
	for _, node := range nodes {
		if !pr.isLocalNode(node) {
			return true
		}
	}

	return false
}

// isLocalNode checks if a node is in the local nodes list
func (pr *PathResolver) isLocalNode(nodeName string) bool {
	for _, localNode := range pr.slurmConfig.LocalNodes {
		if localNode == nodeName {
			return true
		}
	}
	return false
}

// extractPrimaryNode extracts the primary node from a node list
func (pr *PathResolver) extractPrimaryNode(nodeList string) string {
	if nodeList == "" {
		return ""
	}

	nodes := pr.parseNodeList(nodeList)
	if len(nodes) > 0 {
		return nodes[0] // Return first node as primary
	}

	return ""
}

// parseNodeList parses SLURM node list format into individual node names
func (pr *PathResolver) parseNodeList(nodeList string) []string {
	if nodeList == "" {
		return []string{}
	}

	var nodes []string

	// Handle comma-separated nodes
	parts := strings.Split(nodeList, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle range notation like "node[1-3]"
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			rangeNodes := pr.expandNodeRange(part)
			nodes = append(nodes, rangeNodes...)
		} else {
			nodes = append(nodes, part)
		}
	}

	return nodes
}

// expandNodeRange expands SLURM node range notation like "node[1-3]" to ["node1", "node2", "node3"]
func (pr *PathResolver) expandNodeRange(nodeRange string) []string {
	// Find bracket positions
	startBracket := strings.Index(nodeRange, "[")
	endBracket := strings.Index(nodeRange, "]")

	if !pr.isValidNodeRangeBrackets(startBracket, endBracket) {
		return []string{nodeRange}
	}

	prefix := nodeRange[:startBracket]
	rangeStr := nodeRange[startBracket+1 : endBracket]

	// Try different parsing strategies
	if result := pr.parseNumericRange(prefix, rangeStr); len(result) > 0 {
		return result
	}

	if result := pr.parseCommaSeparatedRange(prefix, rangeStr); len(result) > 0 {
		return result
	}

	return pr.parseSingleValueRange(prefix, rangeStr)
}

// isValidNodeRangeBrackets checks if bracket positions are valid
func (pr *PathResolver) isValidNodeRangeBrackets(startBracket, endBracket int) bool {
	return startBracket != -1 && endBracket != -1 && startBracket < endBracket
}

// parseNumericRange handles numeric ranges like "1-3"
func (pr *PathResolver) parseNumericRange(prefix, rangeStr string) []string {
	if !strings.Contains(rangeStr, "-") {
		return nil
	}

	rangeParts := strings.Split(rangeStr, "-")
	if len(rangeParts) != 2 {
		return nil
	}

	var start, end int
	n1, err1 := fmt.Sscanf(rangeParts[0], "%d", &start)
	n2, err2 := fmt.Sscanf(rangeParts[1], "%d", &end)

	if n1 != 1 || err1 != nil || n2 != 1 || err2 != nil {
		return nil
	}

	var nodes []string
	for i := start; i <= end; i++ {
		nodes = append(nodes, fmt.Sprintf("%s%d", prefix, i))
	}
	return nodes
}

// parseCommaSeparatedRange handles comma-separated values like "1,3,5"
func (pr *PathResolver) parseCommaSeparatedRange(prefix, rangeStr string) []string {
	if !strings.Contains(rangeStr, ",") {
		return nil
	}

	values := strings.Split(rangeStr, ",")
	nodes := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		nodes = append(nodes, prefix+value)
	}
	return nodes
}

// parseSingleValueRange handles single values like "node[1]"
func (pr *PathResolver) parseSingleValueRange(prefix, rangeStr string) []string {
	return []string{prefix + rangeStr}
}

// ValidateOutputPath checks if an output path is accessible
func (pr *PathResolver) ValidateOutputPath(filePath string, _ bool, _ string) error {
	if filePath == "" {
		return fmt.Errorf("empty file path")
	}

	if filePath == "/dev/null" {
		return fmt.Errorf("output redirected to /dev/null")
	}

	// Additional validation could include:
	// - Check file permissions (for local files)
	// - Check SSH connectivity (for remote files)
	// - Check if file exists or can be created

	return nil
}

// GetJobOutputPaths returns both stdout and stderr paths for a job
func (pr *PathResolver) GetJobOutputPaths(jobID string) (stdoutPath, stderrPath string, isRemote bool, nodeID string, err error) {
	stdoutPath, isRemoteStdout, nodeIDStdout, err := pr.ResolveOutputPath(jobID, "stdout")
	if err != nil {
		return "", "", false, "", fmt.Errorf("failed to resolve stdout path: %w", err)
	}

	stderrPath, isRemoteStderr, nodeIDStderr, err := pr.ResolveOutputPath(jobID, "stderr")
	if err != nil {
		return "", "", false, "", fmt.Errorf("failed to resolve stderr path: %w", err)
	}

	// For simplicity, use stdout settings if they differ
	// In practice, stdout and stderr should be on the same node
	isRemote = isRemoteStdout || isRemoteStderr
	nodeID = nodeIDStdout
	if nodeIDStdout == "" {
		nodeID = nodeIDStderr
	}

	return stdoutPath, stderrPath, isRemote, nodeID, nil
}
