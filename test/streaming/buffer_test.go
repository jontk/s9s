package streaming_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jontk/s9s/internal/streaming"
)

func TestCircularBuffer_Basic(t *testing.T) {
	buffer := streaming.NewCircularBuffer(5)

	// Test initial state
	if buffer.Size() != 0 {
		t.Errorf("Expected size 0, got %d", buffer.Size())
	}

	if buffer.Capacity() != 5 {
		t.Errorf("Expected capacity 5, got %d", buffer.Capacity())
	}

	// Test appending lines
	lines := []string{"line1", "line2", "line3"}
	buffer.Append(lines)

	if buffer.Size() != 3 {
		t.Errorf("Expected size 3, got %d", buffer.Size())
	}

	// Test getting lines
	result := buffer.GetLines()
	if len(result) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(result))
	}

	for i, line := range lines {
		if result[i] != line {
			t.Errorf("Expected line %d to be %s, got %s", i, line, result[i])
		}
	}
}

func TestCircularBuffer_Overflow(t *testing.T) {
	buffer := streaming.NewCircularBuffer(3)

	// Add more lines than capacity
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	buffer.Append(lines)

	// Should only keep last 3 lines
	if buffer.Size() != 3 {
		t.Errorf("Expected size 3, got %d", buffer.Size())
	}

	if !buffer.IsFull() {
		t.Error("Expected buffer to be full")
	}

	result := buffer.GetLines()
	expected := []string{"line3", "line4", "line5"}

	for i, expectedLine := range expected {
		if result[i] != expectedLine {
			t.Errorf("Expected line %d to be %s, got %s", i, expectedLine, result[i])
		}
	}

	// Check total lines counter
	if buffer.TotalLines() != 5 {
		t.Errorf("Expected total lines 5, got %d", buffer.TotalLines())
	}
}

func TestCircularBuffer_AppendString(t *testing.T) {
	buffer := streaming.NewCircularBuffer(10)

	// Test appending string with multiple lines
	content := "line1\nline2\nline3\n"
	buffer.AppendString(content)

	if buffer.Size() != 3 {
		t.Errorf("Expected size 3, got %d", buffer.Size())
	}

	result := buffer.GetLines()
	expected := []string{"line1", "line2", "line3"}

	for i, expectedLine := range expected {
		if result[i] != expectedLine {
			t.Errorf("Expected line %d to be %s, got %s", i, expectedLine, result[i])
		}
	}
}

func TestCircularBuffer_GetLastN(t *testing.T) {
	buffer := streaming.NewCircularBuffer(10)

	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	buffer.Append(lines)

	// Test getting last N lines
	last3 := buffer.GetLastN(3)
	expected := []string{"line3", "line4", "line5"}

	if len(last3) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(last3))
	}

	for i, expectedLine := range expected {
		if last3[i] != expectedLine {
			t.Errorf("Expected line %d to be %s, got %s", i, expectedLine, last3[i])
		}
	}

	// Test getting more than available
	all := buffer.GetLastN(10)
	if len(all) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(all))
	}

	// Test getting 0
	none := buffer.GetLastN(0)
	if len(none) != 0 {
		t.Errorf("Expected 0 lines, got %d", len(none))
	}
}

func TestCircularBuffer_Clear(t *testing.T) {
	buffer := streaming.NewCircularBuffer(5)

	lines := []string{"line1", "line2", "line3"}
	buffer.Append(lines)

	buffer.Clear()

	if buffer.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", buffer.Size())
	}

	if buffer.IsFull() {
		t.Error("Expected buffer to not be full after clear")
	}

	// Total lines should remain (lifetime statistic)
	if buffer.TotalLines() != 3 {
		t.Errorf("Expected total lines to remain 3, got %d", buffer.TotalLines())
	}
}

func TestCircularBuffer_Resize(t *testing.T) {
	buffer := streaming.NewCircularBuffer(3)

	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	buffer.Append(lines)

	// Should have last 3 lines
	if buffer.Size() != 3 {
		t.Errorf("Expected size 3, got %d", buffer.Size())
	}

	// Resize to larger capacity
	buffer.Resize(7)
	if buffer.Capacity() != 7 {
		t.Errorf("Expected capacity 7, got %d", buffer.Capacity())
	}

	// Should still have the same lines
	if buffer.Size() != 3 {
		t.Errorf("Expected size 3 after resize, got %d", buffer.Size())
	}

	result := buffer.GetLines()
	expected := []string{"line3", "line4", "line5"}

	for i, expectedLine := range expected {
		if result[i] != expectedLine {
			t.Errorf("Expected line %d to be %s, got %s", i, expectedLine, result[i])
		}
	}

	// Resize to smaller capacity
	buffer.Resize(2)
	if buffer.Capacity() != 2 {
		t.Errorf("Expected capacity 2, got %d", buffer.Capacity())
	}

	// Should keep most recent lines
	if buffer.Size() != 2 {
		t.Errorf("Expected size 2 after resize, got %d", buffer.Size())
	}

	result = buffer.GetLines()
	expected = []string{"line4", "line5"}

	for i, expectedLine := range expected {
		if result[i] != expectedLine {
			t.Errorf("Expected line %d to be %s, got %s", i, expectedLine, result[i])
		}
	}
}

