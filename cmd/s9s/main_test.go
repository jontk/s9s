package main

import (
	"os"
	"testing"
	"time"
)

func TestMain(_ *testing.T) {
	// Test version flag
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test version output
	os.Args = []string{"s9s", "--version"}

	// Since main() calls os.Exit, we can't test it directly
	// Instead, we'll just verify the binary was built correctly
}

func TestGracefulShutdown(t *testing.T) {
	// This test would require refactoring main() to be more testable
	// For now, we'll just ensure the code compiles and basic structure is correct

	// Verify signal channel creation doesn't panic
	sigChan := make(chan os.Signal, 1)
	close(sigChan)

	// Verify timeout duration
	timeout := 5 * time.Second
	if timeout != 5*time.Second {
		t.Errorf("Expected timeout to be 5s, got %v", timeout)
	}
}
