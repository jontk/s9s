// Package config provides configuration loading, validation, and management.
package config

import (
	"errors"
	"fmt"
	"os"
	osuser "os/user"
	"path/filepath"
	"strings"

	"github.com/jontk/s9s/internal/fileperms"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	RefreshRate    string            `mapstructure:"refreshRate" yaml:"refreshRate"`
	MaxRetries     int               `mapstructure:"maxRetries" yaml:"maxRetries"`
	DefaultCluster string            `mapstructure:"defaultCluster" yaml:"defaultCluster"`
	Clusters       []ClusterContext  `mapstructure:"clusters" yaml:"clusters"`
	UI             UIConfig          `mapstructure:"ui" yaml:"ui"`
	Views          ViewsConfig       `mapstructure:"views" yaml:"views"`
	Features       FeaturesConfig    `mapstructure:"features" yaml:"features"`
	Shortcuts      []ShortcutConfig  `mapstructure:"shortcuts" yaml:"shortcuts"`
	Aliases        map[string]string `mapstructure:"aliases" yaml:"aliases"`
	Plugins        []PluginConfig    `mapstructure:"plugins" yaml:"plugins"`
	UseMockClient  bool              `mapstructure:"useMockClient" yaml:"useMockClient"`
	PluginSettings PluginSettings    `mapstructure:"pluginSettings" yaml:"pluginSettings"`
	Discovery      DiscoveryConfig   `mapstructure:"discovery" yaml:"discovery"`
	Update         UpdateConfig      `mapstructure:"update" yaml:"update"`

	// Computed fields
	Cluster    ClusterConfig `mapstructure:"-" yaml:"-"`
	ConfigPath string        `mapstructure:"-" yaml:"-"` // Path to the config file that was loaded
}

// DiscoveryConfig holds settings for auto-discovery of slurmrestd endpoint and token
type DiscoveryConfig struct {
	Enabled        bool   `mapstructure:"enabled" yaml:"enabled"`
	EnableEndpoint bool   `mapstructure:"enableEndpoint" yaml:"enableEndpoint"`
	EnableToken    bool   `mapstructure:"enableToken" yaml:"enableToken"`
	Timeout        string `mapstructure:"timeout" yaml:"timeout"`
	DefaultPort    int    `mapstructure:"defaultPort" yaml:"defaultPort"`
	ScontrolPath   string `mapstructure:"scontrolPath" yaml:"scontrolPath"`
}

// ClusterContext represents a cluster context
type ClusterContext struct {
	Name      string        `mapstructure:"name" yaml:"name"`
	Cluster   ClusterConfig `mapstructure:"cluster" yaml:"cluster"`
	Namespace string        `mapstructure:"namespace" yaml:"namespace"`
	ReadOnly  bool          `mapstructure:"readOnly" yaml:"readOnly"`
}

// ClusterConfig holds SLURM cluster connection details
type ClusterConfig struct {
	Endpoint   string `mapstructure:"endpoint" yaml:"endpoint"`
	Token      string `mapstructure:"token" yaml:"token"`
	APIVersion string `mapstructure:"apiVersion" yaml:"apiVersion"`
	Insecure   bool   `mapstructure:"insecure" yaml:"insecure"`
	Timeout    string `mapstructure:"timeout" yaml:"timeout"`
	User       string `mapstructure:"user" yaml:"user"` // Override X-SLURM-USER-NAME header
}

// UIConfig holds UI-related settings
type UIConfig struct {
	Skin        string `mapstructure:"skin" yaml:"skin"`
	Logoless    bool   `mapstructure:"logoless" yaml:"logoless"`
	Crumbsless  bool   `mapstructure:"crumbsless" yaml:"crumbsless"`
	Statusless  bool   `mapstructure:"statusless" yaml:"statusless"`
	Headless    bool   `mapstructure:"headless" yaml:"headless"`
	NoIcons     bool   `mapstructure:"noIcons" yaml:"noIcons"`
	EnableMouse bool   `mapstructure:"enableMouse" yaml:"enableMouse"`
}

