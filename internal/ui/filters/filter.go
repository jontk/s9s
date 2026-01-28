package filters

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FilterOperator represents comparison operators
type FilterOperator string

const (
	// OpEquals is the equals comparison operator.
	OpEquals FilterOperator = "="
	// OpNotEquals is the not equals comparison operator.
	OpNotEquals FilterOperator = "!="
	// OpContains is the contains comparison operator.
	OpContains FilterOperator = "~"
	// OpNotContains is the not contains comparison operator.
	OpNotContains FilterOperator = "!~"
	// OpGreater is the greater than comparison operator.
	OpGreater FilterOperator = ">"
	// OpLess is the less than comparison operator.
	OpLess FilterOperator = "<"
	// OpGreaterEq is the greater than or equal comparison operator.
	OpGreaterEq FilterOperator = ">="
	// OpLessEq is the less than or equal comparison operator.
	OpLessEq FilterOperator = "<="
	// OpRegex is the regex matching operator.
	OpRegex FilterOperator = "=~"
	// OpIn is the in list operator.
	OpIn FilterOperator = "in"
	// OpNotIn is the not in list operator.
	OpNotIn FilterOperator = "not in"
)

// FilterExpression represents a single filter condition
type FilterExpression struct {
	Field    string
	Operator FilterOperator
	Value    interface{}
}

// Filter represents a complex filter with multiple expressions
type Filter struct {
	Expressions []FilterExpression
	Logic       string // "AND" or "OR"
	Name        string // For saved filters
	Description string
}

// FilterParser parses filter strings into filter expressions
type FilterParser struct {
	fieldAliases map[string]string
}

// NewFilterParser creates a new filter parser
func NewFilterParser() *FilterParser {
	return &FilterParser{
		fieldAliases: map[string]string{
			// Common aliases
			"name":      "Name",
			"user":      "User",
			"state":     "State",
			"partition": "Partition",
			"status":    "State",
			"node":      "NodeList",
			"nodes":     "NodeList",
			"time":      "TimeUsed",
			"timelimit": "TimeLimit",
			"cpu":       "CPUs",
			"cpus":      "CPUs",
			"mem":       "Memory",
			"memory":    "Memory",
			"account":   "Account",
			"qos":       "QoS",
			"priority":  "Priority",
		},
	}
}

// Parse parses a filter string into filter expressions
// Examples:
//   - "state=running"
//   - "user=john state=pending"
//   - "memory>4G cpus>=8"
//   - "name~test partition=gpu"
//   - "state in (running,pending)"
func (p *FilterParser) Parse(filterStr string) (*Filter, error) {
	if filterStr == "" {
		return &Filter{Logic: "AND"}, nil
	}

	filter := &Filter{
		Logic:       "AND",
		Expressions: []FilterExpression{},
	}

	// Pre-process for multi-word operators like "in" and "not in"
	parts := p.smartSplit(filterStr)

	for _, part := range parts {
		expr, err := p.parseExpression(part)
		if err != nil {
			return nil, fmt.Errorf("invalid filter expression '%s': %w", part, err)
		}
		filter.Expressions = append(filter.Expressions, *expr)
	}

	return filter, nil
}

