package app

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantArgs []string
	}{
		{
			name:     "simple command",
			input:    "quit",
			wantName: "quit",
			wantArgs: []string{},
		},
		{
			name:     "command with single arg",
			input:    "cancel 12345",
			wantName: "cancel",
			wantArgs: []string{"12345"},
		},
		{
			name:     "command with multiple args",
			input:    "drain node1 maintenance window",
			wantName: "drain",
			wantArgs: []string{"node1", "maintenance", "window"},
		},
		{
			name:     "command with leading/trailing spaces",
			input:    "  hold 999  ",
			wantName: "hold",
			wantArgs: []string{"999"},
		},
		{
			name:     "empty input",
			input:    "",
			wantName: "",
			wantArgs: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			wantName: "",
			wantArgs: nil,
		},
		{
			name:     "uppercase command",
			input:    "QUIT",
			wantName: "quit",
			wantArgs: []string{},
		},
		{
			name:     "mixed case command with args",
			input:    "Cancel 12345",
			wantName: "cancel",
			wantArgs: []string{"12345"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotArgs := ParseCommand(tt.input)
			if gotName != tt.wantName {
				t.Errorf("ParseCommand() gotName = %v, want %v", gotName, tt.wantName)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("ParseCommand() gotArgs = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}
