package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestStreamingConfig(t *testing.T) {
	config := DefaultStreamingConfig()

	if config.ChunkSize != 1000 {
		t.Errorf("Expected ChunkSize 1000, got: %d", config.ChunkSize)
	}
	if config.BufferSize != 64*1024 {
		t.Errorf("Expected BufferSize 64KB, got: %d", config.BufferSize)
	}
	if config.ReadTimeout != 10*time.Second {
		t.Errorf("Expected ReadTimeout 10s, got: %v", config.ReadTimeout)
	}
	if config.WriteTimeout != 5*time.Second {
		t.Errorf("Expected WriteTimeout 5s, got: %v", config.WriteTimeout)
	}
}

func TestChunkValues(t *testing.T) {
	client := &Client{}

	// Create test data
	values := make([]SamplePair, 2500) // 2500 data points
	for i := range values {
		values[i] = SamplePair{
			json.Number(fmt.Sprintf("%d", time.Now().Unix()+int64(i))),
			json.Number(fmt.Sprintf("%.2f", float64(i)*1.5)),
		}
	}

	// Test chunking with size 1000
	chunks := client.chunkValues(values, 1000)

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got: %d", len(chunks))
	}

	if len(chunks[0]) != 1000 {
		t.Errorf("Expected first chunk size 1000, got: %d", len(chunks[0]))
	}

	if len(chunks[1]) != 1000 {
		t.Errorf("Expected second chunk size 1000, got: %d", len(chunks[1]))
	}

	if len(chunks[2]) != 500 {
		t.Errorf("Expected third chunk size 500, got: %d", len(chunks[2]))
	}

	// Verify all data is preserved
	totalValues := 0
	for _, chunk := range chunks {
		totalValues += len(chunk)
	}

	if totalValues != len(values) {
		t.Errorf("Expected total values %d, got: %d", len(values), totalValues)
	}
}

func TestChunkValuesEdgeCases(t *testing.T) {
	client := &Client{}

	// Test empty values
	chunks := client.chunkValues([]SamplePair{}, 1000)
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty values, got: %d", len(chunks))
	}

	// Test single value
	singleValue := []SamplePair{{json.Number("1"), json.Number("2.0")}}
	chunks = client.chunkValues(singleValue, 1000)
	if len(chunks) != 1 || len(chunks[0]) != 1 {
		t.Errorf("Expected 1 chunk with 1 value, got: %d chunks", len(chunks))
	}

	// Test zero chunk size (should use default)
	values := make([]SamplePair, 5)
	chunks = client.chunkValues(values, 0)
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk with default size, got: %d", len(chunks))
	}

	// Test chunk size larger than data
	chunks = client.chunkValues(values, 10)
	if len(chunks) != 1 || len(chunks[0]) != 5 {
		t.Errorf("Expected 1 chunk with 5 values, got: %d chunks with %d values",
			len(chunks), len(chunks[0]))
	}
}

func TestStreamingDataChunk(t *testing.T) {
	chunk := StreamingDataChunk{
		ChunkID:    1,
		Data:       json.RawMessage(`{"test": "data"}`),
		Error:      nil,
		IsComplete: false,
		Timestamp:  time.Now(),
	}

	if chunk.ChunkID != 1 {
		t.Errorf("Expected ChunkID 1, got: %d", chunk.ChunkID)
	}

	if chunk.IsComplete {
		t.Error("Expected IsComplete to be false")
	}

	if chunk.Error != nil {
		t.Errorf("Expected no error, got: %v", chunk.Error)
	}

	// Test JSON marshaling
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("Failed to marshal chunk: %v", err)
	}

	var unmarshaled StreamingDataChunk
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal chunk: %v", err)
	}

	if unmarshaled.ChunkID != chunk.ChunkID {
		t.Errorf("Expected unmarshaled ChunkID %d, got: %d", chunk.ChunkID, unmarshaled.ChunkID)
	}
}

