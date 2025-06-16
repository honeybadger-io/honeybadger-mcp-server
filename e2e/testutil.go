package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// MCPTestServer represents a running MCP server for testing
type MCPTestServer struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	scanner  *bufio.Scanner
	mu       sync.Mutex
	apiToken string
	t        *testing.T
}

// MCPMessage represents a JSON-RPC message
type MCPMessage struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method,omitempty"`
	ID      interface{}            `json:"id,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Result  interface{}            `json:"result,omitempty"`
	Error   interface{}            `json:"error,omitempty"`
}

// StartTestServer starts a new MCP server subprocess for testing
func StartTestServer(t *testing.T, apiToken string) (*MCPTestServer, error) {
	if apiToken == "" {
		apiToken = "test-token"
	}

	// Build the server binary
	buildCmd := exec.Command("go", "build", "-o", "../honeybadger-mcp-server-test", "../cmd/honeybadger-mcp-server")
	if err := buildCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build server: %w", err)
	}

	// Start the server
	cmd := exec.Command("../honeybadger-mcp-server-test", "stdio", "--auth-token", apiToken)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	server := &MCPTestServer{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		scanner:  bufio.NewScanner(stdout),
		apiToken: apiToken,
		t:        t,
	}

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Initialize the connection
	if err := server.Initialize(); err != nil {
		server.Stop()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	return server, nil
}

// Initialize sends the initialize message to establish the connection
func (s *MCPTestServer) Initialize() error {
	initMsg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
		Params: map[string]interface{}{
			"protocolVersion": "0.1.0",
			"capabilities": map[string]interface{}{
				"tools": true,
			},
		},
	}

	resp, err := s.SendMessage(initMsg)
	if err != nil {
		return fmt.Errorf("failed to send initialize: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %v", resp.Error)
	}

	return nil
}

// SendMessage sends a message and waits for a response
func (s *MCPTestServer) SendMessage(msg MCPMessage) (*MCPMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Marshal the message
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send the message
	if _, err := fmt.Fprintf(s.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	// Read the response
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("no response received")
	}

	// Parse the response
	var resp MCPMessage
	if err := json.Unmarshal(s.scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// CallTool calls a tool and returns the result
func (s *MCPTestServer) CallTool(toolName string, arguments map[string]interface{}) (interface{}, error) {
	msg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      time.Now().UnixNano(), // Use timestamp as unique ID
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	if arguments == nil {
		msg.Params["arguments"] = map[string]interface{}{}
	}

	resp, err := s.SendMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tool error: %v", resp.Error)
	}

	return resp.Result, nil
}

// ListTools returns the list of available tools
func (s *MCPTestServer) ListTools() ([]interface{}, error) {
	msg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      time.Now().UnixNano(),
	}

	resp, err := s.SendMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("list tools error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("tools not found in result")
	}

	return tools, nil
}

// Stop gracefully stops the test server
func (s *MCPTestServer) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.stdin.Close()
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}

	// Clean up the test binary
	_ = os.Remove("../honeybadger-mcp-server-test")
}

// ReadStderr reads any error output from the server
func (s *MCPTestServer) ReadStderr() string {
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, s.stderr)
	return buf.String()
}
