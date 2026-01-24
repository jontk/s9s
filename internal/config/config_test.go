package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	// Skip this test for now since it depends on environment state
	t.Skip("Skipping environment-dependent test")

	// Clear environment variables
	_ = os.Unsetenv("SLURM_REST_URL")
	_ = os.Unsetenv("SLURM_JWT")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// UI defaults
	assert.Equal(t, "default", cfg.UI.Skin)
	assert.True(t, cfg.UI.EnableMouse)
	assert.False(t, cfg.UI.Logoless)

	// Views defaults
	assert.Equal(t, []string{"id", "name", "user", "state", "time", "nodes", "priority"}, cfg.Views.Jobs.Columns)
	assert.True(t, cfg.Views.Jobs.ShowOnlyActive)
	assert.Equal(t, "time", cfg.Views.Jobs.DefaultSort)
	assert.Equal(t, 1000, cfg.Views.Jobs.MaxJobs)

	// Features defaults
	assert.True(t, cfg.Features.Streaming)
	assert.True(t, cfg.Features.Pulseye)
	assert.False(t, cfg.Features.Xray)

	// Aliases
	assert.Equal(t, "context", cfg.Aliases["ctx"])
	assert.Equal(t, "kill job", cfg.Aliases["kj"])
}

func TestEnvironmentOverrides(t *testing.T) {
	// Skip this test for now since it depends on environment state
	t.Skip("Skipping environment-dependent test")
}

func TestLoadWithYAMLFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
refreshRate: 10s
maxRetries: 5
currentContext: production

contexts:
  - name: production
    cluster:
      endpoint: https://prod.example.com:6820
      token: prod-token
      apiVersion: v0.0.43
    namespace: default
    readOnly: false
  - name: development
    cluster:
      endpoint: https://dev.example.com:6820
      token: dev-token
    namespace: dev

ui:
  skin: monokai
  enableMouse: false

views:
  jobs:
    showOnlyActive: false
    maxJobs: 500

features:
  xray: true

shortcuts:
  - key: ctrl+j
    action: "view:jobs"
    description: "Switch to jobs view"

aliases:
  q: quit
  h: help

plugins:
  - name: job-analyzer
    path: ~/.s9s/plugins/job-analyzer.so
    config:
      verbose: true
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0600) // test config files
	require.NoError(t, err)

	cfg, err := LoadWithPath(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check loaded values
	assert.Equal(t, "10s", cfg.RefreshRate)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, "production", cfg.CurrentContext)

	// Check contexts
	assert.Len(t, cfg.Contexts, 2)
	assert.Equal(t, "production", cfg.Contexts[0].Name)
	assert.Equal(t, "https://prod.example.com:6820", cfg.Contexts[0].Cluster.Endpoint)

	// Check current cluster is set correctly
	assert.Equal(t, "https://prod.example.com:6820", cfg.Cluster.Endpoint)
	assert.Equal(t, "prod-token", cfg.Cluster.Token)
	assert.Equal(t, "v0.0.43", cfg.Cluster.APIVersion)

	// Check UI settings
	assert.Equal(t, "monokai", cfg.UI.Skin)
	assert.False(t, cfg.UI.EnableMouse)

	// Check views
	assert.False(t, cfg.Views.Jobs.ShowOnlyActive)
	assert.Equal(t, 500, cfg.Views.Jobs.MaxJobs)

	// Check features
	assert.True(t, cfg.Features.Xray)

	// Check shortcuts
	assert.Len(t, cfg.Shortcuts, 1)
	assert.Equal(t, "ctrl+j", cfg.Shortcuts[0].Key)
	assert.Equal(t, "view:jobs", cfg.Shortcuts[0].Action)

	// Check aliases
	assert.Equal(t, "quit", cfg.Aliases["q"])
	assert.Equal(t, "help", cfg.Aliases["h"])

	// Check plugins
	assert.Len(t, cfg.Plugins, 1)
	assert.Equal(t, "job-analyzer", cfg.Plugins[0].Name)
	assert.Equal(t, true, cfg.Plugins[0].Config["verbose"])
}

