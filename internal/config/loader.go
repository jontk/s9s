package config

import ()

// LoadConfig loads configuration with optional path
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return Load()
	}
	return LoadWithPath(configPath)
}