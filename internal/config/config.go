package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	RefreshRate    string            `mapstructure:"refreshRate"`
	MaxRetries     int               `mapstructure:"maxRetries"`
	CurrentContext string            `mapstructure:"currentContext"`
	Contexts       []ContextConfig   `mapstructure:"contexts"`
	UI             UIConfig          `mapstructure:"ui"`
	Views          ViewsConfig       `mapstructure:"views"`
	Features       FeaturesConfig    `mapstructure:"features"`
	Shortcuts      []ShortcutConfig  `mapstructure:"shortcuts"`
	Aliases        map[string]string `mapstructure:"aliases"`
	Plugins        []PluginConfig    `mapstructure:"plugins"`
	UseMockClient  bool              `mapstructure:"useMockClient"`
	PluginSettings PluginSettings    `mapstructure:"pluginSettings"`

	// Computed fields
	Cluster ClusterConfig `mapstructure:"-"`
}

// ContextConfig represents a cluster context
type ContextConfig struct {
	Name      string        `mapstructure:"name"`
	Cluster   ClusterConfig `mapstructure:"cluster"`
	Namespace string        `mapstructure:"namespace"`
	ReadOnly  bool          `mapstructure:"readOnly"`
}

// ClusterConfig holds SLURM cluster connection details
type ClusterConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	Token      string `mapstructure:"token"`
	APIVersion string `mapstructure:"apiVersion"`
	Insecure   bool   `mapstructure:"insecure"`
	Timeout    string `mapstructure:"timeout"`
}

// UIConfig holds UI-related settings
type UIConfig struct {
	Skin        string `mapstructure:"skin"`
	Logoless    bool   `mapstructure:"logoless"`
	Crumbsless  bool   `mapstructure:"crumbsless"`
	Statusless  bool   `mapstructure:"statusless"`
	Headless    bool   `mapstructure:"headless"`
	NoIcons     bool   `mapstructure:"noIcons"`
	EnableMouse bool   `mapstructure:"enableMouse"`
}

// ViewsConfig holds view-specific settings
type ViewsConfig struct {
	Jobs       JobsViewConfig       `mapstructure:"jobs"`
	Nodes      NodesViewConfig      `mapstructure:"nodes"`
	Partitions PartitionsViewConfig `mapstructure:"partitions"`
}

// JobsViewConfig holds jobs view settings
type JobsViewConfig struct {
	Columns        []string `mapstructure:"columns"`
	ShowOnlyActive bool     `mapstructure:"showOnlyActive"`
	DefaultSort    string   `mapstructure:"defaultSort"`
	MaxJobs        int      `mapstructure:"maxJobs"`
}

// NodesViewConfig holds nodes view settings
type NodesViewConfig struct {
	GroupBy         string `mapstructure:"groupBy"`
	ShowUtilization bool   `mapstructure:"showUtilization"`
	MaxNodes        int    `mapstructure:"maxNodes"`
}

// PartitionsViewConfig holds partitions view settings
type PartitionsViewConfig struct {
	ShowQueueDepth bool `mapstructure:"showQueueDepth"`
	ShowWaitTime   bool `mapstructure:"showWaitTime"`
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	Streaming bool `mapstructure:"streaming"`
	Pulseye   bool `mapstructure:"pulseye"`
	Xray      bool `mapstructure:"xray"`
}

// ShortcutConfig represents a custom keyboard shortcut
type ShortcutConfig struct {
	Key         string `mapstructure:"key"`
	Action      string `mapstructure:"action"`
	Description string `mapstructure:"description"`
}

// PluginConfig represents a plugin configuration
type PluginConfig struct {
	Name    string                 `mapstructure:"name"`
	Enabled bool                   `mapstructure:"enabled"`
	Path    string                 `mapstructure:"path"`
	Config  map[string]interface{} `mapstructure:"config"`
}

// PluginSettings contains global plugin settings
type PluginSettings struct {
	EnableAll    bool   `mapstructure:"enableAll"`
	PluginDir    string `mapstructure:"pluginDir"`
	AutoDiscover bool   `mapstructure:"autoDiscover"`
	SafeMode     bool   `mapstructure:"safeMode"`     // Disable external plugins
	MaxMemoryMB  int    `mapstructure:"maxMemoryMB"`  // Memory limit per plugin
	MaxCPUPercent float64 `mapstructure:"maxCPUPercent"` // CPU limit per plugin
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
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
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
	if err := cfg.setCurrentCluster(); err != nil {
		return nil, err
	}

	// Ensure config directory exists
	configDir := filepath.Join(os.Getenv("HOME"), ".s9s")
	os.MkdirAll(configDir, 0755)

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// General defaults
	v.SetDefault("refreshRate", "2s")
	v.SetDefault("maxRetries", 3)
	v.SetDefault("currentContext", "default")
	v.SetDefault("useMockClient", true)

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
}

// applyEnvironmentOverrides applies environment variable overrides
func applyEnvironmentOverrides(cfg *Config) {
	// Check for cluster endpoint override
	if endpoint := os.Getenv("SLURM_REST_URL"); endpoint != "" {
		if cfg.CurrentContext == "" || len(cfg.Contexts) == 0 {
			// Create a default context if none exists
			cfg.Contexts = append(cfg.Contexts, ContextConfig{
				Name: "default",
				Cluster: ClusterConfig{
					Endpoint:   endpoint,
					Token:      os.Getenv("SLURM_JWT"),
					APIVersion: getEnvOrDefault("SLURM_API_VERSION", "v0.0.43"),
				},
			})
			cfg.CurrentContext = "default"
		}
	}
}

// setCurrentCluster sets the current cluster configuration based on context
func (c *Config) setCurrentCluster() error {
	if c.CurrentContext == "" {
		c.CurrentContext = "default"
	}

	// Find the current context
	for _, ctx := range c.Contexts {
		if ctx.Name == c.CurrentContext {
			c.Cluster = ctx.Cluster
			return nil
		}
	}

	// If no context found but we have environment variables, use them
	if endpoint := os.Getenv("SLURM_REST_URL"); endpoint != "" {
		c.Cluster = ClusterConfig{
			Endpoint:   endpoint,
			Token:      os.Getenv("SLURM_JWT"),
			APIVersion: getEnvOrDefault("SLURM_API_VERSION", "v0.0.43"),
			Timeout:    getEnvOrDefault("SLURM_TIMEOUT", "30s"),
		}
		return nil
	}

	return fmt.Errorf("context %q not found", c.CurrentContext)
}

// GetContext returns the context configuration by name
func (c *Config) GetContext(name string) (*ContextConfig, error) {
	for _, ctx := range c.Contexts {
		if ctx.Name == name {
			return &ctx, nil
		}
	}
	return nil, fmt.Errorf("context %q not found", name)
}

// SaveToFile saves the configuration to a file
func (c *Config) SaveToFile(path string) error {
	v := viper.New()

	// Set all values from the config struct
	v.Set("refreshRate", c.RefreshRate)
	v.Set("maxRetries", c.MaxRetries)
	v.Set("currentContext", c.CurrentContext)
	v.Set("contexts", c.Contexts)
	v.Set("ui", c.UI)
	v.Set("views", c.Views)
	v.Set("features", c.Features)
	v.Set("shortcuts", c.Shortcuts)
	v.Set("aliases", c.Aliases)
	v.Set("plugins", c.Plugins)

	// Write to file
	return v.WriteConfigAs(path)
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
