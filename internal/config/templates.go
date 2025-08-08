package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigTemplate represents a configuration template
type ConfigTemplate struct {
	Name        string
	Description string
	Category    string
	Template    string
	Variables   map[string]string
}

// TemplateManager manages configuration templates
type TemplateManager struct {
	templates map[string]*ConfigTemplate
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	tm := &TemplateManager{
		templates: make(map[string]*ConfigTemplate),
	}
	tm.loadBuiltinTemplates()
	return tm
}

// loadBuiltinTemplates loads built-in configuration templates
func (tm *TemplateManager) loadBuiltinTemplates() {
	// Local development template
	tm.templates["local-dev"] = &ConfigTemplate{
		Name:        "local-dev",
		Description: "Local development with SLURM tokens",
		Category:    "development",
		Template: `# Local Development Configuration
currentContext: local
refreshRate: 10s
maxRetries: 3

contexts:
  - name: local
    cluster:
      endpoint: http://localhost:6820
      apiVersion: v0.0.43
      timeout: 30s

# Development settings
useMockClient: false
`,
		Variables: map[string]string{
			"CLUSTER_NAME": "local-cluster",
			"HOST":         "localhost",
			"PORT":         "6820",
			"USER":         os.Getenv("USER"),
		},
	}

	// Enterprise OAuth2 template
	tm.templates["enterprise-oauth2"] = &ConfigTemplate{
		Name:        "enterprise-oauth2",
		Description: "Enterprise setup with OAuth2 and DNS discovery",
		Category:    "enterprise",
		Template: `# Enterprise OAuth2 Configuration
currentContext: production
refreshRate: 30s
maxRetries: 5

contexts:
  - name: production
    cluster:
      endpoint: https://{{.CLUSTER_HOST}}:{{.CLUSTER_PORT}}
      apiVersion: v0.0.43
      timeout: 60s
      insecure: false

# Enterprise settings
useMockClient: false
`,
		Variables: map[string]string{
			"CLUSTER_NAME": "production",
			"CLUSTER_HOST": "slurm.company.com",
			"CLUSTER_PORT": "6820",
			"CA_FILE_PATH": "/etc/ssl/certs/company-ca.pem",
		},
	}

	// Multi-cluster template
	tm.templates["multi-cluster"] = &ConfigTemplate{
		Name:        "multi-cluster",
		Description: "Multiple cluster configuration",
		Category:    "production",
		Template: `# Multi-Cluster Configuration
currentContext: {{.DEFAULT_CLUSTER}}
refreshRate: 30s
maxRetries: 3

contexts:
  - name: production
    cluster:
      endpoint: https://{{.PROD_HOST}}:6820
      apiVersion: v0.0.43
      timeout: 60s
      insecure: false

  - name: development
    cluster:
      endpoint: https://{{.DEV_HOST}}:6820
      apiVersion: v0.0.43
      timeout: 30s
      insecure: true

  - name: testing
    cluster:
      endpoint: https://{{.TEST_HOST}}:6820
      apiVersion: v0.0.43
      timeout: 45s
      insecure: true
`,
		Variables: map[string]string{
			"DEFAULT_CLUSTER": "production",
			"PROD_HOST":       "slurm-prod.company.com",
			"DEV_HOST":        "slurm-dev.company.com", 
			"TEST_HOST":       "slurm-test.company.com",
		},
	}

	// HPC Center template
	tm.templates["hpc-center"] = &ConfigTemplate{
		Name:        "hpc-center",
		Description: "Large HPC center with multiple partitions",
		Category:    "hpc",
		Template: `# HPC Center Configuration
currentContext: {{.CLUSTER_NAME}}
refreshRate: 15s
maxRetries: 5

contexts:
  - name: {{.CLUSTER_NAME}}
    cluster:
      endpoint: https://{{.CONTROLLER_HOST}}:6820
      apiVersion: v0.0.43
      timeout: 120s
      insecure: false

# HPC optimizations
useMockClient: false
`,
		Variables: map[string]string{
			"CLUSTER_NAME":     "hpc-cluster",
			"CONTROLLER_HOST":  "slurmctl.hpc.edu",
			"SLURM_CONF_PATH":  "/etc/slurm/slurm.conf",
			"CA_PATH":          "/etc/ssl/certs/hpc-ca.pem",
			"CERT_PATH":        "/etc/ssl/certs/client.pem",
			"KEY_PATH":         "/etc/ssl/private/client-key.pem",
		},
	}

	// Cloud template
	tm.templates["cloud"] = &ConfigTemplate{
		Name:        "cloud",
		Description: "Cloud-based SLURM with auto-scaling",
		Category:    "cloud",
		Template: `# Cloud SLURM Configuration
currentContext: cloud
refreshRate: 60s
maxRetries: 3

contexts:
  - name: cloud
    cluster:
      endpoint: https://{{.CLOUD_HOST}}:6820
      apiVersion: v0.0.43
      timeout: 90s
      insecure: false

# Cloud-optimized settings
useMockClient: false
`,
		Variables: map[string]string{
			"CLOUD_CLUSTER_NAME": "cloud-cluster",
			"CLOUD_HOST":         "slurm.cloud.example.com",
		},
	}

	// Docker/Container template
	tm.templates["docker"] = &ConfigTemplate{
		Name:        "docker",
		Description: "Containerized SLURM setup",
		Category:    "development",
		Template: `# Docker SLURM Configuration
currentContext: docker
refreshRate: 5s
maxRetries: 2

contexts:
  - name: docker
    cluster:
      endpoint: http://{{.DOCKER_HOST}}:{{.DOCKER_PORT}}
      apiVersion: v0.0.43
      timeout: 15s
      insecure: true

# Container settings
useMockClient: false
`,
		Variables: map[string]string{
			"DOCKER_HOST": "localhost",
			"DOCKER_PORT": "8080",
		},
	}

	// Minimal template
	tm.templates["minimal"] = &ConfigTemplate{
		Name:        "minimal",
		Description: "Minimal configuration for quick setup",
		Category:    "basic",
		Template: `# Minimal s9s Configuration
currentContext: default
refreshRate: 30s
maxRetries: 3

contexts:
  - name: default
    cluster:
      endpoint: https://{{.HOST}}:6820
      apiVersion: v0.0.43
      timeout: 30s

useMockClient: false
`,
		Variables: map[string]string{
			"HOST": "localhost",
		},
	}
}

