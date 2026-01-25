package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/fileperms"
)


// ValidationError represents a configuration error
type ValidationError struct {
	Field       string
	Message     string
	Severity    ValidationSeverity
	Suggestion  string
	AutoFixable bool
}

// ValidationWarning represents a configuration warning
type ValidationWarning struct {
	Field   string
	Message string
	Impact  string
}

// ValidationFix represents an automatic fix that can be applied
type ValidationFix struct {
	Field       string
	Description string
	OldValue    interface{}
	NewValue    interface{}
	Applied     bool
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []ValidationWarning
	Fixes    []ValidationFix
}

//nolint:revive // type alias for backward compatibility
type ConfigValidationResult = ValidationResult

// ValidationSeverity represents the severity of a validation issue
type ValidationSeverity int

const (
	// SeverityError is the error severity level.
	SeverityError ValidationSeverity = iota
	// SeverityWarning is the warning severity level.
	SeverityWarning
	// SeverityInfo is the info severity level.
	SeverityInfo
)

// Validator validates and fixes s9s configuration
type Validator struct {
	config  *Config
	result  *ValidationResult
	autoFix bool
	// TODO(lint): Review unused code - field strict is unused
	// strict    bool
}

//nolint:revive // type alias for backward compatibility
type ConfigValidator = Validator

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(config *Config, autoFix bool) *Validator {
	return &Validator{
		config:  config,
		autoFix: autoFix,
		result: &ValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationWarning{},
			Fixes:    []ValidationFix{},
		},
	}
}

// Validate performs comprehensive configuration validation
func (v *Validator) Validate() *ValidationResult {
	debug.Logger.Printf("Starting configuration validation (autoFix: %v)", v.autoFix)

	// Basic structure validation
	v.validateBasicStructure()

	// Context validation
	v.validateContexts()

	// Cluster validation
	v.validateClusters()

	// Authentication validation
	v.validateAuthentication()

	// Performance settings validation
	v.validatePerformanceSettings()

	// Security settings validation
	v.validateSecurity()

	// Path validation
	v.validatePaths()

	// Environment variable validation
	v.validateEnvironmentVariables()

	// Final assessment
	v.result.Valid = len(v.result.Errors) == 0

	debug.Logger.Printf("Configuration validation completed: valid=%v, errors=%d, warnings=%d, fixes=%d",
		v.result.Valid, len(v.result.Errors), len(v.result.Warnings), len(v.result.Fixes))

	return v.result
}

// validateBasicStructure validates basic configuration structure
func (v *Validator) validateBasicStructure() {
	if v.config == nil {
		v.addError("config", "Configuration is nil", "Initialize configuration", true)
		return
	}

	// Note: Version field not part of current Config struct - skipping validation

	// Refresh rate validation
	if v.config.RefreshRate == "" {
		v.fix("refresh_rate", "Missing refresh rate", "", "30s")
	} else if !v.isValidDuration(v.config.RefreshRate) {
		v.fix("refresh_rate", "Invalid refresh rate format", v.config.RefreshRate, "30s")
	}

	// Current context validation
	if v.config.CurrentContext == "" {
		if len(v.config.Contexts) > 0 {
			v.fix("current_context", "No current context set", "", v.config.Contexts[0].Name)
		} else {
			v.addError("current_context", "No current context and no contexts available", "Add at least one context", false)
		}
	}
}

// validateContexts validates all contexts
func (v *Validator) validateContexts() {
	if len(v.config.Contexts) == 0 {
		v.addError("contexts", "No contexts defined", "Add at least one cluster context", false)
		return
	}

	contextNames := make(map[string]bool)
	for i, context := range v.config.Contexts {
		contextPath := fmt.Sprintf("contexts[%d]", i)

		// Name validation
		if context.Name == "" {
			v.fix(fmt.Sprintf("%s.name", contextPath), "Missing context name", "", fmt.Sprintf("context-%d", i))
			context.Name = fmt.Sprintf("context-%d", i)
		}

		// Check for duplicate names
		if contextNames[context.Name] {
			v.addError(fmt.Sprintf("%s.name", contextPath),
				fmt.Sprintf("Duplicate context name: %s", context.Name),
				"Use unique context names", false)
		}
		contextNames[context.Name] = true

		// Validate cluster configuration within context
		v.validateClusterInContext(context, contextPath)
	}

	// Validate current context exists
	if v.config.CurrentContext != "" {
		found := false
		for _, context := range v.config.Contexts {
			if context.Name == v.config.CurrentContext {
				found = true
				break
			}
		}
		if !found {
			v.fix("current_context",
				fmt.Sprintf("Current context '%s' not found", v.config.CurrentContext),
				v.config.CurrentContext, v.config.Contexts[0].Name)
		}
	}
}

