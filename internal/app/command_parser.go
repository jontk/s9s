package app

import "strings"

// CommandResult represents the outcome of a command execution
type CommandResult struct {
	Success bool
	Message string
	Error   error
}

// CommandHandler is the new signature supporting arguments
type CommandHandler func(args []string) CommandResult

// CommandDef defines a command with metadata
type CommandDef struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	MinArgs     int
	MaxArgs     int // -1 for unlimited
	Handler     CommandHandler
}

// ParseCommand splits input into command name and arguments
func ParseCommand(input string) (name string, args []string) {
	input = strings.TrimSpace(input)
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	return strings.ToLower(parts[0]), parts[1:]
}
