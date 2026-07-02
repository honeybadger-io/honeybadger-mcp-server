package main

import (
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
