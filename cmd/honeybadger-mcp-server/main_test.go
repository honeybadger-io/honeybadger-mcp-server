package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// Only an explicit env var or config-file entry may trip the http-mode
// read-only guard, never the built-in default.
func TestRunHTTPRejectsExplicitReadOnly(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("HONEYBADGER_READ_ONLY", "true")

	err := runHTTP(httpCmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read-only") || !strings.Contains(err.Error(), "http mode") {
		t.Errorf("expected read-only rejection, got: %v", err)
	}
}

func TestRunHTTPRejectsConfigFileReadOnly(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader("read-only: true\n")); err != nil {
		t.Fatal(err)
	}

	err := runHTTP(httpCmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read-only") || !strings.Contains(err.Error(), "http mode") {
		t.Errorf("expected read-only rejection, got: %v", err)
	}
}

func TestRunHTTPRejectsReservedEndpointPaths(t *testing.T) {
	for _, path := range []string{
		"/healthz",
		"/.well-known",
		"/.well-known/oauth-protected-resource",
		"/.well-known/anything",
	} {
		t.Run(path, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("endpoint-path", path)
			viper.Set("public-url", "https://mcp.example.com")
			viper.Set("authorization-server", "https://as.example.com")

			err := runHTTP(httpCmd, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "reserved path") {
				t.Errorf("expected reserved-path rejection, got: %v", err)
			}
		})
	}
}

func TestRunHTTPRequiresConfirmSecret(t *testing.T) {
	for name, secret := range map[string]string{"missing": "", "short": "too-short"} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("public-url", "https://mcp.example.com")
			viper.Set("authorization-server", "https://as.example.com")
			viper.Set("confirm-secret", secret)

			err := runHTTP(httpCmd, nil)
			if err == nil || !strings.Contains(err.Error(), "MCP_CONFIRM_SECRET") {
				t.Errorf("expected confirm-secret rejection, got: %v", err)
			}
		})
	}
}

func TestRunHTTPAllowsDefaultReadOnly(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetDefault("read-only", true)

	// With read-only left at its default, the guard must not trip; boot
	// proceeds to the next config check (missing public-url).
	err := runHTTP(httpCmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "read-only") {
		t.Errorf("default read-only value should not be rejected, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--public-url") {
		t.Errorf("expected missing public-url error, got: %v", err)
	}
}

// HONEYBADGER_API_TOKEN is the name the v3 docs use, and the older
// HONEYBADGER_PERSONAL_AUTH_TOKEN keeps working. When both are set the new name
// wins, so a stale legacy value left in a shell can't shadow it.
func TestAuthTokenEnvNames(t *testing.T) {
	cases := []struct {
		name   string
		apiTok string
		legacy string
		want   string
	}{
		{"new name alone", "hbu_new", "", "hbu_new"},
		{"legacy name alone", "", "hbu_legacy", "hbu_legacy"},
		{"both set", "hbu_new", "hbu_legacy", "hbu_new"},
		{"new name empty", "", "hbu_legacy", "hbu_legacy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			// Keep a real ~/.honeybadger-mcp-server.yaml out of the test.
			t.Setenv("HOME", t.TempDir())
			t.Setenv("HONEYBADGER_API_TOKEN", tc.apiTok)
			t.Setenv("HONEYBADGER_PERSONAL_AUTH_TOKEN", tc.legacy)

			initConfig()

			if got := viper.GetString("auth-token"); got != tc.want {
				t.Errorf("auth-token = %q, want %q", got, tc.want)
			}
		})
	}
}

// documentedEnv lists every env var initConfig binds.
var documentedEnv = []string{
	"HONEYBADGER_API_TOKEN", "HONEYBADGER_PERSONAL_AUTH_TOKEN", "HONEYBADGER_API_URL", "HONEYBADGER_INSTRUCTIONS_URL",
	"LOG_LEVEL", "HONEYBADGER_READ_ONLY", "MCP_ADDRESS", "MCP_ENDPOINT_PATH", "MCP_STATELESS",
	"MCP_PUBLIC_URL", "MCP_AUTHORIZATION_SERVER_URL", "MCP_RESOURCE_URL", "MCP_CONFIRM_SECRET",
}

