package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harriteja/mcp-go-sdk/pkg/client"
	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/response"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// serverHandler implements http.Handler for the MCP server
type serverHandler struct {
	srv *server.Server
}

func (h *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Normalize path by trimming trailing slash and removing version prefix
	path := strings.TrimSuffix(r.URL.Path, "/")
	path = strings.TrimPrefix(path, "/v1")

	// Check method
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Handle request based on path
	switch path {
	case "/initialize":
		var req types.InitializeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		resp, err := h.srv.Initialize(r.Context(), &req)
		if err != nil {
			if mcpErr, ok := err.(*types.Error); ok {
				response.WriteError(w, mcpErr.Code, mcpErr.Message)
				return
			}
			response.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response.WriteJSON(w, http.StatusOK, resp)

	case "/listTools":
		tools, err := h.srv.ListTools(r.Context())
		if err != nil {
			if mcpErr, ok := err.(*types.Error); ok {
				response.WriteError(w, mcpErr.Code, mcpErr.Message)
				return
			}
			response.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response.WriteJSON(w, http.StatusOK, tools)

	case "/callTool":
		var req struct {
			Name string                 `json:"name"`
			Args map[string]interface{} `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Validate tool name
		tools, err := h.srv.ListTools(r.Context())
		if err != nil {
			if mcpErr, ok := err.(*types.Error); ok {
				response.WriteError(w, mcpErr.Code, mcpErr.Message)
				return
			}
			response.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		toolExists := false
		for _, t := range tools {
			if t.Name == req.Name {
				toolExists = true
				break
			}
		}
		if !toolExists {
			response.WriteError(w, http.StatusNotFound, "Tool not found")
			return
		}

		result, err := h.srv.CallTool(r.Context(), req.Name, req.Args)
		if err != nil {
			if mcpErr, ok := err.(*types.Error); ok {
				response.WriteError(w, mcpErr.Code, mcpErr.Message)
				return
			}
			response.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response.WriteJSON(w, http.StatusOK, result)

	default:
		response.WriteError(w, http.StatusNotFound, "endpoint not found")
	}
}

func TestServerIntegration(t *testing.T) {
	// Create server and register handlers
	srv := server.New(server.Options{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Register test tool
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "test-tool",
				Description: "A test tool",
				Parameters: &types.Parameters{
					Type: "object",
					Properties: map[string]types.Parameter{
						"input": {
							Type:        "string",
							Description: "Input parameter",
						},
					},
				},
			},
		}, nil
	})

	srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		if name != "test-tool" {
			return nil, types.NewError(404, "Tool not found")
		}
		input, ok := args["input"].(string)
		if !ok {
			return nil, types.NewError(400, "Invalid input parameter")
		}
		return input + " success", nil
	})

	// Create server handler
	handler := &serverHandler{srv: srv}

	// Create test HTTP server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Create client
	cli := client.New(client.Options{
		ServerURL: ts.URL,
		ClientInfo: types.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	// Test initialization
	t.Run("Initialize", func(t *testing.T) {
		err := cli.Initialize(context.Background())
		require.NoError(t, err)
	})

	// Test list tools
	t.Run("ListTools", func(t *testing.T) {
		tools, err := cli.ListTools(context.Background())
		require.NoError(t, err)
		require.Len(t, tools, 1)
		assert.Equal(t, "test-tool", tools[0].Name)
		assert.NotNil(t, tools[0].Parameters)
		assert.Equal(t, "object", tools[0].Parameters.Type)
		assert.Len(t, tools[0].Parameters.Properties, 1)
		assert.Equal(t, "string", tools[0].Parameters.Properties["input"].Type)
	})

	// Test call tool
	t.Run("CallTool", func(t *testing.T) {
		result, err := cli.CallTool(context.Background(), "test-tool", map[string]interface{}{
			"input": "test",
		})
		require.NoError(t, err)
		assert.Equal(t, "test success", result)
	})

	// Test call tool with invalid parameters
	t.Run("CallToolInvalidParams", func(t *testing.T) {
		_, err := cli.CallTool(context.Background(), "test-tool", nil)
		require.Error(t, err)
		assert.Equal(t, "failed to call tool: MCP error 400: Invalid input parameter", err.Error())
	})

	// Test call non-existent tool
	t.Run("CallToolNotFound", func(t *testing.T) {
		_, err := cli.CallTool(context.Background(), "non-existent", nil)
		require.Error(t, err)
		assert.Equal(t, "failed to call tool: MCP error 404: Tool not found", err.Error())
	})
}

func TestServerWithErrors(t *testing.T) {
	// Create server that fails after 3 requests
	var requestCount int
	srv := server.New(server.Options{
		Name:    "test-server",
		Version: "1.0.0",
	})

	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		requestCount++
		if requestCount > 3 {
			return nil, types.NewError(500, "Internal server error")
		}
		return []types.Tool{}, nil
	})

	// Create server handler
	handler := &serverHandler{srv: srv}

	// Create test HTTP server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Create client
	cli := client.New(client.Options{
		ServerURL: ts.URL,
		ClientInfo: types.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// First three requests should succeed
		for i := 0; i < 3; i++ {
			_, err := cli.ListTools(context.Background())
			require.NoError(t, err)
		}

		// Fourth request should fail
		_, err := cli.ListTools(context.Background())
		require.Error(t, err)
		assert.Equal(t, "failed to list tools: MCP error 500: Internal server error", err.Error())
	})
}
