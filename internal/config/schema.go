package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ConfigField represents a configuration field with metadata
type ConfigField struct {
	Key         string       `json:"key"`
	Label       string       `json:"label"`
	Description string       `json:"description"`
	Type        FieldType    `json:"type"`
	Required    bool         `json:"required"`
	Default     interface{}  `json:"default"`
	Options     []string     `json:"options,omitempty"`  // For select/enum types
	Min         *float64     `json:"min,omitempty"`      // For numeric types
	Max         *float64     `json:"max,omitempty"`      // For numeric types
	Pattern     string       `json:"pattern,omitempty"`  // For string validation
	Group       string       `json:"group"`              // For UI grouping
	Order       int          `json:"order"`              // For UI ordering
	Sensitive   bool         `json:"sensitive"`          // For password fields
	Depends     []Dependency `json:"depends,omitempty"`  // Conditional fields
	Examples    []string     `json:"examples,omitempty"` // Usage examples
}

// FieldType represents the type of a configuration field
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInt      FieldType = "int"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBool     FieldType = "bool"
	FieldTypeDuration FieldType = "duration"
	FieldTypeSelect   FieldType = "select"
	FieldTypeArray    FieldType = "array"
	FieldTypeObject   FieldType = "object"
	FieldTypeContext  FieldType = "context"  // Special type for contexts
	FieldTypeShortcut FieldType = "shortcut" // Special type for shortcuts
	FieldTypePlugin   FieldType = "plugin"   // Special type for plugins
	FieldTypeAlias    FieldType = "alias"    // Special type for aliases
)

// Dependency represents a field dependency
type Dependency struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// ValidationResult represents the result of field validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ConfigSchema defines the complete configuration schema
type ConfigSchema struct {
	Groups []ConfigGroup `json:"groups"`
	Fields []ConfigField `json:"fields"`
}

// ConfigGroup represents a group of related configuration fields
type ConfigGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	Order       int    `json:"order"`
}

// GetConfigSchema returns the complete configuration schema
func GetConfigSchema() *ConfigSchema {
	return &ConfigSchema{
		Groups: []ConfigGroup{
			{ID: "general", Name: "General", Description: "Basic application settings", Icon: "⚙️", Order: 1},
			{ID: "ui", Name: "User Interface", Description: "UI appearance and behavior", Icon: "🎨", Order: 2},
			{ID: "cluster", Name: "Cluster Contexts", Description: "SLURM cluster connections", Icon: "🔗", Order: 3},
			{ID: "views", Name: "View Settings", Description: "Table views and display options", Icon: "📊", Order: 4},
			{ID: "features", Name: "Features", Description: "Feature flags and advanced options", Icon: "🚀", Order: 5},
			{ID: "shortcuts", Name: "Keyboard Shortcuts", Description: "Custom key bindings", Icon: "⌨️", Order: 6},
			{ID: "aliases", Name: "Command Aliases", Description: "Command shortcuts", Icon: "📝", Order: 7},
			{ID: "plugins", Name: "Plugins", Description: "External plugin configuration", Icon: "🔌", Order: 8},
		},
		Fields: getConfigFields(),
	}
}