// parseExpression parses a single filter expression
func (p *FilterParser) parseExpression(expr string) (*FilterExpression, error) {
	// Check for "not in" operator FIRST (longer pattern)
	if strings.Contains(expr, " not in ") {
		parts := strings.SplitN(expr, " not in ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid 'not in' expression")
		}
		field := p.normalizeField(strings.TrimSpace(parts[0]))
		valueList := strings.Trim(parts[1], "()")
		values := strings.Split(valueList, ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return &FilterExpression{
			Field:    field,
			Operator: OpNotIn,
			Value:    values,
		}, nil
	}

	// Check for "in" operator
	if strings.Contains(expr, " in ") {
		parts := strings.SplitN(expr, " in ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid 'in' expression")
		}
		field := p.normalizeField(strings.TrimSpace(parts[0]))
		valueList := strings.Trim(parts[1], "()")
		values := strings.Split(valueList, ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return &FilterExpression{
			Field:    field,
			Operator: OpIn,
			Value:    values,
		}, nil
	}

	// Regular operators
	operators := []struct {
		op  string
		typ FilterOperator
	}{
		{"!=", OpNotEquals},
		{"!~", OpNotContains},
		{">=", OpGreaterEq},
		{"<=", OpLessEq},
		{"=~", OpRegex},
		{"=", OpEquals},
		{"~", OpContains},
		{">", OpGreater},
		{"<", OpLess},
	}

	for _, op := range operators {
		if idx := strings.Index(expr, op.op); idx > 0 {
			field := p.normalizeField(strings.TrimSpace(expr[:idx]))
			value := strings.TrimSpace(expr[idx+len(op.op):])
			value = strings.Trim(value, "\"'")

			return &FilterExpression{
				Field:    field,
				Operator: op.typ,
				Value:    p.parseValue(value),
			}, nil
		}
	}

	return nil, fmt.Errorf("no valid operator found")
}

// normalizeField converts field aliases to canonical field names
func (p *FilterParser) normalizeField(field string) string {
	field = strings.ToLower(field)
	if canonical, ok := p.fieldAliases[field]; ok {
		return canonical
	}
	// Capitalize first letter if not found in aliases
	if len(field) > 0 {
		return strings.ToUpper(field[:1]) + field[1:]
	}
	return field
}

// parseValue attempts to parse the value into appropriate type with advanced parsing
func (p *FilterParser) parseValue(value string) interface{} {
	// Try to parse as number
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	// Try to parse as bool
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	// Try to parse as memory size (e.g., "4G", "1024M")
	if m, err := ParseMemorySize(value); err == nil {
		return m
	}
	// Try to parse as SLURM duration (e.g., "2:30:00", "1-12:00:00")
	if d, err := ParseSlurmDuration(value); err == nil {
		return d
	}
	// Try to parse as Go duration (e.g., "30m", "2h")
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	// Return as string
	return value
}

// Evaluate evaluates a filter against a data object
func (f *Filter) Evaluate(data map[string]interface{}) bool {
	if len(f.Expressions) == 0 {
		return true
	}

	if f.Logic == "OR" {
		for _, expr := range f.Expressions {
			if expr.Evaluate(data) {
				return true
			}
		}
		return false
	}

	// Default to AND logic
	for _, expr := range f.Expressions {
		if !expr.Evaluate(data) {
			return false
		}
	}
	return true
}

// Evaluate evaluates a single filter expression
func (e *FilterExpression) Evaluate(data map[string]interface{}) bool {
	value, exists := data[e.Field]
	if !exists {
		return false
	}

	return evaluateOperator(e.Operator, value, e.Value)
}

// evaluateOperator applies the appropriate comparison operator
func evaluateOperator(operator FilterOperator, value, expected interface{}) bool {
	// Handle special cases that require multiple comparisons
	if operator == OpGreaterEq {
		return compareGreater(value, expected) || compareEqual(value, expected)
	}
	if operator == OpLessEq {
		return compareLess(value, expected) || compareEqual(value, expected)
	}

	// Map remaining operators to their evaluators
	evaluators := map[FilterOperator]func() bool{
		OpEquals:      func() bool { return compareEqual(value, expected) },
		OpNotEquals:   func() bool { return !compareEqual(value, expected) },
		OpContains:    func() bool { return contains(value, expected) },
		OpNotContains: func() bool { return !contains(value, expected) },
		OpGreater:     func() bool { return compareGreater(value, expected) },
		OpLess:        func() bool { return compareLess(value, expected) },
		OpRegex:       func() bool { return matchRegex(value, expected) },
		OpIn:          func() bool { return isIn(value, expected) },
		OpNotIn:       func() bool { return !isIn(value, expected) },
	}

	if evaluator, ok := evaluators[operator]; ok {
		return evaluator()
	}
	return false
}

