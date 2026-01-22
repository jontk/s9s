package filters

import (
	"testing"
	"time"
)

func TestParseMemorySize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"4G", 4 * 1024 * 1024 * 1024, false},
		{"1024M", 1024 * 1024 * 1024, false},
		{"512MB", 512 * 1024 * 1024, false},
		{"2TB", 2 * 1024 * 1024 * 1024 * 1024, false},
		{"256KB", 256 * 1024, false},
		{"1B", 1, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"4.5G", int64(4.5 * 1024 * 1024 * 1024), false},
	}

	for _, test := range tests {
		result, err := ParseMemorySize(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input %s, expected %d, got %d", test.input, test.expected, result)
			}
		}
	}
}

func TestParseSlurmDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"2:30:00", 2*time.Hour + 30*time.Minute, false},
		{"1-12:00:00", 24*time.Hour + 12*time.Hour, false},
		{"0:05:30", 5*time.Minute + 30*time.Second, false},
		{"90m", 90 * time.Minute, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, test := range tests {
		result, err := ParseSlurmDuration(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input %s, expected %v, got %v", test.input, test.expected, result)
			}
		}
	}
}

func TestAdvancedFilterParsing(t *testing.T) {
	parser := NewAdvancedFilterParser()

	// Test memory filtering
	filter, err := parser.Parse("memory>4G")
	if err != nil {
		t.Fatalf("Failed to parse memory filter: %v", err)
	}

	if len(filter.Expressions) != 1 {
		t.Fatalf("Expected 1 expression, got %d", len(filter.Expressions))
	}

	expr := filter.Expressions[0]
	if expr.Field != "Memory" {
		t.Errorf("Expected field 'Memory', got '%s'", expr.Field)
	}
	if expr.Operator != OpGreater {
		t.Errorf("Expected operator '>', got '%s'", expr.Operator)
	}

	// Test the parsed value is correct
	expectedSize := int64(4 * 1024 * 1024 * 1024)
	if expr.Value != expectedSize {
		t.Errorf("Expected value %d, got %v", expectedSize, expr.Value)
	}
}

func TestDateRangeFiltering(t *testing.T) {
	parser := NewAdvancedFilterParser()

	// Test relative date parsing
	dateRange, err := parser.ParseDateRange("today")
	if err != nil {
		t.Fatalf("Failed to parse date range 'today': %v", err)
	}

	if dateRange.Start == nil || dateRange.End == nil {
		t.Fatalf("Date range should have both start and end dates")
	}

	// Check that it's actually today
	now := time.Now()
	expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !dateRange.Start.Equal(expectedStart) {
		t.Errorf("Expected start date %v, got %v", expectedStart, *dateRange.Start)
	}

	// Test date range parsing
	dateRange2, err := parser.ParseDateRange("2024-01-01..2024-01-31")
	if err != nil {
		t.Fatalf("Failed to parse date range: %v", err)
	}

	if dateRange2.Start == nil || dateRange2.End == nil {
		t.Fatalf("Date range should have both start and end dates")
	}
}

func TestFieldTypeDetection(t *testing.T) {
	tests := []struct {
		field    string
		isMemory bool
		isTime   bool
		isDate   bool
	}{
		{"memory", true, false, false},
		{"mem", true, false, false},
		{"time", false, true, false},
		{"timelimit", false, true, false},
		{"submittime", false, false, true},
		{"starttime", false, false, true},
		{"name", false, false, false},
		{"user", false, false, false},
	}

	for _, test := range tests {
		if IsMemoryField(test.field) != test.isMemory {
			t.Errorf("IsMemoryField('%s') = %v, expected %v", test.field, IsMemoryField(test.field), test.isMemory)
		}
		if IsDurationField(test.field) != test.isTime {
			t.Errorf("IsDurationField('%s') = %v, expected %v", test.field, IsDurationField(test.field), test.isTime)
		}
		if IsDateField(test.field) != test.isDate {
			t.Errorf("IsDateField('%s') = %v, expected %v", test.field, IsDateField(test.field), test.isDate)
		}
	}
}

func TestComplexFilterExpressions(t *testing.T) {
	parser := NewAdvancedFilterParser()

	// Test complex filter with multiple advanced features
	filter, err := parser.Parse("memory>4G time>=2:30:00 state=running name=~^test_\\d+$")
	if err != nil {
		t.Fatalf("Failed to parse complex filter: %v", err)
	}

	if len(filter.Expressions) != 4 {
		t.Fatalf("Expected 4 expressions, got %d", len(filter.Expressions))
	}

	// Verify memory expression
	memExpr := filter.Expressions[0]
	if memExpr.Field != "Memory" || memExpr.Operator != OpGreater {
		t.Errorf("Memory expression incorrect: field=%s, op=%s", memExpr.Field, memExpr.Operator)
	}

	// Verify time expression
	timeExpr := filter.Expressions[1]
	if timeExpr.Field != "TimeUsed" || timeExpr.Operator != OpGreaterEq {
		t.Errorf("Time expression incorrect: field=%s, op=%s", timeExpr.Field, timeExpr.Operator)
	}

	// Verify regex expression
	regexExpr := filter.Expressions[3]
	if regexExpr.Field != "Name" || regexExpr.Operator != OpRegex {
		t.Errorf("Regex expression incorrect: field=%s, op=%s", regexExpr.Field, regexExpr.Operator)
	}
}
