package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	httpCmd.Flags().String("public-url", "", "Public origin of this MCP server (e.g. https://mcp.honeybadger.io). Required to advertise OAuth Protected Resource Metadata and serve the 401 discovery challenge")
	httpCmd.Flags().String("auth-server", "", "OAuth authorization server origin (e.g. https://app.honeybadger.io). Required when --public-url is set")
	_ = viper.BindPFlag("address", httpCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("endpoint-path", httpCmd.Flags().Lookup("endpoint-path"))
	_ = viper.BindPFlag("public-url", httpCmd.Flags().Lookup("public-url"))
	_ = viper.BindPFlag("auth-server", httpCmd.Flags().Lookup("auth-server"))

	rootCmd.AddCommand(stdioCmd, httpCmd)
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("auth-token", "", "Honeybadger API token (required)")
	cmd.Flags().String("api-url", "https://app.honeybadger.io", "Honeybadger API URL")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().Bool("read-only", true, "Run in read-only mode, excluding destructive tools")
}

// Bound to viper here (not in addCommonFlags) so the inactive subcommand's
// empty default doesn't shadow the active subcommand's user-supplied value.
func loadConfigFromFlags(cmd *cobra.Command, transportMode string) (*config.Config, error) {
	_ = viper.BindPFlag("auth-token", cmd.Flags().Lookup("auth-token"))
	_ = viper.BindPFlag("api-url", cmd.Flags().Lookup("api-url"))
	_ = viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))

	// Resolve manually: CLI flag wins, otherwise env/config/default.
	readOnly := viper.GetBool("read-only")
	if cmd.Flags().Changed("read-only") {
		readOnly, _ = cmd.Flags().GetBool("read-only")
	}
	return config.Load(
		viper.GetString("auth-token"),
		viper.GetString("api-url"),
		viper.GetString("log-level"),
		readOnly,
		transportMode,
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
	_ = viper.BindEnv("public-url", "HONEYBADGER_MCP_PUBLIC_URL")
	_ = viper.BindEnv("auth-server", "HONEYBADGER_MCP_AUTH_SERVER")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func runStdio(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigFromFlags(cmd, config.TransportStdio)
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
	cfg, err := loadConfigFromFlags(cmd, config.TransportHTTP)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	address := viper.GetString("address")
	endpointPath := normalizeEndpointPath(viper.GetString("endpoint-path"))
	stateless, _ := cmd.Flags().GetBool("stateless")

	publicURL, err := normalizePublicURL(viper.GetString("public-url"))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	authServer := strings.TrimSuffix(viper.GetString("auth-server"), "/")
	// http mode is OAuth-only; --public-url + --auth-server are required so the
	// 401 challenge can advertise PRM. Use stdio for non-OAuth single-tenant.
	if publicURL == "" || authServer == "" {
		return errors.New("configuration error: --public-url and --auth-server are required for http mode")
	}
	if endpointPath == "/healthz" {
		return errors.New("configuration error: --endpoint-path collides with the reserved /healthz handler")
	}

	logger := logging.SetupLogger(cfg.LogLevel)
	logger.Info("Starting Honeybadger MCP Server",
		"version", "1.0.0",
		"transport", "streamable-http",
		"address", address,
		"endpoint_path", endpointPath,
		"stateless", stateless,
		"public_url", publicURL,
		"auth_server", authServer,
		"log_level", cfg.LogLevel,
		"api_url", cfg.APIURL,
		"read_only", cfg.ReadOnly)

	mcpServer := hbmcp.NewServer(cfg)

	// Both WithStateLess and WithStateful are no-ops when their arg is false.
	sessionOpt := server.WithStateLess(true)
	if !stateless {
		sessionOpt = server.WithStateful(true)
	}

	mcpHandler := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath(endpointPath),
		sessionOpt,
		server.WithStreamableHTTPLogger(logger),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return hbmcp.WithAuthToken(ctx, bearerFromRequest(r))
		}),
	)

	rootHandler := http.NewServeMux()
	resource := publicURL + endpointPath
	prmAbsURL := publicURL + wellKnownPRMPath
	rootHandler.Handle(wellKnownPRMPath, prmHandler(resource, []string{authServer}))
	rootHandler.Handle(endpointPath, challengeMiddleware(prmAbsURL, mcpHandler))
	rootHandler.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger.Info("OAuth discovery enabled", "resource", resource, "prm_url", prmAbsURL)

	httpServer := &http.Server{
		Addr:    address,
		Handler: rootHandler,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("Server ready, listening on http", "address", address, "path", endpointPath)
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		} else {
			errCh <- nil
		}
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