// validateClusters validates cluster configurations
func (v *Validator) validateClusters() {
	for i, context := range v.config.Contexts {
		v.validateCluster(context.Cluster, fmt.Sprintf("contexts[%d].cluster", i))
	}
}

// validateCluster validates a single cluster configuration
func (v *Validator) validateCluster(cluster ClusterConfig, basePath string) {
	// Endpoint validation (main connection method)
	if cluster.Endpoint == "" {
		v.addError(fmt.Sprintf("%s.endpoint", basePath),
			"No endpoint specified",
			"Provide SLURM REST API endpoint", false)
	} else if !v.isValidURL(cluster.Endpoint) {
		v.addError(fmt.Sprintf("%s.endpoint", basePath),
			fmt.Sprintf("Invalid endpoint URL: %s", cluster.Endpoint),
			"Use valid URL format (https://host:port)", false)
	}

	// API version validation
	if cluster.APIVersion == "" {
		v.fix(fmt.Sprintf("%s.api_version", basePath), "Missing API version", "", "v0.0.43")
	}

	// Timeout validation
	if cluster.Timeout != "" && !v.isValidDuration(cluster.Timeout) {
		v.fix(fmt.Sprintf("%s.timeout", basePath), "Invalid timeout format", cluster.Timeout, "30s")
	}

	// Token validation (if present)
	if cluster.Token != "" && len(cluster.Token) < 10 {
		v.addWarning(fmt.Sprintf("%s.token", basePath),
			"Token appears to be too short",
			"Verify token is valid SLURM JWT")
	}
}

// validateClusterInContext validates cluster within a context
func (v *Validator) validateClusterInContext(context ContextConfig, contextPath string) {
	v.validateCluster(context.Cluster, fmt.Sprintf("%s.cluster", contextPath))
}

// validateAuthentication validates authentication settings
func (v *Validator) validateAuthentication() {
	// This would validate auth configurations if they were part of the main config
	// For now, we'll add basic validation for auth-related metadata

	// Basic token validation for existing ClusterConfig
	for i, context := range v.config.Contexts {
		if context.Cluster.Token != "" {
			// Basic JWT token validation
			parts := strings.Split(context.Cluster.Token, ".")
			if len(parts) != 3 {
				v.addWarning(fmt.Sprintf("contexts[%d].cluster.token", i),
					"Token format doesn't appear to be JWT",
					"Verify token format is correct for SLURM")
			}
		}
	}
}

// validatePerformanceSettings validates performance-related settings
func (v *Validator) validatePerformanceSettings() {
	// Refresh rate validation
	if v.config.RefreshRate != "" {
		if duration, err := time.ParseDuration(v.config.RefreshRate); err != nil {
			v.fix("refresh_rate", "Invalid refresh rate duration", v.config.RefreshRate, "30s")
		} else if duration < time.Second {
			v.addWarning("refresh_rate",
				"Very fast refresh rate may impact performance",
				"Consider using 5s or higher for production")
		} else if duration > 10*time.Minute {
			v.addWarning("refresh_rate",
				"Very slow refresh rate may show stale data",
				"Consider using 5m or lower for better user experience")
		}
	}

	// Mock client warning
	if v.config.UseMockClient {
		v.addWarning("use_mock_client",
			"Using mock client - no real SLURM connection",
			"Disable for production use")
	}
}

// validateSecurity validates security settings
func (v *Validator) validateSecurity() {
	for i, context := range v.config.Contexts {
		cluster := context.Cluster

		// Check for insecure configurations
		if cluster.Endpoint != "" && strings.HasPrefix(cluster.Endpoint, "http://") {
			v.addWarning(fmt.Sprintf("contexts[%d].cluster.endpoint", i),
				"Using unencrypted HTTP connection",
				"Consider using HTTPS for security")
		}

		// Insecure flag warnings
		if cluster.Insecure {
			v.addWarning(fmt.Sprintf("contexts[%d].cluster.insecure", i),
				"TLS verification disabled",
				"Enable TLS verification for production")
		}
	}
}

