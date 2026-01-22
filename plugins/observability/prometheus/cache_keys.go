package prometheus

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// CacheKeyGenerator provides optimized cache key generation for Prometheus queries
type CacheKeyGenerator struct {
	// Compiled regex patterns for query normalization
	whitespaceRegex *regexp.Regexp
	numberRegex     *regexp.Regexp
	stringRegex     *regexp.Regexp
}

// NewCacheKeyGenerator creates a new cache key generator with compiled patterns
func NewCacheKeyGenerator() *CacheKeyGenerator {
	return &CacheKeyGenerator{
		whitespaceRegex: regexp.MustCompile(`\s+`),
		numberRegex:     regexp.MustCompile(`\b\d+\.?\d*\b`),
		stringRegex:     regexp.MustCompile(`"[^"]*"`),
	}
}

// CacheKeyContext holds context information for cache key generation
type CacheKeyContext struct {
	Query     string
	StartTime *time.Time
	EndTime   *time.Time
	Step      *time.Duration
	QueryType QueryType
}

// QueryType represents the type of Prometheus query
type QueryType int

const (
	QueryTypeInstant QueryType = iota
	QueryTypeRange
	QueryTypeSeries
	QueryTypeLabels
)

// GenerateKey creates an optimized cache key for a Prometheus query
func (g *CacheKeyGenerator) GenerateKey(ctx CacheKeyContext) string {
	// Step 1: Normalize the query string
	normalized := g.normalizeQuery(ctx.Query)

	// Step 2: Create a shorter representation using hash for long queries
	var queryPart string
	if len(normalized) > 100 {
		// Use hash for very long queries
		hash := sha256.Sum256([]byte(normalized))
		queryPart = fmt.Sprintf("hash_%x", hash[:8]) // Use first 8 bytes of hash
	} else {
		queryPart = normalized
	}

	// Step 3: Add context information
	var keyParts []string
	keyParts = append(keyParts, string(rune('0'+int(ctx.QueryType)))) // Query type prefix
	keyParts = append(keyParts, queryPart)

	// Add time information based on query type
	if ctx.QueryType == QueryTypeInstant && ctx.StartTime != nil {
		// For instant queries, use the timestamp as both start and end
		timeRange := fmt.Sprintf("%d-%d", ctx.StartTime.Unix(), ctx.StartTime.Unix())
		keyParts = append(keyParts, timeRange)
	} else if ctx.StartTime != nil && ctx.EndTime != nil {
		// For range queries, use the actual time range
		timeRange := fmt.Sprintf("%d-%d", ctx.StartTime.Unix(), ctx.EndTime.Unix())
		keyParts = append(keyParts, timeRange)
	}

	// Add step information for range queries
	if ctx.Step != nil {
		stepStr := fmt.Sprintf("s%d", int(ctx.Step.Seconds()))
		keyParts = append(keyParts, stepStr)
	}

	return strings.Join(keyParts, "|")
}

// normalizeQuery normalizes a Prometheus query for consistent cache keys
func (g *CacheKeyGenerator) normalizeQuery(query string) string {
	// Step 1: Trim whitespace and convert to lowercase for function names
	normalized := strings.TrimSpace(query)

	// Step 2: Normalize whitespace within duration brackets first [5m] -> [5m]
	normalized = g.normalizeDurationBrackets(normalized)

	// Step 3: Normalize whitespace around operators and punctuation
	normalized = g.normalizeOperatorWhitespace(normalized)

	// Step 4: Normalize whitespace (multiple spaces/tabs/newlines to single space)
	normalized = g.whitespaceRegex.ReplaceAllString(normalized, " ")

	// Step 5: Sort label selectors to handle different orderings
	normalized = g.normalizeLabelSelectors(normalized)

	// Step 6: Normalize numeric literals (optional - can help with similar queries)
	// This is conservative to avoid breaking query semantics

	return normalized
}

// normalizeDurationBrackets normalizes whitespace within duration brackets
func (g *CacheKeyGenerator) normalizeDurationBrackets(query string) string {
	// Pattern to match duration brackets like [5m], [ 1h ], etc.
	durationRegex := regexp.MustCompile(`\[\s*([^]]+?)\s*\]`)

	return durationRegex.ReplaceAllStringFunc(query, func(match string) string {
		// Extract the duration part between brackets
		submatch := durationRegex.FindStringSubmatch(match)
		if len(submatch) > 1 {
			// Remove extra whitespace from duration and rebuild bracket
			duration := strings.TrimSpace(submatch[1])
			return fmt.Sprintf("[%s]", duration)
		}
		return match
	})
}

// normalizeOperatorWhitespace normalizes whitespace around operators and punctuation
func (g *CacheKeyGenerator) normalizeOperatorWhitespace(query string) string {
	// Remove extra spaces around parentheses, braces, and brackets
	// But preserve necessary spaces between keywords and identifiers

	// Handle parentheses - remove spaces inside
	parenRegex := regexp.MustCompile(`\(\s+|\s+\)`)
	normalized := parenRegex.ReplaceAllStringFunc(query, func(match string) string {
		if strings.HasPrefix(match, "(") {
			return "("
		}
		return ")"
	})

	// Handle braces - remove spaces inside AND before metric selectors
	braceRegex := regexp.MustCompile(`\{\s+|\s+\}`)
	normalized = braceRegex.ReplaceAllStringFunc(normalized, func(match string) string {
		if strings.HasPrefix(match, "{") {
			return "{"
		}
		return "}"
	})

	// Remove spaces before braces for metric selectors like "metric {labels}" -> "metric{labels}"
	metricBraceRegex := regexp.MustCompile(`([a-zA-Z_:][a-zA-Z0-9_:]*)\s+\{`)
	normalized = metricBraceRegex.ReplaceAllString(normalized, "$1{")

	// Handle commas - remove extra spaces
	commaRegex := regexp.MustCompile(`\s*,\s*`)
	normalized = commaRegex.ReplaceAllString(normalized, ",")

	// Handle equals - remove extra spaces around equals in label selectors
	equalsRegex := regexp.MustCompile(`\s*=\s*`)
	normalized = equalsRegex.ReplaceAllString(normalized, "=")

	return normalized
}

