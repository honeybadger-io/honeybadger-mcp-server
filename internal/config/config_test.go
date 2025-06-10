package config

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIToken: "test-token",
				APIURL:   "https://api.honeybadger.io/v2",
				LogLevel: "info",
			},
			wantErr: false,
		},
		{
			name: "missing api token",
			config: Config{
				APIToken: "",
				APIURL:   "https://api.honeybadger.io/v2",
				LogLevel: "info",
			},
			wantErr: true,
		},
		{
			name: "minimal valid config",
			config: Config{
				APIToken: "token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		apiToken string
		apiURL   string
		logLevel string
		wantErr  bool
	}{
		{
			name:     "valid configuration",
			apiToken: "test-token",
			apiURL:   "https://api.honeybadger.io/v2",
			logLevel: "info",
			wantErr:  false,
		},
		{
			name:     "missing api token",
			apiToken: "",
			apiURL:   "https://api.honeybadger.io/v2",
			logLevel: "info",
			wantErr:  true,
		},
		{
			name:     "empty url and log level",
			apiToken: "test-token",
			apiURL:   "",
			logLevel: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.apiToken, tt.apiURL, tt.logLevel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && cfg != nil {
				if cfg.APIToken != tt.apiToken {
					t.Errorf("Load() APIToken = %v, want %v", cfg.APIToken, tt.apiToken)
				}
				if cfg.APIURL != tt.apiURL {
					t.Errorf("Load() APIURL = %v, want %v", cfg.APIURL, tt.apiURL)
				}
				if cfg.LogLevel != tt.logLevel {
					t.Errorf("Load() LogLevel = %v, want %v", cfg.LogLevel, tt.logLevel)
				}
			}
		})
	}
}