// GetTemplate returns a template by name
func (tm *TemplateManager) GetTemplate(name string) (*ConfigTemplate, bool) {
	template, exists := tm.templates[name]
	return template, exists
}

// ListTemplates returns all available templates
func (tm *TemplateManager) ListTemplates() []*ConfigTemplate {
	var templates []*ConfigTemplate
	for _, template := range tm.templates {
		templates = append(templates, template)
	}
	return templates
}

// ListTemplatesByCategory returns templates filtered by category
func (tm *TemplateManager) ListTemplatesByCategory(category string) []*ConfigTemplate {
	var templates []*ConfigTemplate
	for _, template := range tm.templates {
		if template.Category == category {
			templates = append(templates, template)
		}
	}
	return templates
}

// GenerateConfig generates a configuration from a template
func (tm *TemplateManager) GenerateConfig(templateName string, variables map[string]string) (string, error) {
	template, exists := tm.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template '%s' not found", templateName)
	}

	// Merge template variables with provided variables
	allVars := make(map[string]string)
	for k, v := range template.Variables {
		allVars[k] = v
	}
	for k, v := range variables {
		allVars[k] = v
	}

	// Simple template substitution (in production would use text/template)
	config := template.Template
	for key, value := range allVars {
		placeholder := "{{." + key + "}}"
		config = strings.ReplaceAll(config, placeholder, value)
	}

	return config, nil
}

// SaveTemplateAsConfig saves a generated template as a configuration file
func (tm *TemplateManager) SaveTemplateAsConfig(templateName string, variables map[string]string, configPath string) error {
	config, err := tm.GenerateConfig(templateName, variables)
	if err != nil {
		return fmt.Errorf("failed to generate config from template: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCategories returns all available template categories
func (tm *TemplateManager) GetCategories() []string {
	categories := make(map[string]bool)
	for _, template := range tm.templates {
		categories[template.Category] = true
	}

	var result []string
	for category := range categories {
		result = append(result, category)
	}
	return result
}

// CreateQuickStartConfig creates a quick-start configuration
func (tm *TemplateManager) CreateQuickStartConfig(clusterHost string, clusterName string) (*Config, error) {
	if clusterName == "" {
		clusterName = "quickstart-cluster"
	}
	if clusterHost == "" {
		clusterHost = "localhost"
	}

	config := &Config{
		CurrentContext: "quickstart",
		RefreshRate:    "30s",
		UseMockClient:  false,
		MaxRetries:     3,
		Contexts: []ContextConfig{
			{
				Name: "quickstart",
				Cluster: ClusterConfig{
					Endpoint:   fmt.Sprintf("https://%s:6820", clusterHost),
					APIVersion: "v0.0.43",
					Timeout:    "30s",
				},
			},
		},
	}

	return config, nil
}

// ValidateTemplate validates a template for correctness
func (tm *TemplateManager) ValidateTemplate(template *ConfigTemplate) []string {
	var errors []string

	if template.Name == "" {
		errors = append(errors, "template name is required")
	}

	if template.Description == "" {
		errors = append(errors, "template description is required")
	}

	if template.Category == "" {
		errors = append(errors, "template category is required")
	}

	if template.Template == "" {
		errors = append(errors, "template content is required")
	}

	// Validate template syntax (basic check)
	if !strings.Contains(template.Template, "version:") {
		errors = append(errors, "template must contain version field")
	}

	if !strings.Contains(template.Template, "contexts:") {
		errors = append(errors, "template must contain contexts field")
	}

	return errors
}

// GetTemplatePreview generates a preview of a template with default variables
func (tm *TemplateManager) GetTemplatePreview(templateName string) (string, error) {
	template, exists := tm.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template '%s' not found", templateName)
	}

	return tm.GenerateConfig(templateName, template.Variables)
}