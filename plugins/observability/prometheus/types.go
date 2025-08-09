package prometheus

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// QueryResult represents a Prometheus query result
type QueryResult struct {
	Status    string     `json:"status"`
	Data      ResultData `json:"data"`
	Error     string     `json:"error,omitempty"`
	ErrorType string     `json:"errorType,omitempty"`
	Warnings  []string   `json:"warnings,omitempty"`
}

// ResultData represents the data portion of a query result
type ResultData struct {
	ResultType ResultType      `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

// ResultType represents the type of result returned by Prometheus
type ResultType string

const (
	ResultTypeMatrix ResultType = "matrix"
	ResultTypeVector ResultType = "vector"
	ResultTypeScalar ResultType = "scalar"
	ResultTypeString ResultType = "string"
)

// Vector represents a vector result
type Vector []VectorSample

// VectorSample represents a single sample in a vector
type VectorSample struct {
	Metric    map[string]string `json:"metric"`
	Value     SamplePair        `json:"value"`
	Timestamp time.Time         `json:"-"`
}

// Matrix represents a matrix result
type Matrix []MatrixSeries

// MatrixSeries represents a time series in a matrix result
type MatrixSeries struct {
	Metric map[string]string `json:"metric"`
	Values []SamplePair      `json:"values"`
}

// SamplePair represents a timestamp-value pair
type SamplePair [2]json.Number

// Timestamp returns the timestamp of the sample
func (s SamplePair) Timestamp() time.Time {
	ts, _ := strconv.ParseFloat(string(s[0]), 64)
	return time.Unix(int64(ts), 0)
}

// Value returns the value of the sample
func (s SamplePair) Value() float64 {
	v, _ := strconv.ParseFloat(string(s[1]), 64)
	return v
}

// UnmarshalJSON implements custom JSON unmarshaling for VectorSample
func (v *VectorSample) UnmarshalJSON(data []byte) error {
	var raw struct {
		Metric map[string]string `json:"metric"`
		Value  SamplePair        `json:"value"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	v.Metric = raw.Metric
	v.Value = raw.Value
	v.Timestamp = raw.Value.Timestamp()

	return nil
}

// GetVector extracts vector data from the result
func (r *QueryResult) GetVector() (Vector, error) {
	if r.Data.ResultType != ResultTypeVector {
		return nil, fmt.Errorf("result is not a vector, got %s", r.Data.ResultType)
	}

	var vector Vector
	if err := json.Unmarshal(r.Data.Result, &vector); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vector: %w", err)
	}

	return vector, nil
}

// GetMatrix extracts matrix data from the result
func (r *QueryResult) GetMatrix() (Matrix, error) {
	if r.Data.ResultType != ResultTypeMatrix {
		return nil, fmt.Errorf("result is not a matrix, got %s", r.Data.ResultType)
	}

	var matrix Matrix
	if err := json.Unmarshal(r.Data.Result, &matrix); err != nil {
		return nil, fmt.Errorf("failed to unmarshal matrix: %w", err)
	}

	return matrix, nil
}

// GetScalar extracts scalar data from the result
func (r *QueryResult) GetScalar() (float64, time.Time, error) {
	if r.Data.ResultType != ResultTypeScalar {
		return 0, time.Time{}, fmt.Errorf("result is not a scalar, got %s", r.Data.ResultType)
	}

	var sample SamplePair
	if err := json.Unmarshal(r.Data.Result, &sample); err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to unmarshal scalar: %w", err)
	}

	return sample.Value(), sample.Timestamp(), nil
}

// GetString extracts string data from the result
func (r *QueryResult) GetString() (string, time.Time, error) {
	if r.Data.ResultType != ResultTypeString {
		return "", time.Time{}, fmt.Errorf("result is not a string, got %s", r.Data.ResultType)
	}

	var sample []json.RawMessage
	if err := json.Unmarshal(r.Data.Result, &sample); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to unmarshal string: %w", err)
	}

	if len(sample) != 2 {
		return "", time.Time{}, fmt.Errorf("invalid string result format")
	}

	var timestamp float64
	if err := json.Unmarshal(sample[0], &timestamp); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}

	var value string
	if err := json.Unmarshal(sample[1], &value); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, time.Unix(int64(timestamp), 0), nil
}

// Alert represents a Prometheus alert
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
	Value       float64           `json:"value"`
}

// AlertState represents the state of an alert
type AlertState string

const (
	AlertStatePending  AlertState = "pending"
	AlertStateFiring   AlertState = "firing"
	AlertStateInactive AlertState = "inactive"
)

// Target represents a Prometheus target
type Target struct {
	Labels          map[string]string `json:"labels"`
	ScrapeURL       string            `json:"scrapeUrl"`
	LastError       string            `json:"lastError"`
	LastScrape      time.Time         `json:"lastScrape"`
	Health          string            `json:"health"`
	GlobalURL       string            `json:"globalUrl"`
	LastScrapeDuration float64        `json:"lastScrapeDuration"`
}

// Metadata represents metric metadata
type Metadata struct {
	Type string   `json:"type"`
	Help string   `json:"help"`
	Unit string   `json:"unit"`
}