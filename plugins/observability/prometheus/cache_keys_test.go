package prometheus

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCacheKeyGenerator(t *testing.T) {
	gen := NewCacheKeyGenerator()

	if gen.whitespaceRegex == nil {
		t.Error("Expected whitespace regex to be initialized")
	}

	if gen.numberRegex == nil {
		t.Error("Expected number regex to be initialized")
	}

	if gen.stringRegex == nil {
		t.Error("Expected string regex to be initialized")
	}
}

func TestNormalizeQuery(t *testing.T) {
	gen := NewCacheKeyGenerator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic whitespace normalization",
			input:    "up    {job=\"prometheus\"}",
			expected: "up{job=\"prometheus\"}",
		},
		{
			name:     "multiline query normalization",
			input:    "rate(\n  http_requests_total[\n    5m\n  ]\n)",
			expected: "rate(http_requests_total[5m])",
		},
		{
			name:     "label selector normalization",
			input:    "cpu_usage{instance=\"server2\",job=\"node\",env=\"prod\"}",
			expected: "cpu_usage{env=\"prod\",instance=\"server2\",job=\"node\"}",
		},
		{
			name:     "complex query with multiple selectors",
			input:    "rate(http_requests{method=\"GET\",code=\"200\"}[5m]) + rate(http_requests{code=\"200\",method=\"POST\"}[5m])",
			expected: "rate(http_requests{code=\"200\",method=\"GET\"}[5m]) + rate(http_requests{code=\"200\",method=\"POST\"}[5m])",
		},
		{
			name:     "query with extra whitespace",
			input:    "  up  { job = \"prometheus\" }  ",
			expected: "up{job=\"prometheus\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.normalizeQuery(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateInstantQueryKey(t *testing.T) {
	gen := NewCacheKeyGenerator()

	now := time.Now()
	roundedTime := now.Truncate(time.Minute)

	tests := []struct {
		name      string
		query     string
		queryTime *time.Time
		expectKey string
	}{
		{
			name:      "simple query without time",
			query:     "up",
			queryTime: nil,
			expectKey: "0|up",
		},
		{
			name:      "simple query with time",
			query:     "up",
			queryTime: &now,
			expectKey: "0|up|" + strconv.FormatInt(roundedTime.Unix(), 10) + "-" + strconv.FormatInt(roundedTime.Unix(), 10),
		},
		{
			name:      "query with label selectors",
			query:     "cpu_usage{job=\"node\",instance=\"server1\"}",
			queryTime: nil,
			expectKey: "0|cpu_usage{instance=\"server1\",job=\"node\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateInstantQueryKey(tt.query, tt.queryTime)
			if result != tt.expectKey {
				t.Errorf("Expected %q, got %q", tt.expectKey, result)
			}
		})
	}
}

func TestGenerateRangeQueryKey(t *testing.T) {
	gen := NewCacheKeyGenerator()

	start := time.Unix(1609459200, 0) // 2021-01-01 00:00:00
	end := time.Unix(1609545600, 0)   // 2021-01-02 00:00:00
	step := 5 * time.Minute

	key := gen.GenerateRangeQueryKey("rate(cpu_usage[5m])", start, end, step)

	expected := "1|rate(cpu_usage[5m])|1609459200-1609545600|s300"
	if key != expected {
		t.Errorf("Expected %q, got %q", expected, key)
	}
}

func TestGenerateSeriesQueryKey(t *testing.T) {
	gen := NewCacheKeyGenerator()

	start := time.Unix(1609459200, 0)
	end := time.Unix(1609545600, 0)
	matches := []string{"cpu_usage", "memory_usage", "disk_usage"}

	key := gen.GenerateSeriesQueryKey(matches, start, end)

	// Should sort the matches
	expected := "2|cpu_usage|disk_usage|memory_usage|1609459200-1609545600"
	if key != expected {
		t.Errorf("Expected %q, got %q", expected, key)
	}

	// Test that different order produces same key
	matchesReordered := []string{"memory_usage", "disk_usage", "cpu_usage"}
	key2 := gen.GenerateSeriesQueryKey(matchesReordered, start, end)

	if key != key2 {
		t.Errorf("Expected same key for reordered matches, got %q and %q", key, key2)
	}
}

func TestGenerateLabelsQueryKey(t *testing.T) {
	gen := NewCacheKeyGenerator()

	key := gen.GenerateLabelsQueryKey()
	expected := "3|labels"

	if key != expected {
		t.Errorf("Expected %q, got %q", expected, key)
	}
}

func TestLongQueryHashing(t *testing.T) {
	gen := NewCacheKeyGenerator()

	// Create a very long query (over 100 characters)
	longQuery := "rate(very_long_metric_name_that_exceeds_normal_length_limits{" +
		"very_long_label_name_1=\"very_long_label_value_1\"," +
		"very_long_label_name_2=\"very_long_label_value_2\"," +
		"very_long_label_name_3=\"very_long_label_value_3\"}" +
		"[5m])"

	key := gen.GenerateInstantQueryKey(longQuery, nil)

	// Should start with query type and have hash prefix
	if !strings.HasPrefix(key, "0|hash_") {
		t.Errorf("Expected key to start with '0|hash_', got: %s", key)
	}

	// Key should be much shorter than original query
	if len(key) >= len(longQuery) {
		t.Errorf("Expected key (%d chars) to be shorter than original query (%d chars)",
			len(key), len(longQuery))
	}
}