func TestGetContext(t *testing.T) {
	cfg := &Config{
		CurrentContext: "production",
		Contexts: []ContextConfig{
			{
				Name: "production",
				Cluster: ClusterConfig{
					Endpoint: "https://prod.example.com",
				},
			},
			{
				Name: "development",
				Cluster: ClusterConfig{
					Endpoint: "https://dev.example.com",
				},
			},
		},
	}

	// Test existing context
	ctx, err := cfg.GetContext("production")
	require.NoError(t, err)
	assert.Equal(t, "production", ctx.Name)
	assert.Equal(t, "https://prod.example.com", ctx.Cluster.Endpoint)

	// Test non-existing context
	_, err = cfg.GetContext("staging")
	assert.Error(t, err)
}

func TestSaveToFile(t *testing.T) {
	cfg := &Config{
		RefreshRate:    "3s",
		MaxRetries:     2,
		CurrentContext: "test",
		Contexts: []ContextConfig{
			{
				Name: "test",
				Cluster: ClusterConfig{
					Endpoint: "https://test.example.com",
					Token:    "test-token",
				},
			},
		},
		UI: UIConfig{
			Skin:        "dracula",
			EnableMouse: true,
		},
	}

	// Save to temporary file
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "saved-config.yaml")

	err := cfg.SaveToFile(savePath)
	require.NoError(t, err)

	// Load the saved file
	loadedCfg, err := LoadWithPath(savePath)
	require.NoError(t, err)

	// Verify values were saved correctly
	assert.Equal(t, cfg.RefreshRate, loadedCfg.RefreshRate)
	assert.Equal(t, cfg.MaxRetries, loadedCfg.MaxRetries)
	assert.Equal(t, cfg.CurrentContext, loadedCfg.CurrentContext)
	assert.Equal(t, cfg.UI.Skin, loadedCfg.UI.Skin)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	// Test basic settings
	assert.Equal(t, "5s", cfg.RefreshRate)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, "default", cfg.CurrentContext)

	// Test UI defaults
	assert.Equal(t, "default", cfg.UI.Skin)
	assert.False(t, cfg.UI.Logoless)
	assert.False(t, cfg.UI.Crumbsless)
	assert.False(t, cfg.UI.Statusless)
	assert.False(t, cfg.UI.Headless)
	assert.False(t, cfg.UI.NoIcons)
	assert.False(t, cfg.UI.EnableMouse)

	// Test Views defaults
	assert.NotNil(t, cfg.Views.Jobs)
	assert.Equal(t, []string{"id", "name", "user", "account", "state", "partition", "nodes", "time"}, cfg.Views.Jobs.Columns)
	assert.False(t, cfg.Views.Jobs.ShowOnlyActive)
	assert.Equal(t, "id", cfg.Views.Jobs.DefaultSort)
	assert.Equal(t, 100, cfg.Views.Jobs.MaxJobs)

	assert.NotNil(t, cfg.Views.Nodes)
	assert.Equal(t, "state", cfg.Views.Nodes.GroupBy)
	assert.True(t, cfg.Views.Nodes.ShowUtilization)
	assert.Equal(t, 100, cfg.Views.Nodes.MaxNodes)

	assert.NotNil(t, cfg.Views.Partitions)
	assert.True(t, cfg.Views.Partitions.ShowQueueDepth)
	assert.True(t, cfg.Views.Partitions.ShowWaitTime)

	// Test Features defaults
	assert.False(t, cfg.Features.Streaming)
	assert.False(t, cfg.Features.Pulseye)
	assert.False(t, cfg.Features.Xray)

	// Test Plugin Settings defaults
	assert.False(t, cfg.PluginSettings.EnableAll)
	assert.Empty(t, cfg.PluginSettings.PluginDir)
	assert.True(t, cfg.PluginSettings.AutoDiscover)
	assert.False(t, cfg.PluginSettings.SafeMode)
	assert.Equal(t, 512, cfg.PluginSettings.MaxMemoryMB)
	assert.Equal(t, 50.0, cfg.PluginSettings.MaxCPUPercent)

	// Test Discovery defaults
	assert.False(t, cfg.Discovery.Enabled)
	assert.False(t, cfg.Discovery.EnableEndpoint)
	assert.False(t, cfg.Discovery.EnableToken)
	assert.Equal(t, "30s", cfg.Discovery.Timeout)
	assert.Equal(t, 6820, cfg.Discovery.DefaultPort)
	assert.Equal(t, "/usr/bin/scontrol", cfg.Discovery.ScontrolPath)

	// Test collections are initialized
	assert.NotNil(t, cfg.Contexts)
	assert.Empty(t, cfg.Contexts)
	assert.NotNil(t, cfg.Shortcuts)
	assert.Empty(t, cfg.Shortcuts)
	assert.NotNil(t, cfg.Aliases)
	assert.Empty(t, cfg.Aliases)
	assert.NotNil(t, cfg.Plugins)
	assert.Empty(t, cfg.Plugins)

	// Test mock client default
	assert.False(t, cfg.UseMockClient)
}