// validatePaths validates file and directory paths
func (v *Validator) validatePaths() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		v.addWarning("paths", "Cannot determine home directory", "Some features may not work")
		return
	}

	configDir := filepath.Join(homeDir, ".s9s")
	v.validateConfigDirectory(configDir)
	v.validateCacheDirectory(configDir)
	v.validateLogsDirectory(configDir)
}

// validateConfigDirectory checks and creates the config directory
func (v *Validator) validateConfigDirectory(configDir string) {
	if v.directoryExists(configDir) {
		return
	}

	if v.autoFix {
		if err := os.MkdirAll(configDir, fileperms.ConfigDir); err != nil {
			v.addError("paths",
				fmt.Sprintf("Cannot create config directory: %s", configDir),
				"Create directory manually with proper permissions", false)
		} else {
			v.result.Fixes = append(v.result.Fixes, ValidationFix{
				Field:       "paths.config_dir",
				Description: "Created missing config directory",
				OldValue:    "missing",
				NewValue:    configDir,
				Applied:     true,
			})
		}
	} else {
		v.addError("paths",
			fmt.Sprintf("Config directory does not exist: %s", configDir),
			"Run 's9s setup' or create directory manually", true)
	}
}

// validateCacheDirectory checks and creates the cache directory
func (v *Validator) validateCacheDirectory(configDir string) {
	if !v.autoFix {
		return
	}

	cacheDir := filepath.Join(configDir, "cache")
	if !v.directoryExists(cacheDir) {
		if err := os.MkdirAll(cacheDir, fileperms.DirUserOnly); err == nil {
			v.result.Fixes = append(v.result.Fixes, ValidationFix{
				Field:       "paths.cache_dir",
				Description: "Created cache directory",
				OldValue:    "missing",
				NewValue:    cacheDir,
				Applied:     true,
			})
		}
	}
}

// validateLogsDirectory checks and creates the logs directory
func (v *Validator) validateLogsDirectory(configDir string) {
	if !v.autoFix {
		return
	}

	logsDir := filepath.Join(configDir, "logs")
	if !v.directoryExists(logsDir) {
		if err := os.MkdirAll(logsDir, fileperms.LogDir); err == nil {
			v.result.Fixes = append(v.result.Fixes, ValidationFix{
				Field:       "paths.logs_dir",
				Description: "Created logs directory",
				OldValue:    "missing",
				NewValue:    logsDir,
				Applied:     true,
			})
		}
	}
}

// validateEnvironmentVariables validates environment variable references
func (v *Validator) validateEnvironmentVariables() {
	// Check for common environment variables that should be set
	requiredEnvVars := []string{"USER", "HOME"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			v.addWarning("environment",
				fmt.Sprintf("Environment variable %s is not set", envVar),
				"Some features may not work correctly")
		}
	}

	// Check for SLURM-related environment variables
	slurmEnvVars := []string{"SLURM_CONF", "SLURM_CONTROLLER_HOST"}
	foundSlurmEnv := false
	for _, envVar := range slurmEnvVars {
		if os.Getenv(envVar) != "" {
			foundSlurmEnv = true
			break
		}
	}

	if !foundSlurmEnv && !v.config.UseMockClient {
		v.addWarning("environment",
			"No SLURM environment variables detected",
			"Consider setting SLURM_CONF or SLURM_CONTROLLER_HOST")
	}
}

// Helper methods

// addError adds a validation error
func (v *Validator) addError(field, message, suggestion string, autoFixable bool) {
	v.result.Errors = append(v.result.Errors, ValidationError{
		Field:       field,
		Message:     message,
		Severity:    SeverityError,
		Suggestion:  suggestion,
		AutoFixable: autoFixable,
	})
}

// addWarning adds a validation warning
func (v *Validator) addWarning(field, message, impact string) {
	v.result.Warnings = append(v.result.Warnings, ValidationWarning{
		Field:   field,
		Message: message,
		Impact:  impact,
	})
}

