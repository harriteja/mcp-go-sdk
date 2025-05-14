package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	"go.uber.org/zap"

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
	Logger *zap.Logger
}

// Transport implements stdio transport
type Transport struct {
	srv    *server.Server
	reader *bufio.Reader
	writer *bufio.Writer
	logger *zap.Logger
}

// New creates a new stdio transport
func New(srv *server.Server, opts Options) *Transport {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
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
	for {
		// Read request
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Parse request
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			t.writeError(500, "failed to parse request: "+err.Error())
			continue
		}

		// Handle request
		switch req.Method {
		case "initialize":
			var params types.InitializeRequest
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.writeError(400, "invalid initialize params: "+err.Error())
				continue
			}
			resp, err := t.srv.Initialize(context.TODO(), &params)
			if err != nil {
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(resp)

		case "listTools":
			tools, err := t.srv.ListTools(context.TODO())
			if err != nil {
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
				t.writeError(400, "invalid callTool params: "+err.Error())
				continue
			}
			result, err := t.srv.CallTool(context.TODO(), params.Name, params.Args)
			if err != nil {
				if mcpErr, ok := err.(*types.Error); ok {
					t.writeError(mcpErr.Code, mcpErr.Message)
				} else {
					t.writeError(500, err.Error())
				}
				continue
			}
			t.writeResponse(result)

		case "listPrompts":
			result, err := t.srv.ListPrompts(context.TODO())
			if err != nil {
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
				t.writeError(400, "invalid getPrompt params: "+err.Error())
				continue
			}
			result, err := t.srv.GetPrompt(context.TODO(), params.Name, params.Args)
			if err != nil {
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		case "listResources":
			result, err := t.srv.ListResources(context.TODO())
			if err != nil {
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		case "readResource":
			var params struct {
				URI string `json:"uri"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.writeError(400, "invalid readResource params: "+err.Error())
				continue
			}
			data, mimeType, err := t.srv.ReadResource(context.TODO(), params.URI)
			if err != nil {
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
			result, err := t.srv.ListResourceTemplates(context.TODO())
			if err != nil {
				t.writeError(500, err.Error())
				continue
			}
			t.writeResponse(result)

		default:
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
		t.logger.Error("failed to write response", zap.Error(err))
		return
	}
	if err := t.writer.Flush(); err != nil {
		t.logger.Error("failed to flush response", zap.Error(err))
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
		t.logger.Error("failed to write error", zap.Error(err))
		return
	}
	if err := t.writer.Flush(); err != nil {
		t.logger.Error("failed to flush error", zap.Error(err))
	}
}
