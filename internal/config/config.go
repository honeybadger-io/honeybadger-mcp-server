package config

import (
	"errors"
	"fmt"
)

const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

type Config struct {
	AuthToken     string
	APIURL        string
	LogLevel      string
	ReadOnly      bool
	TransportMode string
}

func (c *Config) Validate() error {
	// http mode takes the Bearer per-request; startup AuthToken is unused.
	if c.TransportMode == TransportHTTP {
		return nil
	}
	if c.AuthToken == "" {
		return errors.New("auth-token is required")
	}
	return nil
}

func Load(authToken, apiURL, logLevel string, readOnly bool, transportMode string) (*Config, error) {
	cfg := &Config{
		AuthToken:     authToken,
		APIURL:        apiURL,
		LogLevel:      logLevel,
		ReadOnly:      readOnly,
		TransportMode: transportMode,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
