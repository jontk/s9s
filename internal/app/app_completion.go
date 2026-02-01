package app

import (
	"sort"
	"strings"

	"github.com/jontk/s9s/internal/views"
)

// ArgType identifies what kind of argument a command expects
type ArgType int

const (
	ArgTypeNone ArgType = iota
	ArgTypeJobID
	ArgTypeNodeName
)

// getArgType returns the expected argument type for a command
func getArgType(cmdName string) ArgType {
	switch cmdName {
	case "cancel", "hold", "release", "requeue":
		return ArgTypeJobID
	case "drain", "resume":
		return ArgTypeNodeName
	default:
		return ArgTypeNone
	}
}

// getCompletions returns completions for current command line text
func (s *S9s) getCompletions(currentText string) []string {
	// Don't trim - trailing space is significant for argument completion
	text := currentText

	// Return nil for empty input to avoid showing empty dropdown
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// If no space yet, complete command names
	if !strings.Contains(text, " ") {
		return s.getCommandCompletions(strings.TrimSpace(text))
	}

	// Otherwise, complete arguments
	return s.getArgumentCompletions(text)
}

// getCommandCompletions returns matching command names
func (s *S9s) getCommandCompletions(prefix string) []string {
	prefix = strings.ToLower(prefix)
	var completions []string

	for name, cmd := range s.commandRegistry() {
		if strings.HasPrefix(name, prefix) {
			completions = append(completions, name)
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				completions = append(completions, alias)
			}
		}
	}

	sort.Strings(completions)
	return completions
}

// getArgumentCompletions returns matching arguments based on command
func (s *S9s) getArgumentCompletions(text string) []string {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	cmdName := strings.ToLower(parts[0])
	argType := getArgType(cmdName)

	// Get the partial argument being typed (if any)
	var argPrefix string
	if len(parts) > 1 {
		argPrefix = parts[len(parts)-1]
	} else if strings.HasSuffix(text, " ") {
		argPrefix = ""
	} else {
		return nil // Still typing command name
	}

	var candidates []string
	switch argType {
	case ArgTypeJobID:
		candidates = s.getJobIDCandidates()
	case ArgTypeNodeName:
		candidates = s.getNodeNameCandidates()
	default:
		return nil
	}

	// Filter by prefix and format as full completions
	var completions []string
	cmdWithSpace := cmdName + " "
	for _, c := range candidates {
		if strings.HasPrefix(c, argPrefix) {
			completions = append(completions, cmdWithSpace+c)
		}
	}

	sort.Strings(completions)
	return completions
}

// getJobIDCandidates returns job IDs from cached view data
func (s *S9s) getJobIDCandidates() []string {
	view, err := s.viewMgr.GetView("jobs")
	if err != nil {
		return nil
	}
	if jobsView, ok := view.(*views.JobsView); ok {
		return jobsView.GetJobIDs()
	}
	return nil
}

// getNodeNameCandidates returns node names from cached view data
func (s *S9s) getNodeNameCandidates() []string {
	view, err := s.viewMgr.GetView("nodes")
	if err != nil {
		return nil
	}
	if nodesView, ok := view.(*views.NodesView); ok {
		return nodesView.GetNodeNames()
	}
	return nil
}
