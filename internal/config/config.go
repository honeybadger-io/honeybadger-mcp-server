package config

import (
	"errors"
	"fmt"
)

// Config holds the application configuration
type Config struct {
	AuthToken string
	APIURL    string
	LogLevel  string
}

// Validate checks that all required configuration values are present
func (c *Config) Validate() error {
	if c.AuthToken == "" {
		return errors.New("auth-token is required")
	}
	return nil
}

// Load returns a validated configuration
func Load(authToken, apiURL, logLevel string) (*Config, error) {
	cfg := &Config{
		AuthToken: authToken,
		APIURL:    apiURL,
		LogLevel:  logLevel,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}
