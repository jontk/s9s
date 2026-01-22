package filters

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MemoryUnit represents memory size units
type MemoryUnit int64

const (
	Byte MemoryUnit = 1
	KB   MemoryUnit = 1024
	MB   MemoryUnit = 1024 * 1024
	GB   MemoryUnit = 1024 * 1024 * 1024
	TB   MemoryUnit = 1024 * 1024 * 1024 * 1024
)

// DateRangeFilter represents date range filtering
type DateRangeFilter struct {
	Field string
	Start *time.Time
	End   *time.Time
}

// AdvancedFilterParser extends FilterParser with enhanced parsing capabilities
type AdvancedFilterParser struct {
	*FilterParser
	dateFormats []string
}

// NewAdvancedFilterParser creates an enhanced filter parser
func NewAdvancedFilterParser() *AdvancedFilterParser {
	return &AdvancedFilterParser{
		FilterParser: NewFilterParser(),
		dateFormats: []string{
			"2006-01-02",               // YYYY-MM-DD
			"2006-01-02 15:04:05",      // YYYY-MM-DD HH:MM:SS
			"2006-01-02T15:04:05Z",     // ISO 8601
			"01/02/2006",               // MM/DD/YYYY
			"15:04:05",                 // HH:MM:SS (today)
			"Jan 2, 2006",              // Month Day, Year
			"January 2, 2006 15:04:05", // Full format
		},
	}
}