// clearDocumentedEnv keeps values from the developer's shell out of a test.
// Viper treats an empty env var as unset.
func clearDocumentedEnv(t *testing.T) {
	t.Helper()
	for _, k := range documentedEnv {
		t.Setenv(k, "")
	}
}

// Only the documented env names configure the server. Viper's AutomaticEnv
// used to map every key to an unprefixed name (AUTH_TOKEN, READ_ONLY, ...)
// that beat the documented one, so a stray READ_ONLY=false in a shell could
// turn off read-only mode.
func TestGenericEnvNamesIgnored(t *testing.T) {
	generic := map[string]string{
		"AUTH_TOKEN":           "generic-token",
		"API_URL":              "https://generic.example.com",
		"INSTRUCTIONS_URL":     "https://generic.example.com/instructions",
		"READ_ONLY":            "false",
		"ADDRESS":              ":9999",
		"ENDPOINT_PATH":        "/generic",
		"STATELESS":            "false",
		"PUBLIC_URL":           "https://generic.example.com",
		"AUTHORIZATION_SERVER": "https://generic.example.com",
		"RESOURCE_URL":         "https://generic.example.com/mcp",
		"CONFIRM_SECRET":       "generic-secret",
	}

	t.Run("documented names win", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		// Keep a real ~/.honeybadger-mcp-server.yaml out of the test.
		t.Setenv("HOME", t.TempDir())
		for k, v := range generic {
			t.Setenv(k, v)
		}
		documented := map[string]string{
			"HONEYBADGER_PERSONAL_AUTH_TOKEN": "hbu_documented",
			"HONEYBADGER_API_URL":             "https://app.example.com",
			"HONEYBADGER_INSTRUCTIONS_URL":    "https://docs.example.com/instructions",
			"HONEYBADGER_READ_ONLY":           "true",
			"MCP_ADDRESS":                     ":8081",
			"MCP_ENDPOINT_PATH":               "/documented",
			"MCP_STATELESS":                   "true",
			"MCP_PUBLIC_URL":                  "https://mcp.example.com",
			"MCP_AUTHORIZATION_SERVER_URL":    "https://as.example.com",
			"MCP_RESOURCE_URL":                "https://mcp.example.com/mcp",
			"MCP_CONFIRM_SECRET":              "documented-secret",
		}
		for k, v := range documented {
			t.Setenv(k, v)
		}

		initConfig()

		want := map[string]string{
			"auth-token":           "hbu_documented",
			"api-url":              "https://app.example.com",
			"instructions-url":     "https://docs.example.com/instructions",
			"read-only":            "true",
			"address":              ":8081",
			"endpoint-path":        "/documented",
			"stateless":            "true",
			"public-url":           "https://mcp.example.com",
			"authorization-server": "https://as.example.com",
			"resource-url":         "https://mcp.example.com/mcp",
			"confirm-secret":       "documented-secret",
		}
		for key, w := range want {
			if got := viper.GetString(key); got != w {
				t.Errorf("%s = %q, want %q", key, got, w)
			}
		}
	})

	t.Run("generic names alone change nothing", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		t.Setenv("HOME", t.TempDir())
		clearDocumentedEnv(t)
		for k, v := range generic {
			t.Setenv(k, v)
		}

		initConfig()

		if !viper.GetBool("read-only") {
			t.Error("READ_ONLY=false turned off read-only mode")
		}
		for _, key := range []string{
			"auth-token", "api-url", "instructions-url", "address", "endpoint-path",
			"stateless", "public-url", "authorization-server", "resource-url", "confirm-secret",
		} {
			if viper.IsSet(key) {
				t.Errorf("%s set from a generic env var", key)
			}
		}
	})
}

// Dropping AutomaticEnv must keep env > config file > default.
func TestEnvBeatsConfigFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearDocumentedEnv(t)
	if err := os.WriteFile(filepath.Join(home, ".honeybadger-mcp-server.yaml"),
		[]byte("api-url: https://file.example.com\nlog-level: debug\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HONEYBADGER_API_URL", "https://env.example.com")

	initConfig()

	if got := viper.GetString("api-url"); got != "https://env.example.com" {
		t.Errorf("api-url = %q, want env value", got)
	}
	if got := viper.GetString("log-level"); got != "debug" {
		t.Errorf("log-level = %q, want config file value", got)
	}
	if !viper.GetBool("read-only") {
		t.Error("read-only default lost")
	}
}
