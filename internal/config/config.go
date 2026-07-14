package config

import (
	"errors"
	"fmt"
)

const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// DefaultInstructionsURL is where the docs site publishes the LLM
// instruction sets (index.json plus one .txt per set).
const DefaultInstructionsURL = "https://docs.honeybadger.io/resources/llms/instructions"

type Config struct {
	AuthToken       string
	APIURL          string
	InstructionsURL string
	LogLevel        string
	ReadOnly        bool
	TransportMode   string
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

func Load(authToken, apiURL, instructionsURL, logLevel string, readOnly bool, transportMode string) (*Config, error) {
	if instructionsURL == "" {
		instructionsURL = DefaultInstructionsURL
	}
	cfg := &Config{
		AuthToken:       authToken,
		APIURL:          apiURL,
		InstructionsURL: instructionsURL,
		LogLevel:        logLevel,
		ReadOnly:        readOnly,
		TransportMode:   transportMode,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