// Helper functions for comparisons

func compareEqual(a, b interface{}) bool {
	// Convert both to strings for comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func contains(haystack, needle interface{}) bool {
	haystackStr := strings.ToLower(fmt.Sprintf("%v", haystack))
	needleStr := strings.ToLower(fmt.Sprintf("%v", needle))
	return strings.Contains(haystackStr, needleStr)
}

func compareGreater(a, b interface{}) bool {
	// Try memory comparison for memory values
	if aSize, aErr := parseMemoryValue(a); aErr == nil {
		if bSize, bErr := parseMemoryValue(b); bErr == nil {
			return aSize > bSize
		}
	}

	// Try duration comparison for time values
	if aDur, aErr := parseDurationValue(a); aErr == nil {
		if bDur, bErr := parseDurationValue(b); bErr == nil {
			return aDur > bDur
		}
	}

	// Try numeric comparison
	if aNum, aOk := toFloat64(a); aOk {
		if bNum, bOk := toFloat64(b); bOk {
			return aNum > bNum
		}
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", a) > fmt.Sprintf("%v", b)
}

func compareLess(a, b interface{}) bool {
	// Try memory comparison for memory values
	if aSize, aErr := parseMemoryValue(a); aErr == nil {
		if bSize, bErr := parseMemoryValue(b); bErr == nil {
			return aSize < bSize
		}
	}

	// Try duration comparison for time values
	if aDur, aErr := parseDurationValue(a); aErr == nil {
		if bDur, bErr := parseDurationValue(b); bErr == nil {
			return aDur < bDur
		}
	}

	// Try numeric comparison
	if aNum, aOk := toFloat64(a); aOk {
		if bNum, bOk := toFloat64(b); bOk {
			return aNum < bNum
		}
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
}

func matchRegex(value, pattern interface{}) bool {
	re, err := regexp.Compile(fmt.Sprintf("%v", pattern))
	if err != nil {
		return false
	}
	return re.MatchString(fmt.Sprintf("%v", value))
}

func isIn(value, list interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	if listSlice, ok := list.([]string); ok {
		for _, item := range listSlice {
			if valueStr == item {
				return true
			}
		}
	}
	return false
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// SplitRespectingQuotes splits a string by spaces while respecting quotes.
func SplitRespectingQuotes(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, r := range s {
		processQuoteChar(&inQuotes, &quoteChar, r, &current, &result)
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// processQuoteChar handles the logic of processing a single character in quote parsing
func processQuoteChar(inQuotes *bool, quoteChar *rune, r rune, current *strings.Builder, result *[]string) {
	switch {
	case !*inQuotes && (r == '"' || r == '\''):
		*inQuotes = true
		*quoteChar = r
	case *inQuotes && r == *quoteChar:
		*inQuotes = false
		*quoteChar = 0
	case !*inQuotes && r == ' ':
		if current.Len() > 0 {
			*result = append(*result, current.String())
			current.Reset()
		}
	default:
		current.WriteRune(r)
	}
}

// smartSplit splits filter string while respecting multi-word operators like "in" and "not in"
func (p *FilterParser) smartSplit(s string) []string {
	// Pre-process to handle multi-word operators
	// Replace " not in " with a placeholder first (longer match) - no spaces in placeholder
	s = strings.ReplaceAll(s, " not in ", "__NOT_IN__")
	// Replace " in " with a placeholder (but not the ones already replaced) - no spaces in placeholder
	s = strings.ReplaceAll(s, " in ", "__IN__")

	// Now split normally
	parts := SplitRespectingQuotes(s)

	// Post-process to restore operators
	for i, part := range parts {
		part = strings.ReplaceAll(part, "__NOT_IN__", " not in ")
		part = strings.ReplaceAll(part, "__IN__", " in ")
		parts[i] = part
	}

	return parts
}