func TestCircularBuffer_Stats(t *testing.T) {
	buffer := streaming.NewCircularBuffer(5)

	lines := []string{"line1", "line2", "line3"}
	buffer.Append(lines)

	stats := buffer.GetStats()

	if stats.CurrentSize != 3 {
		t.Errorf("Expected current size 3, got %d", stats.CurrentSize)
	}

	if stats.Capacity != 5 {
		t.Errorf("Expected capacity 5, got %d", stats.Capacity)
	}

	if stats.TotalLines != 3 {
		t.Errorf("Expected total lines 3, got %d", stats.TotalLines)
	}

	if stats.IsFull {
		t.Error("Expected buffer not to be full")
	}

	expectedUsage := 60.0 // 3/5 * 100
	if stats.UsagePercent != expectedUsage {
		t.Errorf("Expected usage %.1f%%, got %.1f%%", expectedUsage, stats.UsagePercent)
	}
}

func TestCircularBuffer_MemoryEstimate(t *testing.T) {
	buffer := streaming.NewCircularBuffer(5)

	lines := []string{"short", "medium line", "this is a longer line"}
	buffer.Append(lines)

	memoryUsage := buffer.EstimateMemoryUsage()

	// Should be greater than the sum of string lengths
	totalLength := 0
	for _, line := range lines {
		totalLength += len(line)
	}

	if memoryUsage <= int64(totalLength) {
		t.Errorf("Expected memory usage > %d, got %d", totalLength, memoryUsage)
	}
}

func TestCircularBuffer_ThreadSafety(t *testing.T) {
	buffer := streaming.NewCircularBuffer(100)

	var wg sync.WaitGroup
	writers := 5
	readsPerWriter := 20

	// Start multiple writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < readsPerWriter; j++ {
				line := fmt.Sprintf("writer%d-line%d", writerID, j)
				buffer.AppendString(line + "\n")
			}
		}(i)
	}

	// Start multiple readers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < readsPerWriter; j++ {
				_ = buffer.GetLines()
				_ = buffer.GetLastN(10)
				_ = buffer.GetStats()
			}
		}()
	}

	wg.Wait()

	// Verify final state
	if buffer.Size() > buffer.Capacity() {
		t.Errorf("Buffer size %d exceeds capacity %d", buffer.Size(), buffer.Capacity())
	}

	totalExpected := int64(writers * readsPerWriter)
	if buffer.TotalLines() != totalExpected {
		t.Errorf("Expected total lines %d, got %d", totalExpected, buffer.TotalLines())
	}
}

func TestCircularBuffer_EdgeCases(t *testing.T) {
	// Test zero capacity
	buffer := streaming.NewCircularBuffer(0)
	if buffer.Capacity() != streaming.DefaultBufferSize {
		t.Errorf("Expected default capacity %d for zero input, got %d", streaming.DefaultBufferSize, buffer.Capacity())
	}

	// Test negative capacity
	buffer = streaming.NewCircularBuffer(-5)
	if buffer.Capacity() != streaming.DefaultBufferSize {
		t.Errorf("Expected default capacity %d for negative input, got %d", streaming.DefaultBufferSize, buffer.Capacity())
	}

	// Test empty string append
	buffer = streaming.NewCircularBuffer(5)
	buffer.AppendString("")
	if buffer.Size() != 0 {
		t.Errorf("Expected size 0 after appending empty string, got %d", buffer.Size())
	}

	// Test string with only newlines
	buffer.AppendString("\n\n\n")
	if buffer.Size() != 3 {
		t.Errorf("Expected size 3 after appending newlines, got %d", buffer.Size())
	}

	lines := buffer.GetLines()
	for i, line := range lines {
		if line != "" {
			t.Errorf("Expected empty line %d, got '%s'", i, line)
		}
	}
}