func TestBatchGenerateKeys(t *testing.T) {
	gen := NewCacheKeyGenerator()

	queries := map[string]string{
		"cpu":    "cpu_usage",
		"memory": "memory_usage",
		"disk":   "disk_usage",
	}

	now := time.Now()
	keys := gen.BatchGenerateKeys(queries, &now)

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify all keys are generated
	for name := range queries {
		if _, exists := keys[name]; !exists {
			t.Errorf("Expected key for %s to be generated", name)
		}
	}

	// Verify keys are properly formatted
	for name, key := range keys {
		if !strings.HasPrefix(key, "0|") {
			t.Errorf("Expected key for %s to start with '0|', got: %s", name, key)
		}
	}
}

func TestAnalyzeQueryPatterns(t *testing.T) {
	gen := NewCacheKeyGenerator()

	queries := []string{
		"up",
		"up{job=\"prometheus\"}",
		"up { job=\"prometheus\" }", // Same as above with different whitespace
		"cpu_usage",
		"rate(http_requests[5m])",
	}

	stats := gen.AnalyzeQueryPatterns(queries)

	if stats.TotalQueries != 5 {
		t.Errorf("Expected 5 total queries, got %d", stats.TotalQueries)
	}

	// Should have 4 unique normalized queries (up{job="prometheus"} variations should be same)
	if stats.UniqueNormalized != 4 {
		t.Errorf("Expected 4 unique normalized queries, got %d", stats.UniqueNormalized)
	}

	// Should have some potential hit rate (20% since 1 out of 5 is duplicate)
	expectedHitRate := 20.0
	if stats.PotentialHitRate != expectedHitRate {
		t.Errorf("Expected hit rate %.1f%%, got %.2f%%", expectedHitRate, stats.PotentialHitRate)
	}

	// Should have reasonable average key length
	if stats.AverageKeyLength <= 0 {
		t.Errorf("Expected positive average key length, got %.2f", stats.AverageKeyLength)
	}

	t.Logf("Analysis results: %s", stats.String())
}

func TestAnalyzeQueryPatternsEmpty(t *testing.T) {
	gen := NewCacheKeyGenerator()

	stats := gen.AnalyzeQueryPatterns([]string{})

	if stats.TotalQueries != 0 {
		t.Errorf("Expected 0 total queries, got %d", stats.TotalQueries)
	}

	if stats.UniqueNormalized != 0 {
		t.Errorf("Expected 0 unique queries, got %d", stats.UniqueNormalized)
	}

	if stats.PotentialHitRate != 0 {
		t.Errorf("Expected 0 hit rate, got %.2f", stats.PotentialHitRate)
	}
}

func TestCacheKeyConsistency(t *testing.T) {
	gen := NewCacheKeyGenerator()

	// Same query with different formatting should produce same key
	query1 := "rate(http_requests{method=\"GET\",status=\"200\"}[5m])"
	query2 := "rate(http_requests{status=\"200\", method=\"GET\"}[5m])"
	query3 := "rate( http_requests{ method=\"GET\" , status=\"200\" }[ 5m ] )"

	key1 := gen.GenerateInstantQueryKey(query1, nil)
	key2 := gen.GenerateInstantQueryKey(query2, nil)
	key3 := gen.GenerateInstantQueryKey(query3, nil)

	if key1 != key2 {
		t.Errorf("Expected same key for query1 and query2, got %q and %q", key1, key2)
	}

	if key1 != key3 {
		t.Errorf("Expected same key for query1 and query3, got %q and %q", key1, key3)
	}
}

func TestCacheKeyPerformance(t *testing.T) {
	// Skip strict performance checks in short mode to avoid flakiness in CI
	if testing.Short() {
		t.Skip("Skipping performance timing test in short mode")
	}

	gen := NewCacheKeyGenerator()

	// Test with various query sizes
	queries := []string{
		"up",
		"rate(http_requests[5m])",
		"rate(http_requests{method=\"GET\",status=\"200\",instance=\"server1\"}[5m])",
	}

	// Warm up
	for _, query := range queries {
		gen.GenerateInstantQueryKey(query, nil)
	}

	// Benchmark key generation
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, query := range queries {
			gen.GenerateInstantQueryKey(query, nil)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations*len(queries))

	// Log performance for informational purposes
	// Note: For reliable performance tracking, use Go benchmarks (go test -bench)
	// rather than timing assertions which are sensitive to system load
	t.Logf("Key generation performance: %v per key", avgTime)

	// Sanity check: warn if performance is extremely degraded (10x threshold)
	// This catches serious regressions without being flaky on slower systems
	if avgTime > 5000*time.Microsecond {
		t.Errorf("Key generation severely degraded: %v per key (sanity threshold: 5000µs)", avgTime)
	}
}