// fix applies an automatic fix if autoFix is enabled
func (v *Validator) fix(field, description string, oldValue, newValue interface{}) {
	fix := ValidationFix{
		Field:       field,
		Description: description,
		OldValue:    oldValue,
		NewValue:    newValue,
		Applied:     false,
	}

	if v.autoFix {
		// Apply the fix to the configuration
		v.applyFix(field, newValue)
		fix.Applied = true
	}

	v.result.Fixes = append(v.result.Fixes, fix)
}

// applyFix applies a fix to the configuration
func (v *ConfigValidator) applyFix(field string, newValue interface{}) {
	switch field {
	case "refresh_rate":
		v.config.RefreshRate = newValue.(string)
	case "current_context":
		v.config.CurrentContext = newValue.(string)
		// Add more cases as needed for different fields
	}
}

// Validation helper methods

// isValidDuration checks if a string is a valid duration
func (v *Validator) isValidDuration(duration string) bool {
	_, err := time.ParseDuration(duration)
	return err == nil
}

// isValidURL checks if a string is a valid URL
func (v *Validator) isValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

/*
TODO(lint): Review unused code - func (*ConfigValidator).fileExists is unused

fileExists checks if a file exists
func (v *ConfigValidator) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
*/

// directoryExists checks if a directory exists
func (v *Validator) directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

/*
TODO(lint): Review unused code - func (*ConfigValidator).contains is unused

contains checks if a slice contains a string
func (v *ConfigValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
*/

// ValidateAndFix validates configuration and optionally applies fixes
func ValidateAndFix(config *Config, autoFix bool) *ValidationResult {
	validator := NewConfigValidator(config, autoFix)
	return validator.Validate()
}

// PrintValidationResult prints a formatted validation result
func PrintValidationResult(result *ValidationResult, verbose bool) {
	printValidationStatus(result.Valid)
	printErrorsSection(result.Errors)
	printWarningsSection(result.Warnings, verbose)
	printFixesSection(result.Fixes, verbose)
	printFinalMessage(result)
}

// printValidationStatus prints the overall validation status
func printValidationStatus(valid bool) {
	if valid {
		fmt.Printf("✅ Configuration is valid\n")
	} else {
		fmt.Printf("❌ Configuration has issues\n")
	}
}

// printErrorsSection prints the errors section
func printErrorsSection(errors []ValidationError) {
	if len(errors) == 0 {
		return
	}

	fmt.Printf("\n🚨 Errors (%d):\n", len(errors))
	for _, err := range errors {
		fmt.Printf("   • [%s] %s\n", err.Field, err.Message)
		if err.Suggestion != "" {
			fmt.Printf("     💡 %s\n", err.Suggestion)
		}
	}
}

// printWarningsSection prints the warnings section
func printWarningsSection(warnings []ValidationWarning, verbose bool) {
	if len(warnings) == 0 {
		return
	}

	fmt.Printf("\n⚠️  Warnings (%d):\n", len(warnings))
	for _, warn := range warnings {
		fmt.Printf("   • [%s] %s\n", warn.Field, warn.Message)
		if warn.Impact != "" && verbose {
			fmt.Printf("     📄 Impact: %s\n", warn.Impact)
		}
	}
}

// printFixesSection prints the fixes section
func printFixesSection(fixes []ValidationFix, verbose bool) {
	if len(fixes) == 0 {
		return
	}

	fmt.Printf("\n🔧 Fixes (%d):\n", len(fixes))
	for _, fix := range fixes {
		status := getFixStatus(fix.Applied)
		fmt.Printf("   • [%s] %s (%s)\n", fix.Field, fix.Description, status)
		if verbose && fix.OldValue != fix.NewValue {
			fmt.Printf("     📝 %v → %v\n", fix.OldValue, fix.NewValue)
		}
	}
}

// getFixStatus returns the status string for a fix
func getFixStatus(applied bool) string {
	if applied {
		return "applied"
	}
	return "available"
}

// printFinalMessage prints a final message if configuration is perfect
func printFinalMessage(result *ValidationResult) {
	if result.Valid && len(result.Warnings) == 0 && len(result.Fixes) == 0 {
		fmt.Printf("\n🎉 Configuration is perfect!\n")
	}
}
