package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// HTTPHandler represents a framework-agnostic HTTP handler
type HTTPHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HTTPHandlerFunc is a function that implements HTTPHandler
type HTTPHandlerFunc func(w http.ResponseWriter, r *http.Request)

func (f HTTPHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// HTTPTransport provides HTTP transport for MCP server
type HTTPTransport struct {
	server *server.Server
	logger *zap.Logger
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(srv *server.Server, logger *zap.Logger) *HTTPTransport {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	return &HTTPTransport{
		server: srv,
		logger: logger,
	}
}

// Handler returns the main HTTP handler for the MCP server
func (t *HTTPTransport) Handler() HTTPHandler {
	return HTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			t.handlePost(w, r)
		case http.MethodGet:
			t.handleGet(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (t *HTTPTransport) handlePost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.writeError(w, errors.Wrap(err, "failed to decode request"))
		return
	}

	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		var params types.InitializeRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.writeError(w, errors.Wrap(err, "failed to unmarshal initialize params"))
			return
		}
		result, err = t.server.Initialize(r.Context(), &params)

	case "listTools":
		result, err = t.server.ListTools(r.Context())

	case "callTool":
		var params struct {
			Name string                 `json:"name"`
			Args map[string]interface{} `json:"args"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.writeError(w, errors.Wrap(err, "failed to unmarshal call tool params"))
			return
		}
		result, err = t.server.CallTool(r.Context(), params.Name, params.Args)

	case "listPrompts":
		result, err = t.server.ListPrompts(r.Context())

	case "getPrompt":
		var params struct {
			Name string                 `json:"name"`
			Args map[string]interface{} `json:"args"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.writeError(w, errors.Wrap(err, "failed to unmarshal get prompt params"))
			return
		}
		result, err = t.server.GetPrompt(r.Context(), params.Name, params.Args)

	case "listResources":
		result, err = t.server.ListResources(r.Context())

	case "readResource":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.writeError(w, errors.Wrap(err, "failed to unmarshal read resource params"))
			return
		}
		data, mimeType, err := t.server.ReadResource(r.Context(), params.URI)
		if err != nil {
			t.writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", mimeType)
		if _, err := w.Write(data); err != nil {
			t.logger.Error("Failed to write data", zap.Error(err))
		}
		return

	case "listResourceTemplates":
		result, err = t.server.ListResourceTemplates(r.Context())

	default:
		t.writeError(w, fmt.Errorf("unknown method: %s", req.Method))
		return
	}

	if err != nil {
		t.writeError(w, err)
		return
	}

	t.writeJSON(w, result)
}

func (t *HTTPTransport) handleGet(w http.ResponseWriter, r *http.Request) {
	// Health check endpoint
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		t.logger.Error("Failed to write data", zap.Error(err))
	}
}

func (t *HTTPTransport) writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (t *HTTPTransport) writeError(w http.ResponseWriter, err error) {
	t.logger.Error("Request error", zap.Error(err))

	mcpErr, ok := err.(*types.Error)
	if !ok {
		mcpErr = &types.Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(mcpErr.Code)
	if err := json.NewEncoder(w).Encode(mcpErr); err != nil {
		t.logger.Error("Failed to encode error response", zap.Error(err))
	}
}