// normalizeLabelSelectors sorts label selectors within metric selectors
func (g *CacheKeyGenerator) normalizeLabelSelectors(query string) string {
	// Pattern to match metric selectors: metric_name{label1="value1",label2="value2"}
	selectorRegex := regexp.MustCompile(`([a-zA-Z_:][a-zA-Z0-9_:]*)\{([^}]+)\}`)

	return selectorRegex.ReplaceAllStringFunc(query, func(match string) string {
		parts := selectorRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match // Return unchanged if pattern doesn't match expected format
		}

		metricName := parts[1]
		labelsPart := parts[2]

		// Split labels and sort them
		labels := strings.Split(labelsPart, ",")
		for i, label := range labels {
			labels[i] = strings.TrimSpace(label)
		}

		// Sort labels for consistent ordering
		sort.Strings(labels)

		return fmt.Sprintf("%s{%s}", metricName, strings.Join(labels, ","))
	})
}

// GenerateInstantQueryKey generates a cache key for instant queries
func (g *CacheKeyGenerator) GenerateInstantQueryKey(query string, queryTime *time.Time) string {
	ctx := CacheKeyContext{
		Query:     query,
		QueryType: QueryTypeInstant,
	}

	// Add timestamp for instant queries if provided
	if queryTime != nil && !queryTime.IsZero() {
		// Round to nearest minute to improve cache hit rate for similar timestamps
		rounded := queryTime.Truncate(time.Minute)
		ctx.StartTime = &rounded
	}

	return g.GenerateKey(ctx)
}

// GenerateRangeQueryKey generates a cache key for range queries
func (g *CacheKeyGenerator) GenerateRangeQueryKey(query string, start, end time.Time, step time.Duration) string {
	ctx := CacheKeyContext{
		Query:     query,
		StartTime: &start,
		EndTime:   &end,
		Step:      &step,
		QueryType: QueryTypeRange,
	}

	return g.GenerateKey(ctx)
}

// GenerateSeriesQueryKey generates a cache key for series queries
func (g *CacheKeyGenerator) GenerateSeriesQueryKey(matches []string, start, end time.Time) string {
	// Sort matches for consistent ordering
	sortedMatches := make([]string, len(matches))
	copy(sortedMatches, matches)
	sort.Strings(sortedMatches)

	query := strings.Join(sortedMatches, "|")

	ctx := CacheKeyContext{
		Query:     query,
		StartTime: &start,
		EndTime:   &end,
		QueryType: QueryTypeSeries,
	}

	return g.GenerateKey(ctx)
}

// GenerateLabelsQueryKey generates a cache key for labels queries
func (g *CacheKeyGenerator) GenerateLabelsQueryKey() string {
	ctx := CacheKeyContext{
		Query:     "labels",
		QueryType: QueryTypeLabels,
	}

	return g.GenerateKey(ctx)
}

// BatchGenerateKeys generates cache keys for multiple queries efficiently
func (g *CacheKeyGenerator) BatchGenerateKeys(queries map[string]string, queryTime *time.Time) map[string]string {
	keys := make(map[string]string, len(queries))

	for name, query := range queries {
		keys[name] = g.GenerateInstantQueryKey(query, queryTime)
	}

	return keys
}

// EstimateCacheEfficiency analyzes query patterns to estimate cache effectiveness
type CacheEfficiencyStats struct {
	TotalQueries     int
	UniqueNormalized int
	PotentialHitRate float64
	AverageKeyLength float64
	LongQueryCount   int // Queries that will be hashed
}

// AnalyzeQueryPatterns analyzes a set of queries to estimate cache efficiency
func (g *CacheKeyGenerator) AnalyzeQueryPatterns(queries []string) CacheEfficiencyStats {
	if len(queries) == 0 {
		return CacheEfficiencyStats{}
	}

	normalizedSet := make(map[string]int)
	totalKeyLength := 0
	longQueries := 0

	for _, query := range queries {
		normalized := g.normalizeQuery(query)
		normalizedSet[normalized]++

		keyLength := len(normalized)
		if keyLength > 100 {
			longQueries++
			keyLength = 24 // Estimated hash key length
		}
		totalKeyLength += keyLength
	}

	uniqueNormalized := len(normalizedSet)
	potentialSavings := len(queries) - uniqueNormalized
	potentialHitRate := 0.0
	if len(queries) > 0 {
		potentialHitRate = float64(potentialSavings) / float64(len(queries)) * 100
	}

	avgKeyLength := 0.0
	if len(queries) > 0 {
		avgKeyLength = float64(totalKeyLength) / float64(len(queries))
	}

	return CacheEfficiencyStats{
		TotalQueries:     len(queries),
		UniqueNormalized: uniqueNormalized,
		PotentialHitRate: potentialHitRate,
		AverageKeyLength: avgKeyLength,
		LongQueryCount:   longQueries,
	}
}

// String returns a string representation of cache efficiency stats
func (s CacheEfficiencyStats) String() string {
	return fmt.Sprintf(
		"Queries: %d, Unique: %d, Potential Hit Rate: %.1f%%, Avg Key Length: %.1f, Long Queries: %d",
		s.TotalQueries, s.UniqueNormalized, s.PotentialHitRate, s.AverageKeyLength, s.LongQueryCount,
	)
}