func TestStreamifyMatrixData(t *testing.T) {
	client := &Client{}

	// Create test matrix data
	matrix := Matrix{
		{
			Metric: map[string]string{"instance": "server1", "job": "test"},
			Values: make([]SamplePair, 2500), // Large dataset
		},
		{
			Metric: map[string]string{"instance": "server2", "job": "test"},
			Values: make([]SamplePair, 1500), // Medium dataset
		},
	}

	// Fill with test data
	baseTime := time.Now().Unix()
	for i := range matrix[0].Values {
		matrix[0].Values[i] = SamplePair{
			json.Number(fmt.Sprintf("%d", baseTime+int64(i))),
			json.Number(fmt.Sprintf("%.2f", float64(i)*1.1)),
		}
	}

	for i := range matrix[1].Values {
		matrix[1].Values[i] = SamplePair{
			json.Number(fmt.Sprintf("%d", baseTime+int64(i))),
			json.Number(fmt.Sprintf("%.2f", float64(i)*0.9)),
		}
	}

	// Marshal matrix to raw JSON
	rawData, err := json.Marshal(matrix)
	if err != nil {
		t.Fatalf("Failed to marshal test matrix: %v", err)
	}

	// Create stream channel
	dataStream := make(chan StreamingDataChunk, 20)
	config := StreamingConfig{
		ChunkSize:    1000,
		WriteTimeout: 1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Process streaming data
	go client.streamifyMatrixData(ctx, json.RawMessage(rawData), dataStream, config)

	// Collect chunks with timeout
	var chunks []StreamingDataChunk
	var completedChunk *StreamingDataChunk

	timeout := time.After(3 * time.Second)

	for {
		select {
		case chunk, ok := <-dataStream:
			if !ok {
				// Stream closed normally
				goto checkResults
			}

			if chunk.Error != nil {
				t.Fatalf("Streaming error in chunk %d: %v", chunk.ChunkID, chunk.Error)
			}

			chunks = append(chunks, chunk)

			if chunk.IsComplete {
				completedChunk = &chunk
				// Don't break here, continue until channel closes
			}

		case <-timeout:
			// If we have received some chunks, that's good enough for testing
			if len(chunks) > 0 {
				goto checkResults
			}
			t.Fatal("Streaming timed out with no chunks received")

		case <-ctx.Done():
			if len(chunks) > 0 {
				goto checkResults
			}
			t.Fatal("Context cancelled with no chunks received")
		}
	}

checkResults:
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	if completedChunk == nil {
		t.Error("Expected at least one chunk to be marked as complete")
	}

	// Verify chunk IDs are sequential
	for i, chunk := range chunks {
		if chunk.ChunkID != i {
			t.Errorf("Expected chunk ID %d, got: %d", i, chunk.ChunkID)
		}
	}

	t.Logf("Received %d chunks from streaming", len(chunks))
}

func TestCollectStreamingResults(t *testing.T) {
	// Create mock streaming result
	dataStream := make(chan StreamingDataChunk, 5)
	streamResult := &QueryResultStream{
		Status: "streaming",
		Data: StreamingResultData{
			ResultType: ResultTypeMatrix,
		},
		Stream: dataStream,
	}

	// Create test chunks
	testSeries1 := []MatrixSeries{{
		Metric: map[string]string{"instance": "server1"},
		Values: []SamplePair{
			{json.Number("1234567890"), json.Number("1.0")},
			{json.Number("1234567891"), json.Number("2.0")},
		},
	}}

	testSeries2 := []MatrixSeries{{
		Metric: map[string]string{"instance": "server2"},
		Values: []SamplePair{
			{json.Number("1234567890"), json.Number("3.0")},
			{json.Number("1234567891"), json.Number("4.0")},
		},
	}}

	chunk1Data, _ := json.Marshal(testSeries1)
	chunk2Data, _ := json.Marshal(testSeries2)

	// Send chunks
	go func() {
		defer close(dataStream)

		dataStream <- StreamingDataChunk{
			ChunkID:    0,
			Data:       json.RawMessage(chunk1Data),
			IsComplete: false,
			Timestamp:  time.Now(),
		}

		dataStream <- StreamingDataChunk{
			ChunkID:    1,
			Data:       json.RawMessage(chunk2Data),
			IsComplete: true,
			Timestamp:  time.Now(),
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Collect results
	result, err := CollectStreamingResults(ctx, streamResult)
	if err != nil {
		t.Fatalf("Failed to collect streaming results: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got: %s", result.Status)
	}

	if result.Data.ResultType != ResultTypeMatrix {
		t.Errorf("Expected result type matrix, got: %s", result.Data.ResultType)
	}

	// Parse collected matrix
	var matrix Matrix
	if err := json.Unmarshal(result.Data.Result, &matrix); err != nil {
		t.Fatalf("Failed to parse collected matrix: %v", err)
	}

	if len(matrix) != 2 {
		t.Errorf("Expected 2 series, got: %d", len(matrix))
	}

	t.Logf("Collected matrix with %d series", len(matrix))
}

func TestCollectStreamingResultsWithError(t *testing.T) {
	dataStream := make(chan StreamingDataChunk, 2)
	streamResult := &QueryResultStream{
		Status: "streaming",
		Stream: dataStream,
	}

	// Send error chunk
	go func() {
		defer close(dataStream)

		dataStream <- StreamingDataChunk{
			ChunkID: 0,
			Error:   errors.New("test streaming error"),
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should return error
	_, err := CollectStreamingResults(ctx, streamResult)
	if err == nil {
		t.Error("Expected error from streaming results, got nil")
	}

	if !strings.Contains(err.Error(), "test streaming error") {
		t.Errorf("Expected error to contain 'test streaming error', got: %v", err)
	}
}

func TestCollectStreamingResultsTimeout(t *testing.T) {
	dataStream := make(chan StreamingDataChunk, 1)
	streamResult := &QueryResultStream{
		Status: "streaming",
		Stream: dataStream,
	}

	// Don't send any data, just let it timeout

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should timeout
	_, err := CollectStreamingResults(ctx, streamResult)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got: %v", err)
	}
}