// ViewsConfig holds view-specific settings
type ViewsConfig struct {
	Jobs       JobsViewConfig       `mapstructure:"jobs" yaml:"jobs"`
	Nodes      NodesViewConfig      `mapstructure:"nodes" yaml:"nodes"`
	Partitions PartitionsViewConfig `mapstructure:"partitions" yaml:"partitions"`
}

// JobsViewConfig holds jobs view settings
type JobsViewConfig struct {
	Columns        []string            `mapstructure:"columns" yaml:"columns"`
	ShowOnlyActive bool                `mapstructure:"showOnlyActive" yaml:"showOnlyActive"`
	DefaultSort    string              `mapstructure:"defaultSort" yaml:"defaultSort"`
	MaxJobs        int                 `mapstructure:"maxJobs" yaml:"maxJobs"`
	Submission     JobSubmissionConfig `mapstructure:"submission" yaml:"submission"`
}

// JobSubmissionConfig holds job submission form settings and templates
type JobSubmissionConfig struct {
	FormDefaults         map[string]any      `mapstructure:"formDefaults" yaml:"formDefaults"`
	HiddenFields         []string            `mapstructure:"hiddenFields" yaml:"hiddenFields"`
	FieldOptions         map[string][]string `mapstructure:"fieldOptions" yaml:"fieldOptions"`
	ShowBuiltinTemplates *bool               `mapstructure:"showBuiltinTemplates" yaml:"showBuiltinTemplates"`
	TemplateSources      []string            `mapstructure:"templateSources" yaml:"templateSources"`
	Templates            []JobTemplateConfig `mapstructure:"templates" yaml:"templates"`
}

// JobTemplateConfig represents a user-defined job submission template
type JobTemplateConfig struct {
	Name         string         `mapstructure:"name" yaml:"name"`
	Description  string         `mapstructure:"description" yaml:"description"`
	Defaults     map[string]any `mapstructure:"defaults" yaml:"defaults"`
	HiddenFields []string       `mapstructure:"hiddenFields" yaml:"hiddenFields"`
}

// NodesViewConfig holds nodes view settings
type NodesViewConfig struct {
	GroupBy         string `mapstructure:"groupBy" yaml:"groupBy"`
	ShowUtilization bool   `mapstructure:"showUtilization" yaml:"showUtilization"`
	MaxNodes        int    `mapstructure:"maxNodes" yaml:"maxNodes"`
}

