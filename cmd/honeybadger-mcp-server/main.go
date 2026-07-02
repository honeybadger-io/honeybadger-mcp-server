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

	"github.com/MicahParks/keyfunc/v3"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/config"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/hbmcp"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/httptransport"
	"github.com/honeybadger-io/honeybadger-mcp-server/internal/logging"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Set via -ldflags "-X main.version=..." at Docker build time.
var version = "dev"

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
	// stdio-only: http mode gates on token scope instead.
	stdioCmd.Flags().Bool("read-only", true, "Run in read-only mode, excluding destructive tools")

	// HTTP-specific flags (bound to viper here since only httpCmd defines them)
	httpCmd.Flags().String("address", ":8080", "Address to listen on (e.g. :8080)")
	httpCmd.Flags().String("endpoint-path", "/mcp", "HTTP path the MCP endpoint is served from")
	httpCmd.Flags().Bool("stateless", true, "Run in stateless mode (recommended for horizontally scaled deployments)")
	httpCmd.Flags().String("public-url", "", "Public origin of this MCP server (e.g. https://mcp.honeybadger.io). Required to advertise OAuth Protected Resource Metadata and serve the 401 discovery challenge")
	httpCmd.Flags().String("authorization-server", "", "OAuth authorization server origin (e.g. https://app.honeybadger.io). Required when --public-url is set")
	_ = viper.BindPFlag("address", httpCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("endpoint-path", httpCmd.Flags().Lookup("endpoint-path"))
	_ = viper.BindPFlag("stateless", httpCmd.Flags().Lookup("stateless"))
	_ = viper.BindPFlag("public-url", httpCmd.Flags().Lookup("public-url"))
	_ = viper.BindPFlag("authorization-server", httpCmd.Flags().Lookup("authorization-server"))

	rootCmd.AddCommand(stdioCmd, httpCmd)
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("auth-token", "", "Honeybadger API token (required)")
	cmd.Flags().String("api-url", "https://app.honeybadger.io", "Honeybadger API URL")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
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
	_ = viper.BindEnv("address", "MCP_ADDRESS")
	_ = viper.BindEnv("endpoint-path", "MCP_ENDPOINT_PATH")
	_ = viper.BindEnv("stateless", "MCP_STATELESS")
	_ = viper.BindEnv("public-url", "MCP_PUBLIC_URL")
	_ = viper.BindEnv("authorization-server", "MCP_AUTHORIZATION_SERVER_URL")

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
		"version", version,
		"transport", "stdio",
		"log_level", cfg.LogLevel,
		"api_url", cfg.APIURL,
		"read_only", cfg.ReadOnly)

	mcpServer := hbmcp.NewServer(cfg, version)

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

	// Token scope is the only write-access control in http mode; reject an
	// explicit read-only setting rather than let it be silently ignored.
	// Sources are checked directly because viper.IsSet also reports the
	// SetDefault value, which must not trip this.
	if os.Getenv("HONEYBADGER_READ_ONLY") != "" || viper.InConfig("read-only") {
		return errors.New("configuration error: read-only (HONEYBADGER_READ_ONLY) is not supported in http mode; write access is granted per-token by OAuth scope")
	}

	address := viper.GetString("address")
	endpointPath := httptransport.NormalizeEndpointPath(viper.GetString("endpoint-path"))
	stateless := viper.GetBool("stateless")

	publicURL, err := httptransport.NormalizePublicURL(viper.GetString("public-url"))
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	authServer := strings.TrimSuffix(viper.GetString("authorization-server"), "/")
	// http mode is OAuth-only; --public-url + --authorization-server are required so the
	// 401 challenge can advertise PRM. Use stdio for non-OAuth single-tenant.
	if publicURL == "" || authServer == "" {
		return errors.New("configuration error: --public-url and --authorization-server are required for http mode")
	}
	// /healthz and /.well-known/* are reserved (health checks and PRM); a
	// collision would otherwise panic the mux with a duplicate-pattern error
	// at registration time instead of a clean configuration error.
	if endpointPath == "/healthz" || endpointPath == "/.well-known" || strings.HasPrefix(endpointPath, "/.well-known/") {
		return fmt.Errorf("configuration error: --endpoint-path %q collides with a reserved path (/healthz, /.well-known/...)", endpointPath)
	}

	logger := logging.SetupLogger(cfg.LogLevel)
	logger.Info("Starting Honeybadger MCP Server",
		"version", version,
		"transport", "streamable-http",
		"address", address,
		"endpoint_path", endpointPath,
		"stateless", stateless,
		"public_url", publicURL,
		"authorization_server", authServer,
		"log_level", cfg.LogLevel,
		"api_url", cfg.APIURL)

	mcpServer := hbmcp.NewServer(cfg, version)

	// Both WithStateLess and WithStateful are no-ops when their arg is false.
	sessionOpt := server.WithStateLess(true)
	if !stateless {
		sessionOpt = server.WithStateful(true)
	}

	mcpHandler := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath(endpointPath),
		sessionOpt,
		server.WithStreamableHTTPLogger(logger),
	)

	bootCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	md, err := httptransport.DiscoverAS(bootCtx, authServer)
	if err != nil {
		return err
	}
	if err := httptransport.VerifyJWKSReachable(bootCtx, md.JWKSURI); err != nil {
		return err
	}
	jwks, err := keyfunc.NewDefault([]string{md.JWKSURI})
	if err != nil {
		return err
	}
	logger.Info("AS discovery complete", "issuer", md.Issuer, "jwks_uri", md.JWKSURI)

	rootHandler := http.NewServeMux()
	resource := publicURL + endpointPath
	// Path-qualified per RFC 9728: PRM URL = /.well-known/oauth-protected-resource + resource path.
	prmPath := httptransport.WellKnownPRMPath + endpointPath
	prmAbsURL := publicURL + prmPath
	handler := httptransport.PRMHandler(resource, []string{authServer})
	rootHandler.Handle(prmPath, handler)
	rootHandler.Handle(httptransport.WellKnownPRMPath, handler) // legacy origin-level path for clients that don't derive
	rootHandler.Handle(endpointPath, httptransport.ValidateMiddleware(prmAbsURL, jwks.Keyfunc, md.Issuer, mcpHandler))
	rootHandler.HandleFunc("/healthz", httptransport.HealthHandler)
	logger.Info("OAuth discovery enabled", "resource", resource, "prm_url", prmAbsURL)

	httpServer := &http.Server{
		Addr:    address,
		Handler: rootHandler,
		// No ReadTimeout/WriteTimeout: Streamable HTTP holds long-lived SSE
		// streams. Header reads are bounded so slow-header connections can't
		// pin goroutines when the server is exposed without an L7 proxy.
		ReadHeaderTimeout: 10 * time.Second,
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Close mcp-go sessions first so any long-lived Streamable-HTTP
		// listen connections drop, then let net/http drain the rest.
		if err := mcpHandler.Shutdown(shutdownCtx); err != nil {
			logger.Error("MCP shutdown error", "error", err)
		}
		// Graceful drain; if a client keep-alive holds a connection past
		// the deadline, force-close so we don't hang on shutdown.
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Warn("HTTP graceful shutdown timed out, forcing close", "error", err)
			_ = httpServer.Close()
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