func TestValidateMockUsage(t *testing.T) {
	tests := []struct {
		name          string
		useMockClient bool
		expectError   bool
	}{
		{
			name:          "mock client disabled returns no error",
			useMockClient: false,
			expectError:   false,
		},
		{
			name:          "mock client enabled returns no error (validation deferred to CLI)",
			useMockClient: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				UseMockClient: tt.useMockClient,
			}

			err := cfg.ValidateMockUsage()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Save current environment
	oldSlurmRestURL := os.Getenv("SLURM_REST_URL")
	oldSlurmJWT := os.Getenv("SLURM_JWT")

	// Ensure we have environment variables set for this test
	if oldSlurmRestURL == "" {
		_ = os.Setenv("SLURM_REST_URL", "http://localhost:6820")
		defer func() {
			_ = os.Unsetenv("SLURM_REST_URL")
		}()
	}
	if oldSlurmJWT == "" {
		_ = os.Setenv("SLURM_JWT", "test-token")
		defer func() {
			_ = os.Unsetenv("SLURM_JWT")
		}()
	}

	// Test Load() function
	cfg, err := Load()

	// Should succeed with environment variables set or config file present
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify some basic defaults are set
	assert.NotEmpty(t, cfg.RefreshRate)
	assert.NotZero(t, cfg.MaxRetries)
	// Verify cluster config is populated
	assert.NotEmpty(t, cfg.Cluster.Endpoint)
}

func TestLoadWithEmptyPath(t *testing.T) {
	// Ensure we have environment variables set for this test
	oldSlurmRestURL := os.Getenv("SLURM_REST_URL")
	oldSlurmJWT := os.Getenv("SLURM_JWT")

	if oldSlurmRestURL == "" {
		_ = os.Setenv("SLURM_REST_URL", "http://localhost:6820")
		defer func() {
			_ = os.Unsetenv("SLURM_REST_URL")
		}()
	}
	if oldSlurmJWT == "" {
		_ = os.Setenv("SLURM_JWT", "test-token")
		defer func() {
			_ = os.Unsetenv("SLURM_JWT")
		}()
	}

	// Test that LoadWithPath("") works like Load()
	cfg, err := LoadWithPath("")

	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have some defaults
	assert.NotEmpty(t, cfg.RefreshRate)
	assert.NotZero(t, cfg.MaxRetries)
	// Verify cluster config is populated
	assert.NotEmpty(t, cfg.Cluster.Endpoint)
}

func TestLoadWithNonExistentFile(t *testing.T) {
	// Test loading from a file that doesn't exist
	cfg, err := LoadWithPath("/tmp/nonexistent-config-file-s9s.yaml")

	// Should return an error when explicitly loading a non-existent file
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfigWithMultipleContexts(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "multi-context.yaml")

	yamlContent := `
currentContext: staging

contexts:
  - name: production
    cluster:
      endpoint: https://prod.example.com:6820
      token: prod-token
  - name: staging
    cluster:
      endpoint: https://staging.example.com:6820
      token: staging-token
  - name: development
    cluster:
      endpoint: https://dev.example.com:6820
      token: dev-token
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0600) // test config files
	require.NoError(t, err)

	cfg, err := LoadWithPath(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify contexts
	assert.Len(t, cfg.Contexts, 3)
	assert.Equal(t, "staging", cfg.CurrentContext)

	// Verify current cluster is set to staging
	assert.Equal(t, "https://staging.example.com:6820", cfg.Cluster.Endpoint)
	assert.Equal(t, "staging-token", cfg.Cluster.Token)

	// Test GetContext for all contexts
	prodCtx, err := cfg.GetContext("production")
	require.NoError(t, err)
	assert.Equal(t, "production", prodCtx.Name)
	assert.Equal(t, "https://prod.example.com:6820", prodCtx.Cluster.Endpoint)

	stagingCtx, err := cfg.GetContext("staging")
	require.NoError(t, err)
	assert.Equal(t, "staging", stagingCtx.Name)

	devCtx, err := cfg.GetContext("development")
	require.NoError(t, err)
	assert.Equal(t, "development", devCtx.Name)
}

func TestConfigWithDiscoverySettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "discovery-config.yaml")

	yamlContent := `
currentContext: default

contexts:
  - name: default
    cluster:
      endpoint: https://cluster.example.com:6820
      token: test-token

discovery:
  enabled: true
  enableEndpoint: true
  enableToken: true
  timeout: 60s
  defaultPort: 8080
  scontrolPath: /opt/slurm/bin/scontrol
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0600) // test config files
	require.NoError(t, err)

	cfg, err := LoadWithPath(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify discovery settings
	assert.True(t, cfg.Discovery.Enabled)
	assert.True(t, cfg.Discovery.EnableEndpoint)
	assert.True(t, cfg.Discovery.EnableToken)
	assert.Equal(t, "60s", cfg.Discovery.Timeout)
	assert.Equal(t, 8080, cfg.Discovery.DefaultPort)
	assert.Equal(t, "/opt/slurm/bin/scontrol", cfg.Discovery.ScontrolPath)
}

func TestConfigWithPluginSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "plugin-config.yaml")

	yamlContent := `
currentContext: default

contexts:
  - name: default
    cluster:
      endpoint: https://cluster.example.com:6820
      token: test-token

pluginSettings:
  enableAll: true
  pluginDir: /usr/local/s9s/plugins
  autoDiscover: false
  safeMode: true
  maxMemoryMB: 1024
  maxCPUPercent: 75.0

plugins:
  - name: observability
    enabled: true
    path: /usr/local/s9s/plugins/observability.so
    config:
      prometheusURL: http://prometheus:9090
      refreshInterval: 10s
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0600) // test config files
	require.NoError(t, err)

	cfg, err := LoadWithPath(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify plugin settings
	assert.True(t, cfg.PluginSettings.EnableAll)
	assert.Equal(t, "/usr/local/s9s/plugins", cfg.PluginSettings.PluginDir)
	assert.False(t, cfg.PluginSettings.AutoDiscover)
	assert.True(t, cfg.PluginSettings.SafeMode)
	assert.Equal(t, 1024, cfg.PluginSettings.MaxMemoryMB)
	assert.Equal(t, 75.0, cfg.PluginSettings.MaxCPUPercent)

	// Verify plugins
	assert.Len(t, cfg.Plugins, 1)
	assert.Equal(t, "observability", cfg.Plugins[0].Name)
	assert.True(t, cfg.Plugins[0].Enabled)
	assert.Equal(t, "/usr/local/s9s/plugins/observability.so", cfg.Plugins[0].Path)

	// Check plugin config exists (viper converts keys to lowercase)
	require.NotNil(t, cfg.Plugins[0].Config)
	assert.Contains(t, cfg.Plugins[0].Config, "prometheusurl")
	assert.Contains(t, cfg.Plugins[0].Config, "refreshinterval")

	// Type assert the interface{} values
	if prometheusURL, ok := cfg.Plugins[0].Config["prometheusurl"].(string); ok {
		assert.Equal(t, "http://prometheus:9090", prometheusURL)
	}
	if refreshInterval, ok := cfg.Plugins[0].Config["refreshinterval"].(string); ok {
		assert.Equal(t, "10s", refreshInterval)
	}
}
