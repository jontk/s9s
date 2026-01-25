// Package prometheus provides streaming capabilities for large Prometheus query results.
//
// IMPORTANT: Prometheus API does not natively support streaming responses.
// This implementation provides CLIENT-SIDE streaming by:
// 1. Fetching complete responses from Prometheus
// 2. Post-processing large datasets into manageable chunks
// 3. Streaming chunks to consumer applications for memory efficiency
//
// This is particularly useful for:
// - Large range queries with many data points
// - Memory-constrained environments
// - Progressive data processing without blocking on large datasets
package prometheus

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// StreamingConfig holds configuration for streaming queries
type StreamingConfig struct {
	ChunkSize    int           // Number of data points per chunk (default: 1000)
	BufferSize   int           // Buffer size for streaming (default: 64KB)
	ReadTimeout  time.Duration // Timeout for reading each chunk (default: 10s)
	WriteTimeout time.Duration // Timeout for writing each chunk (default: 5s)
}

// DefaultStreamingConfig returns default streaming configuration
func DefaultStreamingConfig() StreamingConfig {
	return StreamingConfig{
		ChunkSize:    1000,
		BufferSize:   64 * 1024, // 64KB
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

// QueryResultStream represents a streaming query result
type QueryResultStream struct {
	Status    string                    `json:"status"`
	Data      StreamingResultData       `json:"data"`
	Error     string                    `json:"error,omitempty"`
	ErrorType string                    `json:"errorType,omitempty"`
	Warnings  []string                  `json:"warnings,omitempty"`
	Stream    <-chan StreamingDataChunk `json:"-"`
}

// StreamingResultData represents streaming result metadata
type StreamingResultData struct {
	ResultType  ResultType `json:"resultType"`
	ChunkCount  int        `json:"chunkCount"`
	TotalPoints int        `json:"totalPoints"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     time.Time  `json:"endTime"`
}

// StreamingDataChunk represents a chunk of streaming data
type StreamingDataChunk struct {
	ChunkID    int             `json:"chunkId"`
	Data       json.RawMessage `json:"data"`
	Error      error           `json:"error,omitempty"`
	IsComplete bool            `json:"isComplete"`
	Timestamp  time.Time       `json:"timestamp"`
}

// QueryRangeStream executes a Prometheus range query and returns results as a client-side stream
// Note: Prometheus API does not natively support streaming. This method fetches the complete
// response and then streams it in chunks for better memory management and progressive processing.
func (c *Client) QueryRangeStream(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResultStream, error) {
	return c.QueryRangeStreamWithConfig(ctx, query, start, end, step, DefaultStreamingConfig())
}

// QueryRangeStreamWithConfig executes a range query with client-side streaming and custom configuration
// This provides memory-efficient processing of large datasets by chunking the response.
func (c *Client) QueryRangeStreamWithConfig(ctx context.Context, query string, start, end time.Time, step time.Duration, config StreamingConfig) (*QueryResultStream, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))

	// Make HTTP request for streaming
	resp, err := c.doStreamingRequest(ctx, "GET", "/api/v1/query_range?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("streaming range query failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("streaming range query failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Create streaming result
	streamResult := &QueryResultStream{
		Status: "streaming",
		Data: StreamingResultData{
			ResultType: ResultTypeMatrix,
			StartTime:  start,
			EndTime:    end,
		},
	}

	// Create data stream channel
	dataStream := make(chan StreamingDataChunk, 10) // Buffered channel
	streamResult.Stream = dataStream

	// Start streaming goroutine
	go c.processStreamingResponse(ctx, resp.Body, dataStream, config)

	return streamResult, nil
}

// doStreamingRequest performs an HTTP request optimized for streaming
func (c *Client) doStreamingRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	fullURL := c.endpoint + path

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming request: %w", err)
	}

	// Add authentication
	if c.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// Set headers optimized for streaming
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "identity") // Disable compression for streaming
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	return resp, nil
}

// processStreamingResponse processes the HTTP response body as a stream
func (c *Client) processStreamingResponse(ctx context.Context, body io.ReadCloser, dataStream chan<- StreamingDataChunk, config StreamingConfig) {
	defer close(dataStream)
	defer func() { _ = body.Close() }()

	// Create a buffered reader for efficient reading
	reader := bufio.NewReaderSize(body, config.BufferSize)

	// First, read the initial response to get metadata
	var initialResponse struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string          `json:"resultType"`
			Result     json.RawMessage `json:"result"`
		} `json:"data"`
		Error     string   `json:"error,omitempty"`
		ErrorType string   `json:"errorType,omitempty"`
		Warnings  []string `json:"warnings,omitempty"`
	}

	// Read the complete response first (for compatibility with standard Prometheus API)
	allData, err := io.ReadAll(reader)
	if err != nil {
		dataStream <- StreamingDataChunk{
			ChunkID:   -1,
			Error:     fmt.Errorf("failed to read response: %w", err),
			Timestamp: time.Now(),
		}
		return
	}

	if err := json.Unmarshal(allData, &initialResponse); err != nil {
		dataStream <- StreamingDataChunk{
			ChunkID:   -1,
			Error:     fmt.Errorf("failed to parse response: %w", err),
			Timestamp: time.Now(),
		}
		return
	}

	if initialResponse.Status != "success" {
		dataStream <- StreamingDataChunk{
			ChunkID:   -1,
			Error:     fmt.Errorf("query failed: %s", initialResponse.Error),
			Timestamp: time.Now(),
		}
		return
	}

	// Process the result data in chunks
	c.streamifyMatrixData(ctx, initialResponse.Data.Result, dataStream, config)
}

// streamifyMatrixData converts matrix data into streaming chunks
func (c *Client) streamifyMatrixData(ctx context.Context, rawData json.RawMessage, dataStream chan<- StreamingDataChunk, config StreamingConfig) {
	var matrix Matrix
	if err := json.Unmarshal(rawData, &matrix); err != nil {
		c.sendStreamChunkError(dataStream, -1, fmt.Errorf("failed to parse matrix data: %w", err))
		return
	}

	chunkID := 0
	for seriesIdx, series := range matrix {
		if !c.checkStreamContext(ctx, dataStream, chunkID) {
			return
		}

		valueChunks := c.chunkValues(series.Values, config.ChunkSize)
		for chunkIdx, valueChunk := range valueChunks {
			if !c.processAndStreamChunk(ctx, dataStream, &series, valueChunk, seriesIdx, chunkIdx, len(matrix), len(valueChunks), &chunkID, config) {
				return
			}
		}
	}
}

func (c *Client) checkStreamContext(ctx context.Context, dataStream chan<- StreamingDataChunk, chunkID int) bool {
	select {
	case <-ctx.Done():
		c.sendStreamChunkError(dataStream, chunkID, ctx.Err())
		return false
	default:
		return true
	}
}

func (c *Client) processAndStreamChunk(ctx context.Context, dataStream chan<- StreamingDataChunk, series *MatrixSeries, valueChunk []SamplePair, seriesIdx, chunkIdx, matrixLen, numChunks int, chunkID *int, config StreamingConfig) bool {
	chunkSeries := MatrixSeries{
		Metric: series.Metric,
		Values: valueChunk,
	}

	chunkData, err := json.Marshal([]MatrixSeries{chunkSeries})
	if err != nil {
		c.sendStreamChunkError(dataStream, *chunkID, fmt.Errorf("failed to marshal chunk: %w", err))
		return false
	}

	isLastChunk := (seriesIdx == matrixLen-1) && (chunkIdx == numChunks-1)
	chunk := StreamingDataChunk{
		ChunkID:    *chunkID,
		Data:       json.RawMessage(chunkData),
		IsComplete: isLastChunk,
		Timestamp:  time.Now(),
	}

	return c.sendStreamChunkWithTimeout(ctx, dataStream, &chunk, config.WriteTimeout, chunkID)
}

func (c *Client) sendStreamChunkError(dataStream chan<- StreamingDataChunk, chunkID int, err error) {
	dataStream <- StreamingDataChunk{
		ChunkID:   chunkID,
		Error:     err,
		Timestamp: time.Now(),
	}
}

func (c *Client) sendStreamChunkWithTimeout(ctx context.Context, dataStream chan<- StreamingDataChunk, chunk *StreamingDataChunk, timeout time.Duration, chunkID *int) bool {
	select {
	case dataStream <- *chunk:
		*chunkID++
		return true
	case <-ctx.Done():
		c.sendStreamChunkError(dataStream, chunk.ChunkID, ctx.Err())
		return false
	case <-time.After(timeout):
		c.sendStreamChunkError(dataStream, chunk.ChunkID, fmt.Errorf("write timeout for chunk %d", chunk.ChunkID))
		return false
	}
}

// chunkValues splits a slice of SamplePairs into chunks of specified size
func (c *Client) chunkValues(values []SamplePair, chunkSize int) [][]SamplePair {
	if chunkSize <= 0 {
		chunkSize = 1000 // Default chunk size
	}

	var chunks [][]SamplePair
	for i := 0; i < len(values); i += chunkSize {
		end := i + chunkSize
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[i:end])
	}

	return chunks
}

// CollectStreamingResults collects all chunks from a streaming result into a standard QueryResult
func CollectStreamingResults(ctx context.Context, streamResult *QueryResultStream) (*QueryResult, error) {
	var allSeries []MatrixSeries

	for {
		select {
		case chunk, ok := <-streamResult.Stream:
			if !ok {
				// Stream closed
				result := &QueryResult{
					Status: "success",
					Data: ResultData{
						ResultType: ResultTypeMatrix,
					},
				}

				// Marshal the collected series
				matrixData, err := json.Marshal(allSeries)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal collected data: %w", err)
				}
				result.Data.Result = json.RawMessage(matrixData)

				return result, nil
			}

			if chunk.Error != nil {
				return nil, fmt.Errorf("streaming error in chunk %d: %w", chunk.ChunkID, chunk.Error)
			}

			// Parse and append chunk data
			var chunkSeries []MatrixSeries
			if err := json.Unmarshal(chunk.Data, &chunkSeries); err != nil {
				return nil, fmt.Errorf("failed to parse chunk %d: %w", chunk.ChunkID, err)
			}

			allSeries = append(allSeries, chunkSeries...)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