// ParseMemorySize parses memory size strings like "4G", "1024M", "512MB"
func ParseMemorySize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

	// Handle common patterns
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]?B?)$`)
	matches := re.FindStringSubmatch(sizeStr)

	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid memory size format: %s", sizeStr)
	}

	// Parse the numeric part
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", matches[1])
	}

	// Parse the unit
	unit := matches[2]
	var multiplier MemoryUnit

	switch unit {
	case "", "B":
		multiplier = Byte
	case "K", "KB":
		multiplier = KB
	case "M", "MB":
		multiplier = MB
	case "G", "GB":
		multiplier = GB
	case "T", "TB":
		multiplier = TB
	default:
		return 0, fmt.Errorf("unknown memory unit: %s", unit)
	}

	return int64(value * float64(multiplier)), nil
}

// ParseDuration parses duration strings like "2:30:00", "1-12:00:00", "90min"
func ParseSlurmDuration(duration string) (time.Duration, error) {
	duration = strings.TrimSpace(duration)

	// Handle SLURM time formats: [DD-]HH:MM:SS
	if strings.Contains(duration, ":") {
		// Check for DD-HH:MM:SS format
		if strings.Contains(duration, "-") {
			parts := strings.SplitN(duration, "-", 2)
			if len(parts) == 2 {
				days, err := strconv.Atoi(parts[0])
				if err != nil {
					return 0, fmt.Errorf("invalid day format: %s", parts[0])
				}

				timePart := parts[1]
				timeComponents := strings.Split(timePart, ":")
				if len(timeComponents) != 3 {
					return 0, fmt.Errorf("invalid time format: %s", timePart)
				}

				hours, err := strconv.Atoi(timeComponents[0])
				if err != nil {
					return 0, fmt.Errorf("invalid hour format: %s", timeComponents[0])
				}

				minutes, err := strconv.Atoi(timeComponents[1])
				if err != nil {
					return 0, fmt.Errorf("invalid minute format: %s", timeComponents[1])
				}

				seconds, err := strconv.Atoi(timeComponents[2])
				if err != nil {
					return 0, fmt.Errorf("invalid second format: %s", timeComponents[2])
				}

				totalSeconds := days*24*3600 + hours*3600 + minutes*60 + seconds
				return time.Duration(totalSeconds) * time.Second, nil
			}
		}

		// Handle HH:MM:SS format
		timeComponents := strings.Split(duration, ":")
		if len(timeComponents) == 3 {
			hours, err := strconv.Atoi(timeComponents[0])
			if err != nil {
				return 0, fmt.Errorf("invalid hour format: %s", timeComponents[0])
			}

			minutes, err := strconv.Atoi(timeComponents[1])
			if err != nil {
				return 0, fmt.Errorf("invalid minute format: %s", timeComponents[1])
			}

			seconds, err := strconv.Atoi(timeComponents[2])
			if err != nil {
				return 0, fmt.Errorf("invalid second format: %s", timeComponents[2])
			}

			totalSeconds := hours*3600 + minutes*60 + seconds
			return time.Duration(totalSeconds) * time.Second, nil
		}
	}

	// Fall back to Go's duration parser for formats like "90m", "2h30m"
	return time.ParseDuration(duration)
}

// ParseDateRange parses date range expressions like "2024-01-01..2024-01-31" or "last week"
func (p *AdvancedFilterParser) ParseDateRange(rangeStr string) (*DateRangeFilter, error) {
	rangeStr = strings.TrimSpace(rangeStr)

	// Handle relative dates first
	if relativeRange := p.parseRelativeDate(rangeStr); relativeRange != nil {
		return relativeRange, nil
	}

	// Handle range formats: start..end
	if strings.Contains(rangeStr, "..") {
		parts := strings.Split(rangeStr, "..")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid date range format: %s", rangeStr)
		}

		startDate, err := p.parseDate(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %v", err)
		}

		endDate, err := p.parseDate(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid end date: %v", err)
		}

		return &DateRangeFilter{
			Start: startDate,
			End:   endDate,
		}, nil
	}

	// Single date - treat as exact day
	date, err := p.parseDate(rangeStr)
	if err != nil {
		return nil, err
	}

	// Create range for the entire day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	return &DateRangeFilter{
		Start: &startOfDay,
		End:   &endOfDay,
	}, nil
}

// parseDate attempts to parse a date string using multiple formats
func (p *AdvancedFilterParser) parseDate(dateStr string) (*time.Time, error) {
	for _, format := range p.dateFormats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return &date, nil
		}
	}
	return nil, fmt.Errorf("unrecognized date format: %s", dateStr)
}

// parseRelativeDate handles relative date expressions
func (p *AdvancedFilterParser) parseRelativeDate(relativeStr string) *DateRangeFilter {
	now := time.Now()
	relativeStr = strings.ToLower(strings.TrimSpace(relativeStr))

	switch relativeStr {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.Add(24 * time.Hour).Add(-time.Nanosecond)
		return &DateRangeFilter{Start: &start, End: &end}

	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		end := start.Add(24 * time.Hour).Add(-time.Nanosecond)
		return &DateRangeFilter{Start: &start, End: &end}

	case "this week":
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		start := now.AddDate(0, 0, -weekday+1) // Monday
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7).Add(-time.Nanosecond)
		return &DateRangeFilter{Start: &start, End: &end}

	case "last week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := now.AddDate(0, 0, -weekday-6) // Previous Monday
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7).Add(-time.Nanosecond)
		return &DateRangeFilter{Start: &start, End: &end}

	case "this month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return &DateRangeFilter{Start: &start, End: &end}

	case "last month":
		start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return &DateRangeFilter{Start: &start, End: &end}

	case "last 24h", "last 24 hours":
		start := now.Add(-24 * time.Hour)
		return &DateRangeFilter{Start: &start, End: &now}

	case "last 7d", "last 7 days":
		start := now.AddDate(0, 0, -7)
		return &DateRangeFilter{Start: &start, End: &now}

	case "last 30d", "last 30 days":
		start := now.AddDate(0, 0, -30)
		return &DateRangeFilter{Start: &start, End: &now}
	}

	// Handle "last N days/hours/minutes" patterns
	if strings.HasPrefix(relativeStr, "last ") {
		return p.parseLastNPattern(relativeStr, now)
	}

	return nil
}

// parseLastNPattern handles patterns like "last 5 days", "last 2 hours"
func (p *AdvancedFilterParser) parseLastNPattern(pattern string, now time.Time) *DateRangeFilter {
	re := regexp.MustCompile(`^last (\d+)\s*(d|day|days|h|hour|hours|m|min|minute|minutes)s?$`)
	matches := re.FindStringSubmatch(pattern)

	if len(matches) != 3 {
		return nil
	}

	count, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}

	unit := matches[2]
	var start time.Time

	switch unit {
	case "m", "min", "minute", "minutes":
		start = now.Add(time.Duration(-count) * time.Minute)
	case "h", "hour", "hours":
		start = now.Add(time.Duration(-count) * time.Hour)
	case "d", "day", "days":
		start = now.AddDate(0, 0, -count)
	default:
		return nil
	}

	return &DateRangeFilter{Start: &start, End: &now}
}

// Enhanced comparison functions with advanced parsing

/*
TODO(lint): Review unused code - func compareMemorySize is unused

compareMemorySize compares memory sizes with unit parsing
func compareMemorySize(a, b interface{}, op FilterOperator) bool {
	aSize, aErr := parseMemoryValue(a)
	bSize, bErr := parseMemoryValue(b)

	if aErr != nil || bErr != nil {
		// Fall back to string comparison
		return compareStrings(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b), op)
	}

	switch op {
	case OpEquals:
		return aSize == bSize
	case OpNotEquals:
		return aSize != bSize
	case OpGreater:
		return aSize > bSize
	case OpLess:
		return aSize < bSize
	case OpGreaterEq:
		return aSize >= bSize
	case OpLessEq:
		return aSize <= bSize
	default:
		return false
	}
}
*/

// parseDurationValue attempts to parse a duration value
func parseDurationValue(v interface{}) (time.Duration, error) {
	switch val := v.(type) {
	case time.Duration:
		return val, nil
	case string:
		return ParseSlurmDuration(val)
	default:
		return ParseSlurmDuration(fmt.Sprintf("%v", v))
	}
}

// parseMemoryValue attempts to parse a memory value
func parseMemoryValue(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case string:
		return ParseMemorySize(val)
	default:
		return ParseMemorySize(fmt.Sprintf("%v", v))
	}
}

/*
TODO(lint): Review unused code - func compareStrings is unused

compareStrings compares strings with the given operator
func compareStrings(a, b string, op FilterOperator) bool {
	switch op {
	case OpEquals:
		return a == b
	case OpNotEquals:
		return a != b
	case OpContains:
		return strings.Contains(strings.ToLower(a), strings.ToLower(b))
	case OpNotContains:
		return !strings.Contains(strings.ToLower(a), strings.ToLower(b))
	case OpGreater:
		return a > b
	case OpLess:
		return a < b
	case OpGreaterEq:
		return a >= b
	case OpLessEq:
		return a <= b
	case OpRegex:
		re, err := regexp.Compile(b)
		if err != nil {
			return false
		}
		return re.MatchString(a)
	default:
		return false
	}
}
*/

// IsMemoryField checks if a field represents memory
func IsMemoryField(fieldName string) bool {
	memoryFields := []string{"memory", "mem", "realmemory", "allocmem"}
	fieldLower := strings.ToLower(fieldName)
	for _, f := range memoryFields {
		if fieldLower == f {
			return true
		}
	}
	return false
}

// IsDurationField checks if a field represents duration/time
func IsDurationField(fieldName string) bool {
	timeFields := []string{"time", "timelimit", "timeused", "elapsed", "runtime", "walltime"}
	fieldLower := strings.ToLower(fieldName)
	for _, f := range timeFields {
		if fieldLower == f {
			return true
		}
	}
	return false
}

// IsDateField checks if a field represents a date/timestamp
func IsDateField(fieldName string) bool {
	dateFields := []string{"submittime", "starttime", "endtime", "created", "modified", "lastupdate"}
	fieldLower := strings.ToLower(fieldName)
	for _, f := range dateFields {
		if strings.Contains(fieldLower, f) {
			return true
		}
	}
	return false
}
