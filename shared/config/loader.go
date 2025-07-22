package config

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/like-mike/relai-gateway/shared/models"
	"gopkg.in/yaml.v2"
)

var globalConfig *models.Config

// LoadConfig reads and parses the configuration file once
func LoadConfig(path string) (*models.Config, error) {
	if path == "" {
		path = "config.yml"
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	globalConfig = &config
	log.Printf("Configuration loaded from %s", path)
	return &config, nil
}

// GetConfig returns the global configuration
func GetConfig() *models.Config {
	return globalConfig
}

// GetActiveTheme returns the currently active theme
func GetActiveTheme() (*models.Theme, error) {
	if globalConfig == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	activeThemeKey := globalConfig.ActiveTheme
	if activeThemeKey == "" {
		activeThemeKey = "default"
	}

	theme, exists := globalConfig.Themes[activeThemeKey]
	if !exists {
		return nil, fmt.Errorf("theme '%s' not found", activeThemeKey)
	}

	return &theme, nil
}

// GetThemeContextData returns theme data for templates
func GetThemeContextData() (*models.ThemeContextData, error) {
	if globalConfig == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	activeTheme, err := GetActiveTheme()
	if err != nil {
		return nil, err
	}

	return &models.ThemeContextData{
		Config:      globalConfig,
		ActiveTheme: activeTheme,
		ThemeKey:    globalConfig.ActiveTheme,
	}, nil
}
