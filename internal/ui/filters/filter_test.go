package filters

import (
	"testing"
)

func TestFilterOperators(t *testing.T) {
	parser := NewFilterParser()

	testCases := []struct {
		name     string
		filter   string
		data     map[string]interface{}
		expected bool
	}{
		// Test Equals operator
		{
			name:     "equals_string",
			filter:   "name=test",
			data:     map[string]interface{}{"Name": "test"},
			expected: true,
		},
		{
			name:     "equals_number",
			filter:   "cpus=4",
			data:     map[string]interface{}{"CPUs": 4},
			expected: true,
		},
		{
			name:     "equals_false",
			filter:   "name=test",
			data:     map[string]interface{}{"Name": "other"},
			expected: false,
		},

		// Test Not Equals operator
		{
			name:     "not_equals_true",
			filter:   "state!=running",
			data:     map[string]interface{}{"State": "pending"},
			expected: true,
		},
		{
			name:     "not_equals_false",
			filter:   "state!=running",
			data:     map[string]interface{}{"State": "running"},
			expected: false,
		},

		// Test Contains operator
		{
			name:     "contains_true",
			filter:   "name~test",
			data:     map[string]interface{}{"Name": "my_test_job"},
			expected: true,
		},
		{
			name:     "contains_false",
			filter:   "name~test",
			data:     map[string]interface{}{"Name": "production"},
			expected: false,
		},

		// Test Not Contains operator
		{
			name:     "not_contains_true",
			filter:   "name!~test",
			data:     map[string]interface{}{"Name": "production"},
			expected: true,
		},
		{
			name:     "not_contains_false",
			filter:   "name!~test",
			data:     map[string]interface{}{"Name": "my_test_job"},
			expected: false,
		},

		// Test Greater Than operator
		{
			name:     "greater_than_true",
			filter:   "cpus>4",
			data:     map[string]interface{}{"CPUs": 8},
			expected: true,
		},
		{
			name:     "greater_than_false",
			filter:   "cpus>4",
			data:     map[string]interface{}{"CPUs": 2},
			expected: false,
		},

		// Test Less Than operator
		{
			name:     "less_than_true",
			filter:   "cpus<8",
			data:     map[string]interface{}{"CPUs": 4},
			expected: true,
		},
		{
			name:     "less_than_false",
			filter:   "cpus<8",
			data:     map[string]interface{}{"CPUs": 12},
			expected: false,
		},

		// Test Greater Than or Equal operator
		{
			name:     "greater_equal_true",
			filter:   "cpus>=4",
			data:     map[string]interface{}{"CPUs": 4},
			expected: true,
		},
		{
			name:     "greater_equal_false",
			filter:   "cpus>=4",
			data:     map[string]interface{}{"CPUs": 2},
			expected: false,
		},

		// Test Less Than or Equal operator
		{
			name:     "less_equal_true",
			filter:   "cpus<=8",
			data:     map[string]interface{}{"CPUs": 8},
			expected: true,
		},
		{
			name:     "less_equal_false",
			filter:   "cpus<=8",
			data:     map[string]interface{}{"CPUs": 12},
			expected: false,
		},

		// Test Regex operator
		{
			name:     "regex_true",
			filter:   "name=~^test.*$",
			data:     map[string]interface{}{"Name": "test_job_123"},
			expected: true,
		},
		{
			name:     "regex_false",
			filter:   "name=~^test.*$",
			data:     map[string]interface{}{"Name": "prod_job_123"},
			expected: false,
		},

		// Test In operator
		{
			name:     "in_true",
			filter:   "state in (running,pending)",
			data:     map[string]interface{}{"State": "running"},
			expected: true,
		},
		{
			name:     "in_false",
			filter:   "state in (running,pending)",
			data:     map[string]interface{}{"State": "completed"},
			expected: false,
		},

		// Test Not In operator
		{
			name:     "not_in_true",
			filter:   "state not in (running,pending)",
			data:     map[string]interface{}{"State": "completed"},
			expected: true,
		},
		{
			name:     "not_in_false",
			filter:   "state not in (running,pending)",
			data:     map[string]interface{}{"State": "running"},
			expected: false,
		},

		// Test multiple conditions (AND logic)
		{
			name:     "multiple_and_true",
			filter:   "state=running cpus>4",
			data:     map[string]interface{}{"State": "running", "CPUs": 8},
			expected: true,
		},
		{
			name:     "multiple_and_false",
			filter:   "state=running cpus>4",
			data:     map[string]interface{}{"State": "running", "CPUs": 2},
			expected: false,
		},

		// Test field aliases
		{
			name:     "alias_user",
			filter:   "user=john",
			data:     map[string]interface{}{"User": "john"},
			expected: true,
		},
		{
			name:     "alias_mem",
			filter:   "mem>1024",
			data:     map[string]interface{}{"Memory": 2048},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := parser.Parse(tc.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tc.filter, err)
			}

			result := filter.Evaluate(tc.data)
			if result != tc.expected {
				t.Errorf("Filter '%s' with data %v: expected %v, got %v",
					tc.filter, tc.data, tc.expected, result)
			}
		})
	}
}

func TestFilterParserErrors(t *testing.T) {
	parser := NewFilterParser()

	errorCases := []struct {
		name   string
		filter string
	}{
		{
			name:   "no_operator",
			filter: "name_without_operator",
		},
		{
			name:   "invalid_operator",
			filter: "state @ running", // Invalid operator
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.Parse(tc.filter)
			if err == nil {
				t.Errorf("Expected error for filter '%s', but got none", tc.filter)
			}
		})
	}
}

func TestFieldAliases(t *testing.T) {
	parser := NewFilterParser()

	aliases := map[string]string{
		"name":      "Name",
		"user":      "User",
		"state":     "State",
		"partition": "Partition",
		"status":    "State",
		"node":      "NodeList",
		"nodes":     "NodeList",
		"cpu":       "CPUs",
		"cpus":      "CPUs",
		"mem":       "Memory",
		"memory":    "Memory",
		"account":   "Account",
		"qos":       "QoS",
		"priority":  "Priority",
	}

	for alias, canonical := range aliases {
		t.Run(alias, func(t *testing.T) {
			normalized := parser.normalizeField(alias)
			if normalized != canonical {
				t.Errorf("Expected alias '%s' to normalize to '%s', got '%s'",
					alias, canonical, normalized)
			}
		})
	}
}

func TestComplexFilters(t *testing.T) {
	parser := NewFilterParser()

	testCases := []struct {
		name     string
		filter   string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:   "complex_job_filter",
			filter: "state=running user=john cpus>=4 mem>1024",
			data: map[string]interface{}{
				"State":  "running",
				"User":   "john",
				"CPUs":   8,
				"Memory": 2048,
			},
			expected: true,
		},
		{
			name:   "complex_node_filter",
			filter: "state!=down partition~gpu cpus>16",
			data: map[string]interface{}{
				"State":     "idle",
				"Partition": "gpu_partition",
				"CPUs":      32,
			},
			expected: true,
		},
		{
			name:   "in_operator_with_multiple_values",
			filter: "state in (running,pending,completing)",
			data: map[string]interface{}{
				"State": "completing",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := parser.Parse(tc.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tc.filter, err)
			}

			result := filter.Evaluate(tc.data)
			if result != tc.expected {
				t.Errorf("Filter '%s' with data %v: expected %v, got %v",
					tc.filter, tc.data, tc.expected, result)
			}
		})
	}
}