// getConfigFields returns all configuration fields with metadata
func getConfigFields() []ConfigField {
	return []ConfigField{
		// General settings
		{
			Key:         "refreshRate",
			Label:       "Refresh Rate",
			Description: "How often to refresh data from the cluster",
			Type:        FieldTypeDuration,
			Required:    true,
			Default:     "2s",
			Group:       "general",
			Order:       1,
			Examples:    []string{"1s", "5s", "10s", "30s"},
		},
		{
			Key:         "maxRetries",
			Label:       "Max Retries",
			Description: "Maximum number of API call retries",
			Type:        FieldTypeInt,
			Required:    true,
			Default:     3,
			Min:         &[]float64{0}[0],
			Max:         &[]float64{10}[0],
			Group:       "general",
			Order:       2,
		},
		{
			Key:         "currentContext",
			Label:       "Current Context",
			Description: "Active cluster context to use",
			Type:        FieldTypeString,
			Required:    true,
			Default:     "default",
			Group:       "general",
			Order:       3,
		},
		{
			Key:         "useMockClient",
			Label:       "Use Mock Client",
			Description: "Use mock data instead of real cluster connection (for development)",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     false,
			Group:       "general",
			Order:       4,
		},

		// UI settings
		{
			Key:         "ui.skin",
			Label:       "Theme",
			Description: "Color theme for the application",
			Type:        FieldTypeSelect,
			Required:    true,
			Default:     "default",
			Options:     []string{"default", "monokai", "dracula", "solarized", "github"},
			Group:       "ui",
			Order:       1,
		},
		{
			Key:         "ui.enableMouse",
			Label:       "Enable Mouse",
			Description: "Allow mouse interaction in the terminal UI",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     true,
			Group:       "ui",
			Order:       2,
		},
		{
			Key:         "ui.logoless",
			Label:       "Hide Logo",
			Description: "Hide the application logo in the header",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     false,
			Group:       "ui",
			Order:       3,
		},
		{
			Key:         "ui.statusless",
			Label:       "Hide Status Bar",
			Description: "Hide the status bar at the bottom",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     false,
			Group:       "ui",
			Order:       4,
		},
		{
			Key:         "ui.noIcons",
			Label:       "Disable Icons",
			Description: "Show text instead of icons (for terminal compatibility)",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     false,
			Group:       "ui",
			Order:       5,
		},

		// View settings
		{
			Key:         "views.jobs.columns",
			Label:       "Job Columns",
			Description: "Columns to display in the jobs view",
			Type:        FieldTypeArray,
			Required:    true,
			Default:     []string{"id", "name", "user", "state", "time", "nodes", "priority"},
			Options:     []string{"id", "name", "user", "account", "state", "time", "nodes", "cpus", "memory", "priority", "partition", "qos"},
			Group:       "views",
			Order:       1,
		},
		{
			Key:         "views.jobs.showOnlyActive",
			Label:       "Show Only Active Jobs",
			Description: "Hide completed and failed jobs by default",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     true,
			Group:       "views",
			Order:       2,
		},
		{
			Key:         "views.jobs.defaultSort",
			Label:       "Default Sort Column",
			Description: "Column to sort jobs by default",
			Type:        FieldTypeSelect,
			Required:    true,
			Default:     "time",
			Options:     []string{"id", "name", "user", "state", "time", "priority"},
			Group:       "views",
			Order:       3,
		},
		{
			Key:         "views.jobs.maxJobs",
			Label:       "Max Jobs to Display",
			Description: "Maximum number of jobs to show in the table",
			Type:        FieldTypeInt,
			Required:    true,
			Default:     1000,
			Min:         &[]float64{10}[0],
			Max:         &[]float64{10000}[0],
			Group:       "views",
			Order:       4,
		},
		{
			Key:         "views.nodes.groupBy",
			Label:       "Group Nodes By",
			Description: "How to group nodes in the nodes view",
			Type:        FieldTypeSelect,
			Required:    true,
			Default:     "partition",
			Options:     []string{"partition", "state", "feature", "none"},
			Group:       "views",
			Order:       5,
		},
		{
			Key:         "views.nodes.showUtilization",
			Label:       "Show Utilization",
			Description: "Display CPU and memory utilization bars",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     true,
			Group:       "views",
			Order:       6,
		},

		// Features
		{
			Key:         "features.streaming",
			Label:       "Enable Streaming",
			Description: "Real-time updates via WebSocket connections",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     true,
			Group:       "features",
			Order:       1,
		},
		{
			Key:         "features.pulseye",
			Label:       "Enable Health Scanner",
			Description: "Automated cluster health monitoring",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     true,
			Group:       "features",
			Order:       2,
		},
		{
			Key:         "features.xray",
			Label:       "Enable X-Ray Mode",
			Description: "Deep inspection and debugging features",
			Type:        FieldTypeBool,
			Required:    false,
			Default:     false,
			Group:       "features",
			Order:       3,
		},

		// Special fields for complex types
		{
			Key:         "contexts",
			Label:       "Cluster Contexts",
			Description: "SLURM cluster connection configurations",
			Type:        FieldTypeContext,
			Required:    false,
			Group:       "cluster",
			Order:       1,
		},
		{
			Key:         "shortcuts",
			Label:       "Keyboard Shortcuts",
			Description: "Custom key bindings for actions",
			Type:        FieldTypeShortcut,
			Required:    false,
			Group:       "shortcuts",
			Order:       1,
		},
		{
			Key:         "aliases",
			Label:       "Command Aliases",
			Description: "Short names for common commands",
			Type:        FieldTypeAlias,
			Required:    false,
			Group:       "aliases",
			Order:       1,
		},
		{
			Key:         "plugins",
			Label:       "Plugins",
			Description: "External plugin configurations",
			Type:        FieldTypePlugin,
			Required:    false,
			Group:       "plugins",
			Order:       1,
		},
	}
}

