package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
and supports STDIO and Streamable HTTP transports.`,
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Run the MCP server in STDIO mode",
		Long:  `Run the MCP server using standard input/output for communication.`,
		RunE:  runStdio,
	}

	httpCmd = &cobra.Command{
		Use:   "http",
		Short: "Run the MCP server over Streamable HTTP",
		Long: `Run the MCP server over the Streamable HTTP transport (MCP spec 2025-03-26).
By default the server runs in stateless mode, suitable for horizontally scaled
deployments behind a load balancer (e.g. AWS Fargate behind an ALB).`,
		RunE: runHTTP,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.honeybadger-mcp-server.yaml)")

	addCommonFlags(stdioCmd)
	addCommonFlags(httpCmd)

	// HTTP-specific flags (bound to viper here since only httpCmd defines them)
	httpCmd.Flags().String("address", ":8080", "Address to listen on (e.g. :8080)")
	httpCmd.Flags().String("endpoint-path", "/mcp", "HTTP path the MCP endpoint is served from")
	httpCmd.Flags().Bool("stateless", true, "Run in stateless mode (recommended for horizontally scaled deployments)")
	_ = viper.BindPFlag("address", httpCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("endpoint-path", httpCmd.Flags().Lookup("endpoint-path"))

	rootCmd.AddCommand(stdioCmd, httpCmd)
}

// addCommonFlags registers the flags shared by every transport subcommand.
// Viper bindings are NOT created here — they are bound to the active command's
// flagset in loadConfigFromFlags so the unused command's empty default doesn't
// shadow the active command's user-supplied value.
func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("auth-token", "", "Honeybadger API token (required)")
	cmd.Flags().String("api-url", "https://app.honeybadger.io", "Honeybadger API URL")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().Bool("read-only", true, "Run in read-only mode, excluding destructive tools")
}

// loadConfigFromFlags resolves the shared config from viper/flags. It binds
// viper to the active command's flagset just-in-time so each subcommand gets
// its own --auth-token / --api-url / --log-level wired up correctly.
func loadConfigFromFlags(cmd *cobra.Command) (*config.Config, error) {
	_ = viper.BindPFlag("auth-token", cmd.Flags().Lookup("auth-token"))
	_ = viper.BindPFlag("api-url", cmd.Flags().Lookup("api-url"))
	_ = viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))

	// Resolve read-only manually: CLI flag wins, otherwise env/config/default.
	readOnly := viper.GetBool("read-only")
	if cmd.Flags().Changed("read-only") {
		readOnly, _ = cmd.Flags().GetBool("read-only")
	}
	return config.Load(
		viper.GetString("auth-token"),
		viper.GetString("api-url"),
		viper.GetString("log-level"),
		readOnly,
	)
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

	// Set defaults
	viper.SetDefault("read-only", true)

	// Bind specific environment variables
	_ = viper.BindEnv("auth-token", "HONEYBADGER_PERSONAL_AUTH_TOKEN")
	_ = viper.BindEnv("api-url", "HONEYBADGER_API_URL")
	_ = viper.BindEnv("log-level", "LOG_LEVEL")
	_ = viper.BindEnv("read-only", "HONEYBADGER_READ_ONLY")
	_ = viper.BindEnv("address", "HONEYBADGER_MCP_ADDRESS")
	_ = viper.BindEnv("endpoint-path", "HONEYBADGER_MCP_ENDPOINT_PATH")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func runStdio(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	logger := logging.SetupLogger(cfg.LogLevel)
	logger.Info("Starting Honeybadger MCP Server",
		"version", "1.0.0",
		"transport", "stdio",
		"log_level", cfg.LogLevel,
		"api_url", cfg.APIURL,
		"read_only", cfg.ReadOnly)

	mcpServer := hbmcp.NewServer(cfg)

	logger.Info("Server ready, listening on stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		logger.Error("Server error", "error", err)
	}

	logger.Info("Server stopped")
	return nil
}

func runHTTP(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	address := viper.GetString("address")
	endpointPath := viper.GetString("endpoint-path")
	stateless, _ := cmd.Flags().GetBool("stateless")

	logger := logging.SetupLogger(cfg.LogLevel)
	logger.Info("Starting Honeybadger MCP Server",
		"version", "1.0.0",
		"transport", "streamable-http",
		"address", address,
		"endpoint_path", endpointPath,
		"stateless", stateless,
		"log_level", cfg.LogLevel,
		"api_url", cfg.APIURL,
		"read_only", cfg.ReadOnly)

	mcpServer := hbmcp.NewServer(cfg)

	// WithStateLess/WithStateful are each a no-op when their argument is false,
	// so we must dispatch on the flag to actually flip between the two managers.
	sessionOpt := server.WithStateLess(true)
	if !stateless {
		sessionOpt = server.WithStateful(true)
	}
	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath(endpointPath),
		sessionOpt,
		server.WithStreamableHTTPLogger(logger),
	)

	// Run Start() in a goroutine so we can shut down cleanly on signal.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("Server ready, listening on http", "address", address, "path", endpointPath)
		errCh <- httpServer.Start(address)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("Server error", "error", err)
			return err
		}
	case sig := <-sigCh:
		logger.Info("Shutdown signal received", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Shutdown error", "error", err)
		}
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
