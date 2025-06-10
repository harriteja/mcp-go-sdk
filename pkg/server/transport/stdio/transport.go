package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Options holds configuration for the stdio transport
type Options struct {
	// Reader is the input reader
	Reader io.Reader
	// Writer is the output writer
	Writer io.Writer
	// Logger is the logger to use
	Logger types.Logger
}

// Transport implements stdio transport
type Transport struct {
	srv    *server.Server
	reader *bufio.Reader
	writer *bufio.Writer
	logger types.Logger
}

// New creates a new stdio transport
func New(srv *server.Server, opts Options) *Transport {
	if opts.Logger == nil {
		opts.Logger = types.NewNoOpLogger()
	}

	return &Transport{
		srv:    srv,
		reader: bufio.NewReader(opts.Reader),
		writer: bufio.NewWriter(opts.Writer),
		logger: opts.Logger,
	}
}

// Start starts the transport
func (t *Transport) Start() error {
	ctx := context.Background()
	t.logger.Info(ctx, "stdio", "start", "Starting StdIO transport")

	for {
		// Read request
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				t.logger.Info(ctx, "stdio", "start", "Received EOF, shutting down")
				return nil
			}
			t.logger.Error(ctx, "stdio", "read", "Error reading input: "+err.Error())
			return err
		}

		// Parse request
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			t.logger.Error(ctx, "stdio", "parse", "Failed to parse request: "+err.Error())
			t.writeError(500, "failed to parse request: "+err.Error())
			continue
		}

		t.logger.Info(ctx, "stdio", "request", "Received request: "+req.Method)

		// Handle request
		switch req.Method {
		case "initialize":
			var params types.InitializeRequest
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.logger.Error(ctx, "stdio", "initialize", "Invalid initialize params: "+err.Error())
				t.writeError(400, "invalid initialize params: "+err.Error())
				continue
			}
			resp, err := t.srv.Initialize(ctx, &params)
			if err != nil {
				t.logger.Error(ctx, "stdio", "initialize", "Failed to initialize: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(resp)

		case "initialized":
			var notification types.InitializedNotification
			if err := json.Unmarshal(req.Params, &notification); err != nil {
				t.logger.Error(ctx, "stdio", "initialized", "Invalid initialized params: "+err.Error())
				t.writeError(400, "invalid initialized params: "+err.Error())
				continue
			}
			err := t.srv.Initialized(ctx, &notification)
			if err != nil {
				t.logger.Error(ctx, "stdio", "initialized", "Failed to process initialized notification: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			// No response needed for notifications

		case "ping":
			var params types.PingRequest
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.logger.Error(ctx, "stdio", "ping", "Invalid ping params: "+err.Error())
				t.writeError(400, "invalid ping params: "+err.Error())
				continue
			}
			resp, err := t.srv.Ping(ctx, &params)
			if err != nil {
				t.logger.Error(ctx, "stdio", "ping", "Failed to ping: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(resp)

		case "cancel":
			var params types.CancelRequest
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.logger.Error(ctx, "stdio", "cancel", "Invalid cancel params: "+err.Error())
				t.writeError(400, "invalid cancel params: "+err.Error())
				continue
			}
			err := t.srv.Cancel(ctx, &params)
			if err != nil {
				t.logger.Error(ctx, "stdio", "cancel", "Failed to cancel: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			// No response needed for notifications

		case "listTools":
			tools, err := t.srv.ListTools(ctx)
			if err != nil {
				t.logger.Error(ctx, "stdio", "listTools", "Failed to list tools: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(tools)

		case "callTool":
			var params struct {
				Name string                 `json:"name"`
				Args map[string]interface{} `json:"args"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.logger.Error(ctx, "stdio", "callTool", "Invalid callTool params: "+err.Error())
				t.writeError(400, "invalid callTool params: "+err.Error())
				continue
			}
			result, err := t.srv.CallTool(ctx, params.Name, params.Args)
			if err != nil {
				t.logger.Error(ctx, "stdio", "callTool", "Failed to call tool: "+err.Error())
				if mcpErr, ok := err.(*types.Error); ok {
					t.writeError(mcpErr.Code, mcpErr.Message)
				} else {
					t.writeError(500, err.Error())
				}
				continue
			}
			t.writeResponse(result)

		case "listPrompts":
			result, err := t.srv.ListPrompts(ctx)
			if err != nil {
				t.logger.Error(ctx, "stdio", "listPrompts", "Failed to list prompts: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		case "getPrompt":
			var params struct {
				Name string                 `json:"name"`
				Args map[string]interface{} `json:"args"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.logger.Error(ctx, "stdio", "getPrompt", "Invalid getPrompt params: "+err.Error())
				t.writeError(400, "invalid getPrompt params: "+err.Error())
				continue
			}
			result, err := t.srv.GetPrompt(ctx, params.Name, params.Args)
			if err != nil {
				t.logger.Error(ctx, "stdio", "getPrompt", "Failed to get prompt: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		case "listResources":
			result, err := t.srv.ListResources(ctx)
			if err != nil {
				t.logger.Error(ctx, "stdio", "listResources", "Failed to list resources: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		case "readResource":
			var params struct {
				URI string `json:"uri"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.logger.Error(ctx, "stdio", "readResource", "Invalid readResource params: "+err.Error())
				t.writeError(400, "invalid readResource params: "+err.Error())
				continue
			}
			data, mimeType, err := t.srv.ReadResource(ctx, params.URI)
			if err != nil {
				t.logger.Error(ctx, "stdio", "readResource", "Failed to read resource: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			result := struct {
				Data     []byte `json:"data"`
				MimeType string `json:"mimeType"`
			}{
				Data:     data,
				MimeType: mimeType,
			}
			t.writeResponse(result)

		case "listResourceTemplates":
			result, err := t.srv.ListResourceTemplates(ctx)
			if err != nil {
				t.logger.Error(ctx, "stdio", "listResourceTemplates", "Failed to list resource templates: "+err.Error())
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		default:
			t.logger.Error(ctx, "stdio", "method", "Unknown method: "+req.Method)
			t.writeError(404, "unknown method: "+req.Method)
		}
	}
}

func (t *Transport) writeResponse(v interface{}) {
	resp := struct {
		Result interface{} `json:"result"`
	}{
		Result: v,
	}

	if err := json.NewEncoder(t.writer).Encode(resp); err != nil {
		t.logger.Error(context.Background(), "stdio", "writeResponse", "Failed to write response: "+err.Error())
		return
	}
	if err := t.writer.Flush(); err != nil {
		t.logger.Error(context.Background(), "stdio", "writeResponse", "Failed to flush response: "+err.Error())
	}
}

func (t *Transport) writeError(code int, message string) {
	resp := struct {
		Error *types.Error `json:"error"`
	}{
		Error: &types.Error{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(t.writer).Encode(resp); err != nil {
		t.logger.Error(context.Background(), "stdio", "writeError", "Failed to write error: "+err.Error())
		return
	}
	if err := t.writer.Flush(); err != nil {
		t.logger.Error(context.Background(), "stdio", "writeError", "Failed to flush error: "+err.Error())
	}
}