// ValidateField validates a single configuration field value
func (cf *ConfigField) ValidateField(value interface{}) ValidationResult {
	result := ValidationResult{Valid: true}

	// Check if required field is present
	if cf.Required && (value == nil || value == "") {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("%s is required", cf.Label))
		return result
	}

	// Skip validation if field is empty and not required
	if value == nil || value == "" {
		return result
	}

	// Type-specific validation
	switch cf.Type {
	case FieldTypeString:
		if err := cf.validateString(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	case FieldTypeInt:
		if err := cf.validateInt(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	case FieldTypeFloat:
		if err := cf.validateFloat(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	case FieldTypeBool:
		if err := cf.validateBool(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	case FieldTypeDuration:
		if err := cf.validateDuration(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	case FieldTypeSelect:
		if err := cf.validateSelect(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	case FieldTypeArray:
		if err := cf.validateArray(value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result
}

// validateString validates string field values
func (cf *ConfigField) validateString(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a string", cf.Label)
	}

	// Pattern validation
	if cf.Pattern != "" {
		matched, err := regexp.MatchString(cf.Pattern, str)
		if err != nil {
			return fmt.Errorf("invalid pattern for %s: %v", cf.Label, err)
		}
		if !matched {
			return fmt.Errorf("%s does not match required pattern", cf.Label)
		}
	}

	return nil
}

// validateInt validates integer field values
func (cf *ConfigField) validateInt(value interface{}) error {
	var num int64

	switch v := value.(type) {
	case int:
		num = int64(v)
	case int64:
		num = v
	case float64:
		num = int64(v)
	case string:
		var err error
		num, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be a valid integer", cf.Label)
		}
	default:
		return fmt.Errorf("%s must be an integer", cf.Label)
	}

	// Range validation
	if cf.Min != nil && float64(num) < *cf.Min {
		return fmt.Errorf("%s must be at least %.0f", cf.Label, *cf.Min)
	}
	if cf.Max != nil && float64(num) > *cf.Max {
		return fmt.Errorf("%s must be at most %.0f", cf.Label, *cf.Max)
	}

	return nil
}

// validateFloat validates float field values
func (cf *ConfigField) validateFloat(value interface{}) error {
	var num float64

	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case string:
		var err error
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("%s must be a valid number", cf.Label)
		}
	default:
		return fmt.Errorf("%s must be a number", cf.Label)
	}

	// Range validation
	if cf.Min != nil && num < *cf.Min {
		return fmt.Errorf("%s must be at least %.2f", cf.Label, *cf.Min)
	}
	if cf.Max != nil && num > *cf.Max {
		return fmt.Errorf("%s must be at most %.2f", cf.Label, *cf.Max)
	}

	return nil
}

// validateBool validates boolean field values
func (cf *ConfigField) validateBool(value interface{}) error {
	switch v := value.(type) {
	case bool:
		return nil
	case string:
		_, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("%s must be true or false", cf.Label)
		}
	default:
		return fmt.Errorf("%s must be true or false", cf.Label)
	}
	return nil
}

// validateDuration validates duration field values
func (cf *ConfigField) validateDuration(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a duration string (e.g., '5s', '2m')", cf.Label)
	}

	_, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("%s must be a valid duration (e.g., '5s', '2m', '1h')", cf.Label)
	}

	return nil
}

// validateSelect validates select field values
func (cf *ConfigField) validateSelect(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a string", cf.Label)
	}

	// Check if value is in options
	for _, option := range cf.Options {
		if str == option {
			return nil
		}
	}

	return fmt.Errorf("%s must be one of: %s", cf.Label, strings.Join(cf.Options, ", "))
}

// validateArray validates array field values
func (cf *ConfigField) validateArray(value interface{}) error {
	// Use reflection to check if value is a slice
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return fmt.Errorf("%s must be an array", cf.Label)
	}

	// If we have options, validate each element
	if len(cf.Options) > 0 {
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			str, ok := elem.(string)
			if !ok {
				return fmt.Errorf("%s array elements must be strings", cf.Label)
			}

			valid := false
			for _, option := range cf.Options {
				if str == option {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("%s array element '%s' must be one of: %s", cf.Label, str, strings.Join(cf.Options, ", "))
			}
		}
	}

	return nil
}

// GetFieldByKey finds a configuration field by its key
func (cs *ConfigSchema) GetFieldByKey(key string) *ConfigField {
	for _, field := range cs.Fields {
		if field.Key == key {
			return &field
		}
	}
	return nil
}

// GetFieldsByGroup returns all fields in a specific group
func (cs *ConfigSchema) GetFieldsByGroup(groupID string) []ConfigField {
	var fields []ConfigField
	for _, field := range cs.Fields {
		if field.Group == groupID {
			fields = append(fields, field)
		}
	}
	return fields
}

// ValidateConfig validates an entire configuration against the schema
func (cs *ConfigSchema) ValidateConfig(config *Config) map[string]ValidationResult {
	results := make(map[string]ValidationResult)

	// Use reflection to validate all fields
	configValue := reflect.ValueOf(config).Elem()
	cs.validateStruct(configValue, "", results)

	return results
}

// validateStruct recursively validates a struct using reflection
func (cs *ConfigSchema) validateStruct(structValue reflect.Value, prefix string, results map[string]ValidationResult) {
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Skip unexported or computed fields
		if !field.CanInterface() || fieldType.Tag.Get("mapstructure") == "-" {
			continue
		}

		// Get the mapstructure tag or use field name
		fieldName := fieldType.Tag.Get("mapstructure")
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}

		key := fieldName
		if prefix != "" {
			key = prefix + "." + fieldName
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			cs.validateStruct(field, key, results)
			continue
		}

		// Find schema field and validate
		if schemaField := cs.GetFieldByKey(key); schemaField != nil {
			value := field.Interface()
			results[key] = schemaField.ValidateField(value)
		}
	}
}
