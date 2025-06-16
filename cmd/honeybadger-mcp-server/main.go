package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "honeybadger-mcp-server",
		Short: "MCP server for Honeybadger",
		Long: `Honeybadger MCP Server provides a machine-readable interface to the
Honeybadger API using the MCP protocol. It's designed for use with LLM agents
and communicates via STDIO transport.`,
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Run the MCP server in STDIO mode",
		Long:  `Run the MCP server using standard input/output for communication.`,
		RunE:  runStdio,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.honeybadger-mcp-server.yaml)")

	// Stdio command flags
	stdioCmd.Flags().String("auth-token", "", "Honeybadger API token (required)")
	stdioCmd.Flags().String("api-url", "https://app.honeybadger.io", "Honeybadger API URL")
	stdioCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")

	// Bind flags to viper
	viper.BindPFlag("auth-token", stdioCmd.Flags().Lookup("auth-token"))
	viper.BindPFlag("api-url", stdioCmd.Flags().Lookup("api-url"))
	viper.BindPFlag("log-level", stdioCmd.Flags().Lookup("log-level"))

	rootCmd.AddCommand(stdioCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".honeybadger-mcp-server")
	}

	// Environment variable binding
	viper.SetEnvPrefix("")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Bind specific environment variables
	viper.BindEnv("auth-token", "HONEYBADGER_PERSONAL_AUTH_TOKEN")
	viper.BindEnv("api-url", "HONEYBADGER_API_URL")
	viper.BindEnv("log-level", "LOG_LEVEL")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func runStdio(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(
		viper.GetString("auth-token"),
		viper.GetString("api-url"),
		viper.GetString("log-level"),
	)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Setup structured logging with configured level
	logger := logging.SetupLogger(cfg.LogLevel)

	// Create MCP server
	logger.Info("Starting Honeybadger MCP Server",
		"version", "1.0.0",
		"log_level", cfg.LogLevel,
		"api_url", cfg.APIURL)

	mcpServer := hbmcp.NewServer(cfg)

	// Run the server
	logger.Info("Server ready, listening on stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	logger.Info("Server stopped")

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