// PartitionsViewConfig holds partitions view settings
type PartitionsViewConfig struct {
	ShowQueueDepth bool `mapstructure:"showQueueDepth" yaml:"showQueueDepth"`
	ShowWaitTime   bool `mapstructure:"showWaitTime" yaml:"showWaitTime"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	Streaming      bool `mapstructure:"streaming" yaml:"streaming"`
	Pulseye        bool `mapstructure:"pulseye" yaml:"pulseye"`
	Xray           bool `mapstructure:"xray" yaml:"xray"`
	AppDiagnostics bool `mapstructure:"appDiagnostics" yaml:"appDiagnostics"`
}

// ShortcutConfig represents a custom keyboard shortcut
type ShortcutConfig struct {
	Key         string `mapstructure:"key" yaml:"key"`
	Action      string `mapstructure:"action" yaml:"action"`
	Description string `mapstructure:"description" yaml:"description"`
}

// PluginConfig represents a plugin configuration
type PluginConfig struct {
	Name    string         `mapstructure:"name" yaml:"name"`
	Enabled bool           `mapstructure:"enabled" yaml:"enabled"`
	Path    string         `mapstructure:"path" yaml:"path"`
	Config  map[string]any `mapstructure:"config" yaml:"config"`
}

// UpdateConfig holds auto-update check settings
type UpdateConfig struct {
	Enabled       bool   `mapstructure:"enabled" yaml:"enabled"`
	AutoInstall   bool   `mapstructure:"autoInstall" yaml:"autoInstall"`
	CheckInterval string `mapstructure:"checkInterval" yaml:"checkInterval"`
	PreRelease    bool   `mapstructure:"preRelease" yaml:"preRelease"`
}

// PluginSettings contains global plugin settings
type PluginSettings struct {
	EnableAll     bool    `mapstructure:"enableAll" yaml:"enableAll"`
	PluginDir     string  `mapstructure:"pluginDir" yaml:"pluginDir"`
	AutoDiscover  bool    `mapstructure:"autoDiscover" yaml:"autoDiscover"`
	SafeMode      bool    `mapstructure:"safeMode" yaml:"safeMode"`           // Disable external plugins
	MaxMemoryMB   int     `mapstructure:"maxMemoryMB" yaml:"maxMemoryMB"`     // Memory limit per plugin
	MaxCPUPercent float64 `mapstructure:"maxCPUPercent" yaml:"maxCPUPercent"` // CPU limit per plugin
}

// DefaultConfig returns a configuration with sensible defaults
// NOTE: These values must match setDefaults() to ensure consistent behavior
func DefaultConfig() *Config {
	return &Config{
		RefreshRate:    "2s", // Aligned with setDefaults
		MaxRetries:     3,
		DefaultCluster: "default",
		Clusters:       []ClusterContext{},
		UI: UIConfig{
			Skin:        "default",
			Logoless:    false,
			Crumbsless:  false,
			Statusless:  false,
			Headless:    false,
			NoIcons:     false,
			EnableMouse: true, // Aligned with setDefaults
		},
		Views: ViewsConfig{
			Jobs: JobsViewConfig{
				Columns:        []string{"id", "name", "user", "state", "time", "nodes", "priority"}, // Aligned with setDefaults
				ShowOnlyActive: true,                                                                 // Aligned with setDefaults
				DefaultSort:    "time",                                                               // Aligned with setDefaults
				MaxJobs:        1000,                                                                 // Aligned with setDefaults
			},
			Nodes: NodesViewConfig{
				GroupBy:         "partition", // Aligned with setDefaults
				ShowUtilization: true,
				MaxNodes:        500, // Aligned with setDefaults
			},
			Partitions: PartitionsViewConfig{
				ShowQueueDepth: true,
				ShowWaitTime:   true,
			},
		},
		Features: FeaturesConfig{
			Streaming: true, // Aligned with setDefaults
			Pulseye:   true, // Aligned with setDefaults
			Xray:      false,
		},
		Shortcuts: []ShortcutConfig{},
		Aliases: map[string]string{ // Aligned with setDefaults
			"ctx": "context",
			"kj":  "kill job",
			"dj":  "describe job",
			"dn":  "describe node",
			"sub": "submit job",
		},
		Plugins:       []PluginConfig{},
		UseMockClient: false,
		PluginSettings: PluginSettings{
			EnableAll:     false,
			PluginDir:     "$HOME/.s9s/plugins", // Aligned with setDefaults
			AutoDiscover:  true,
			SafeMode:      false,
			MaxMemoryMB:   100,  // Aligned with setDefaults
			MaxCPUPercent: 25.0, // Aligned with setDefaults
		},
		Update: UpdateConfig{
			Enabled:       true,
			AutoInstall:   false,
			CheckInterval: "24h",
			PreRelease:    false,
		},
		Discovery: DiscoveryConfig{
			Enabled:        true,  // Aligned with setDefaults
			EnableEndpoint: true,  // Aligned with setDefaults
			EnableToken:    true,  // Aligned with setDefaults
			Timeout:        "10s", // Aligned with setDefaults
			DefaultPort:    6820,
			ScontrolPath:   "scontrol", // Aligned with setDefaults (binary name, not full path)
		},
	}
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	return LoadWithPath("")
}

// LoadWithPath reads configuration from a specific file path
func LoadWithPath(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file details
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Config search paths
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.s9s")
		v.AddConfigPath("/etc/s9s")
	}

	// Environment variable support
	v.SetEnvPrefix("S9S")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Read config file if exists
	if err := v.ReadInConfig(); err != nil {
		// If config file not found, use defaults and environment
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	// Create config struct
	cfg := &Config{}

	// Unmarshal configuration
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Override with environment variables
	applyEnvironmentOverrides(cfg)

	// Set the current cluster based on context
	if err := cfg.SetCurrentCluster(); err != nil {
		return nil, err
	}

	// Record the config file path that was used
	if configPath != "" {
		cfg.ConfigPath = configPath
	} else if usedFile := v.ConfigFileUsed(); usedFile != "" {
		cfg.ConfigPath = usedFile
	} else {
		cfg.ConfigPath = filepath.Join(os.Getenv("HOME"), ".s9s", "config.yaml")
	}

	// Ensure config directory exists
	configDir := filepath.Join(os.Getenv("HOME"), ".s9s")
	_ = os.MkdirAll(configDir, fileperms.ConfigDir)

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// General defaults
	v.SetDefault("refreshRate", "2s")
	v.SetDefault("maxRetries", 3)
	v.SetDefault("defaultCluster", "default")
	v.SetDefault("useMockClient", false)

	// UI defaults
	v.SetDefault("ui.skin", "default")
	v.SetDefault("ui.enableMouse", true)
	v.SetDefault("ui.logoless", false)
	v.SetDefault("ui.crumbsless", false)
	v.SetDefault("ui.statusless", false)
	v.SetDefault("ui.headless", false)
	v.SetDefault("ui.noIcons", false)

	// Views defaults
	v.SetDefault("views.jobs.columns", []string{"id", "name", "user", "state", "time", "nodes", "priority"})
	v.SetDefault("views.jobs.showOnlyActive", true)
	v.SetDefault("views.jobs.defaultSort", "time")
	v.SetDefault("views.jobs.maxJobs", 1000)

	v.SetDefault("views.jobs.submission.templateSources", []string{"builtin", "config", "saved"})

	v.SetDefault("views.nodes.groupBy", "partition")
	v.SetDefault("views.nodes.showUtilization", true)
	v.SetDefault("views.nodes.maxNodes", 500)

	v.SetDefault("views.partitions.showQueueDepth", true)
	v.SetDefault("views.partitions.showWaitTime", true)

	// Features defaults
	v.SetDefault("features.streaming", true)
	v.SetDefault("features.pulseye", true)
	v.SetDefault("features.xray", false)

	// Plugin settings defaults
	v.SetDefault("pluginSettings.enableAll", false)
	v.SetDefault("pluginSettings.pluginDir", "$HOME/.s9s/plugins")
	v.SetDefault("pluginSettings.autoDiscover", true)
	v.SetDefault("pluginSettings.safeMode", false)
	v.SetDefault("pluginSettings.maxMemoryMB", 100)
	v.SetDefault("pluginSettings.maxCPUPercent", 25.0)

	// Default aliases
	v.SetDefault("aliases", map[string]string{
		"ctx": "context",
		"kj":  "kill job",
		"dj":  "describe job",
		"dn":  "describe node",
		"sub": "submit job",
	})

	// Update defaults
	v.SetDefault("update.enabled", true)
	v.SetDefault("update.autoInstall", false)
	v.SetDefault("update.checkInterval", "24h")
	v.SetDefault("update.preRelease", false)

	// Discovery defaults
	v.SetDefault("discovery.enabled", true)
	v.SetDefault("discovery.enableEndpoint", true)
	v.SetDefault("discovery.enableToken", true)
	v.SetDefault("discovery.timeout", "10s")
	v.SetDefault("discovery.defaultPort", 6820)
	v.SetDefault("discovery.scontrolPath", "scontrol")
}

// applyEnvironmentOverrides applies environment variable overrides
// Environment variables always take precedence over config file settings
func applyEnvironmentOverrides(cfg *Config) {
	// Check for cluster endpoint override (support both S9S_ prefixed and unprefixed)
	endpoint := os.Getenv("S9S_SLURM_REST_URL")
	if endpoint == "" {
		endpoint = os.Getenv("SLURM_REST_URL")
	}

	token := os.Getenv("S9S_SLURM_JWT")
	if token == "" {
		token = os.Getenv("SLURM_JWT")
	}

	apiVersion := os.Getenv("SLURM_API_VERSION")
	if apiVersion == "" {
		apiVersion = "v0.0.43"
	}

	// If endpoint is set in environment, it overrides everything
	if endpoint != "" {
		// Create or update default context with environment values
		defaultEntry := ClusterContext{
			Name: "default",
			Cluster: ClusterConfig{
				Endpoint:   endpoint,
				Token:      token,
				APIVersion: apiVersion,
			},
		}

		// Check if default cluster already exists
		found := false
		for i, cl := range cfg.Clusters {
			if cl.Name == "default" {
				cfg.Clusters[i] = defaultEntry
				found = true
				break
			}
		}

		// If not found, add it
		if !found {
			cfg.Clusters = append(cfg.Clusters, defaultEntry)
		}

		cfg.DefaultCluster = "default"
	}
}

// SetCurrentCluster sets the current cluster configuration based on the default cluster
func (c *Config) SetCurrentCluster() error {
	if c.DefaultCluster == "" {
		c.DefaultCluster = "default"
	}

	// Find the matching cluster entry
	for _, cl := range c.Clusters {
		if cl.Name == c.DefaultCluster {
			c.Cluster = cl.Cluster
			return nil
		}
	}

	// If no context found but we have environment variables, use them
	// Support both S9S_ prefixed and unprefixed environment variables
	endpoint := os.Getenv("S9S_SLURM_REST_URL")
	if endpoint == "" {
		endpoint = os.Getenv("SLURM_REST_URL")
	}

	token := os.Getenv("S9S_SLURM_JWT")
	if token == "" {
		token = os.Getenv("SLURM_JWT")
	}

	if endpoint != "" {
		c.Cluster = ClusterConfig{
			Endpoint:   endpoint,
			Token:      token,
			APIVersion: getEnvOrDefault("SLURM_API_VERSION", "v0.0.43"),
			Timeout:    getEnvOrDefault("SLURM_TIMEOUT", "30s"),
		}
		return nil
	}

	// If no clusters exist and discovery is enabled, allow startup with empty cluster
	// Auto-discovery will populate the cluster configuration
	if len(c.Clusters) == 0 && c.Discovery.Enabled {
		c.Cluster = ClusterConfig{
			// Leave APIVersion empty to enable auto-detection from slurmrestd
			Timeout: "30s",
		}
		return nil
	}

	return fmt.Errorf("cluster %q not found", c.DefaultCluster)
}

// GetCluster returns the cluster context by name
func (c *Config) GetCluster(name string) (*ClusterContext, error) {
	for _, cl := range c.Clusters {
		if cl.Name == name {
			return &cl, nil
		}
	}
	return nil, fmt.Errorf("cluster %q not found", name)
}

// ResolveSlurmUser returns the effective SLURM username for API requests.
// Priority: SLURM_USER_NAME env > cluster config user > USER env > OS current user.
// Returns empty string if no username can be determined.
func (c *Config) ResolveSlurmUser() string {
	return ResolveSlurmUserForCluster(&c.Cluster)
}

// ResolveSlurmUserForCluster resolves the SLURM username from a cluster config.
// Priority: SLURM_USER_NAME env > cluster config user > USER env > OS current user.
func ResolveSlurmUserForCluster(cfg *ClusterConfig) string {
	if u := os.Getenv("SLURM_USER_NAME"); u != "" {
		return u
	}
	if cfg != nil && cfg.User != "" {
		return cfg.User
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u, err := osuser.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return ""
}

// SaveToFile saves the configuration to a file
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ValidateMockUsage validates if mock client usage is allowed
func (c *Config) ValidateMockUsage() error {
	if !c.UseMockClient {
		return nil // No mock usage, nothing to validate
	}

	// Import mock validator dynamically to avoid circular imports
	// This validation is done in CLI layer instead
	return nil
}
