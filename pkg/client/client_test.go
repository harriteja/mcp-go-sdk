package client

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func TestClient_StdioTransport(t *testing.T) {
	// Create pipes for bidirectional communication
	clientToServerReader, clientToServerWriter := io.Pipe()
	serverToClientReader, serverToClientWriter := io.Pipe()

	// Create client
	cli := New(Options{
		Reader: serverToClientReader,
		Writer: clientToServerWriter,
		ClientInfo: types.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	// Start mock server
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		defer clientToServerReader.Close()
		defer serverToClientWriter.Close()

		// Handle initialize
		var req struct {
			Method string                  `json:"method"`
			Params types.InitializeRequest `json:"params"`
		}
		if err := json.NewDecoder(clientToServerReader).Decode(&req); err != nil {
			t.Errorf("Failed to decode initialize request: %v", err)
			return
		}

		resp := struct {
			Result types.InitializeResponse `json:"result"`
		}{
			Result: types.InitializeResponse{
				ProtocolVersion: req.Params.ProtocolVersion,
				ServerInfo: types.Implementation{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
		}
		if err := json.NewEncoder(serverToClientWriter).Encode(resp); err != nil {
			t.Errorf("Failed to encode initialize response: %v", err)
			return
		}

		// Handle list tools
		if err := json.NewDecoder(clientToServerReader).Decode(&req); err != nil {
			t.Errorf("Failed to decode list tools request: %v", err)
			return
		}

		toolsResp := struct {
			Result []types.Tool `json:"result"`
		}{
			Result: []types.Tool{
				{
					Name:        "test-tool",
					Description: "A test tool",
				},
			},
		}
		if err := json.NewEncoder(serverToClientWriter).Encode(toolsResp); err != nil {
			t.Errorf("Failed to encode list tools response: %v", err)
			return
		}

		// Handle call tool
		var callReq struct {
			Method string `json:"method"`
			Params struct {
				Name string                 `json:"name"`
				Args map[string]interface{} `json:"args"`
			} `json:"params"`
		}
		if err := json.NewDecoder(clientToServerReader).Decode(&callReq); err != nil {
			t.Errorf("Failed to decode call tool request: %v", err)
			return
		}

		callResp := struct {
			Result map[string]interface{} `json:"result"`
		}{
			Result: map[string]interface{}{
				"result": "success",
			},
		}
		if err := json.NewEncoder(serverToClientWriter).Encode(callResp); err != nil {
			t.Errorf("Failed to encode call tool response: %v", err)
			return
		}
	}()

	// Initialize client
	err := cli.Initialize(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-server", cli.serverInfo.Name)

	// List tools
	tools, err := cli.ListTools(context.Background())
	require.NoError(t, err)
	assert.Len(t, tools, 1)
	assert.Equal(t, "test-tool", tools[0].Name)

	// Call tool
	result, err := cli.CallTool(context.Background(), "test-tool", map[string]interface{}{
		"arg": "value",
	})
	require.NoError(t, err)
	assert.Equal(t, "success", result.(map[string]interface{})["result"])

	// Clean up
	clientToServerWriter.Close()
	serverToClientReader.Close()

	// Wait for server to finish
	select {
	case <-serverDone:
		// Server completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for server to complete")
	}
}

func TestClient_StdioError(t *testing.T) {
	// Create pipes for bidirectional communication
	clientToServerReader, clientToServerWriter := io.Pipe()
	serverToClientReader, serverToClientWriter := io.Pipe()

	// Create client
	cli := New(Options{
		Reader: serverToClientReader,
		Writer: clientToServerWriter,
		ClientInfo: types.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	// Start mock server
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		defer clientToServerReader.Close()
		defer serverToClientWriter.Close()

		// Read request
		var req struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(clientToServerReader).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}

		// Send error response
		errResp := struct {
			Error *types.Error `json:"error"`
		}{
			Error: &types.Error{
				Code:    404,
				Message: "Not found",
			},
		}
		if err := json.NewEncoder(serverToClientWriter).Encode(errResp); err != nil {
			t.Errorf("Failed to encode error response: %v", err)
			return
		}
	}()

	// Make request that will fail
	_, err := cli.ListTools(context.Background())
	require.Error(t, err)

	// Check error type and details
	var mcpErr *types.Error
	require.ErrorAs(t, err, &mcpErr, "Expected error to be *types.Error")
	assert.Equal(t, 404, mcpErr.Code)
	assert.Equal(t, "Not found", mcpErr.Message)

	// Clean up
	clientToServerWriter.Close()
	serverToClientReader.Close()

	// Wait for server to finish
	select {
	case <-serverDone:
		// Server completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for server to complete")
	}
}
