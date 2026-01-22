package streaming

import (
	"strings"
	"sync"
)

// CircularBuffer implements a thread-safe circular buffer for storing job output lines
type CircularBuffer struct {
	data       []string // Circular buffer storage
	head       int      // Current write position
	tail       int      // Current read position
	size       int      // Current number of elements
	capacity   int      // Maximum capacity
	totalLines int64    // Total lines ever added (for statistics)
	mu         sync.RWMutex
}

// NewCircularBuffer creates a new circular buffer with the specified capacity
func NewCircularBuffer(capacity int) *CircularBuffer {
	if capacity <= 0 {
		capacity = DefaultBufferSize
	}

	return &CircularBuffer{
		data:       make([]string, capacity),
		head:       0,
		tail:       0,
		size:       0,
		capacity:   capacity,
		totalLines: 0,
	}
}

// Append adds new lines to the buffer
func (cb *CircularBuffer) Append(lines []string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for _, line := range lines {
		cb.appendSingle(line)
	}
}

// AppendString adds a single string (potentially containing multiple lines) to the buffer
func (cb *CircularBuffer) AppendString(content string) {
	if content == "" {
		return
	}

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Remove empty last line if content ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	cb.Append(lines)
}

// appendSingle adds a single line to the buffer (internal, assumes lock held)
func (cb *CircularBuffer) appendSingle(line string) {
	// Add to current head position
	cb.data[cb.head] = line
	cb.head = (cb.head + 1) % cb.capacity
	cb.totalLines++

	// If buffer is full, advance tail
	if cb.size == cb.capacity {
		cb.tail = (cb.tail + 1) % cb.capacity
	} else {
		cb.size++
	}
}

// GetLines returns all current lines in the buffer
func (cb *CircularBuffer) GetLines() []string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.size == 0 {
		return []string{}
	}

	result := make([]string, cb.size)

	// Copy from tail to end of buffer
	if cb.tail < cb.head || cb.size < cb.capacity {
		// Simple case: no wraparound or buffer not full
		copy(result, cb.data[cb.tail:cb.tail+cb.size])
	} else {
		// Wraparound case: copy from tail to end, then from start to head
		firstPart := cb.capacity - cb.tail
		copy(result[:firstPart], cb.data[cb.tail:])
		copy(result[firstPart:], cb.data[:cb.head])
	}

	return result
}

// GetLastN returns the last N lines from the buffer
func (cb *CircularBuffer) GetLastN(n int) []string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.size == 0 || n <= 0 {
		return []string{}
	}

	// Limit n to available lines
	if n > cb.size {
		n = cb.size
	}

	result := make([]string, n)

	// Calculate starting position
	startPos := cb.size - n
	actualTail := (cb.tail + startPos) % cb.capacity

	// Copy the requested lines
	if actualTail < cb.head || (actualTail >= cb.tail && cb.size < cb.capacity) {
		// No wraparound case
		copy(result, cb.data[actualTail:actualTail+n])
	} else {
		// Wraparound case
		firstPart := cb.capacity - actualTail
		if firstPart >= n {
			copy(result, cb.data[actualTail:actualTail+n])
		} else {
			copy(result[:firstPart], cb.data[actualTail:])
			copy(result[firstPart:], cb.data[:n-firstPart])
		}
	}

	return result
}

// Size returns the current number of lines in the buffer
func (cb *CircularBuffer) Size() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.size
}

// Capacity returns the maximum capacity of the buffer
func (cb *CircularBuffer) Capacity() int {
	return cb.capacity
}

// TotalLines returns the total number of lines ever added to the buffer
func (cb *CircularBuffer) TotalLines() int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.totalLines
}

// IsFull returns true if the buffer is at capacity
func (cb *CircularBuffer) IsFull() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.size == cb.capacity
}

// Clear removes all lines from the buffer
func (cb *CircularBuffer) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.head = 0
	cb.tail = 0
	cb.size = 0
	// Note: we don't reset totalLines as it's a lifetime statistic
}

// GetStats returns buffer statistics
func (cb *CircularBuffer) GetStats() BufferStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return BufferStats{
		CurrentSize:  cb.size,
		Capacity:     cb.capacity,
		TotalLines:   cb.totalLines,
		IsFull:       cb.size == cb.capacity,
		UsagePercent: float64(cb.size) / float64(cb.capacity) * 100,
	}
}

// BufferStats contains statistics about a circular buffer
type BufferStats struct {
	CurrentSize  int     `json:"current_size"`
	Capacity     int     `json:"capacity"`
	TotalLines   int64   `json:"total_lines"`
	IsFull       bool    `json:"is_full"`
	UsagePercent float64 `json:"usage_percent"`
}

// EstimateMemoryUsage estimates the memory usage of the buffer in bytes
func (cb *CircularBuffer) EstimateMemoryUsage() int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var totalSize int64

	// Estimate string memory usage
	for i := 0; i < cb.size; i++ {
		pos := (cb.tail + i) % cb.capacity
		totalSize += int64(len(cb.data[pos])) + 16 // 16 bytes overhead per string
	}

	// Add slice overhead
	totalSize += int64(cb.capacity) * 8 // 8 bytes per string pointer

	return totalSize
}

// Resize changes the buffer capacity (creates a new buffer and copies data)
func (cb *CircularBuffer) Resize(newCapacity int) {
	if newCapacity <= 0 {
		newCapacity = DefaultBufferSize
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if newCapacity == cb.capacity {
		return // No change needed
	}

	// Get current lines
	currentLines := make([]string, cb.size)
	for i := 0; i < cb.size; i++ {
		pos := (cb.tail + i) % cb.capacity
		currentLines[i] = cb.data[pos]
	}

	// Create new buffer
	cb.data = make([]string, newCapacity)
	cb.capacity = newCapacity
	cb.head = 0
	cb.tail = 0
	cb.size = 0

	// Re-insert lines (may truncate if new capacity is smaller)
	linesToKeep := len(currentLines)
	if linesToKeep > newCapacity {
		// Keep the most recent lines
		currentLines = currentLines[linesToKeep-newCapacity:]
	}

	for i, line := range currentLines {
		if i >= newCapacity {
			break
		}
		cb.data[i] = line
		cb.size++
	}
	cb.head = cb.size % cb.capacity
}
