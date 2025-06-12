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
				AuthToken: "test-token",
				APIURL:    "https://api.honeybadger.io/v2",
				LogLevel:  "info",
			},
			wantErr: false,
		},
		{
			name: "missing api token",
			config: Config{
				AuthToken: "",
				APIURL:    "https://api.honeybadger.io/v2",
				LogLevel:  "info",
			},
			wantErr: true,
		},
		{
			name: "minimal valid config",
			config: Config{
				AuthToken: "token",
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
		name      string
		authToken string
		apiURL    string
		logLevel  string
		wantErr   bool
	}{
		{
			name:      "valid configuration",
			authToken: "test-token",
			apiURL:    "https://api.honeybadger.io/v2",
			logLevel:  "info",
			wantErr:   false,
		},
		{
			name:      "missing api token",
			authToken: "",
			apiURL:    "https://api.honeybadger.io/v2",
			logLevel:  "info",
			wantErr:   true,
		},
		{
			name:      "empty url and log level",
			authToken: "test-token",
			apiURL:    "",
			logLevel:  "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.authToken, tt.apiURL, tt.logLevel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && cfg != nil {
				if cfg.AuthToken != tt.authToken {
					t.Errorf("Load() AuthToken = %v, want %v", cfg.AuthToken, tt.authToken)
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
