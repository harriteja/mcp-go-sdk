package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// MCPHandler is a WebSocket handler for MCP protocol methods
type MCPHandler struct {
	server *server.Server
}

// NewMCPHandler creates a new MCP protocol handler
func NewMCPHandler(server *server.Server) *MCPHandler {
	return &MCPHandler{
		server: server,
	}
}

// RegisterMCPHandlers registers all MCP protocol handlers with the WebSocket server
func RegisterMCPHandlers(wsServer *Server, mcpServer *server.Server) {
	handler := NewMCPHandler(mcpServer)

	// Register message handlers
	wsServer.RegisterHandler("initialize", handler)
	wsServer.RegisterHandler("initialized", handler)
	wsServer.RegisterHandler("listTools", handler)
	wsServer.RegisterHandler("callTool", handler)
	wsServer.RegisterHandler("listPrompts", handler)
	wsServer.RegisterHandler("getPrompt", handler)
	wsServer.RegisterHandler("listResources", handler)
	wsServer.RegisterHandler("readResource", handler)
	wsServer.RegisterHandler("ping", handler)
	wsServer.RegisterHandler("cancel", handler)
}

// HandleMessage implements the WebSocket Handler interface
func (h *MCPHandler) HandleMessage(ctx context.Context, conn *websocket.Conn, msg Message) error {
	// Create a request ID context
	// TODO: Extract request ID from message if available

	switch msg.Type {
	case "initialize":
		return h.handleInitialize(ctx, conn, msg)
	case "initialized":
		return h.handleInitialized(ctx, conn, msg)
	case "listTools":
		return h.handleListTools(ctx, conn, msg)
	case "callTool":
		return h.handleCallTool(ctx, conn, msg)
	case "listPrompts":
		return h.handleListPrompts(ctx, conn, msg)
	case "getPrompt":
		return h.handleGetPrompt(ctx, conn, msg)
	case "listResources":
		return h.handleListResources(ctx, conn, msg)
	case "readResource":
		return h.handleReadResource(ctx, conn, msg)
	case "ping":
		return h.handlePing(ctx, conn, msg)
	case "cancel":
		return h.handleCancel(ctx, conn, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleInitialize handles initialize requests
func (h *MCPHandler) handleInitialize(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var req types.InitializeRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return sendErrorResponse(conn, "initialize", err)
	}

	resp, err := h.server.Initialize(ctx, &req)
	if err != nil {
		return sendErrorResponse(conn, "initialize", err)
	}

	return sendResponse(conn, "initialize", resp)
}

// handleInitialized handles initialized notifications
func (h *MCPHandler) handleInitialized(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var notification types.InitializedNotification
	if err := json.Unmarshal(msg.Payload, &notification); err != nil {
		return sendErrorResponse(conn, "initialized", err)
	}

	if err := h.server.Initialized(ctx, &notification); err != nil {
		return sendErrorResponse(conn, "initialized", err)
	}

	// Initialized is a notification and doesn't require a response
	return nil
}

// handleListTools handles list tools requests
func (h *MCPHandler) handleListTools(ctx context.Context, conn *websocket.Conn, msg Message) error {
	tools, err := h.server.ListTools(ctx)
	if err != nil {
		return sendErrorResponse(conn, "listTools", err)
	}

	return sendResponse(conn, "listTools", tools)
}

// handleCallTool handles call tool requests
func (h *MCPHandler) handleCallTool(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var req struct {
		Name string                 `json:"name"`
		Args map[string]interface{} `json:"args"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return sendErrorResponse(conn, "callTool", err)
	}

	result, err := h.server.CallTool(ctx, req.Name, req.Args)
	if err != nil {
		return sendErrorResponse(conn, "callTool", err)
	}

	return sendResponse(conn, "callTool", result)
}

// handleListPrompts handles list prompts requests
func (h *MCPHandler) handleListPrompts(ctx context.Context, conn *websocket.Conn, msg Message) error {
	prompts, err := h.server.ListPrompts(ctx)
	if err != nil {
		return sendErrorResponse(conn, "listPrompts", err)
	}

	return sendResponse(conn, "listPrompts", prompts)
}

// handleGetPrompt handles get prompt requests
func (h *MCPHandler) handleGetPrompt(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var req struct {
		Name string                 `json:"name"`
		Args map[string]interface{} `json:"args"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return sendErrorResponse(conn, "getPrompt", err)
	}

	prompt, err := h.server.GetPrompt(ctx, req.Name, req.Args)
	if err != nil {
		return sendErrorResponse(conn, "getPrompt", err)
	}

	return sendResponse(conn, "getPrompt", prompt)
}

// handleListResources handles list resources requests
func (h *MCPHandler) handleListResources(ctx context.Context, conn *websocket.Conn, msg Message) error {
	resources, err := h.server.ListResources(ctx)
	if err != nil {
		return sendErrorResponse(conn, "listResources", err)
	}

	return sendResponse(conn, "listResources", resources)
}

// handleReadResource handles read resource requests
func (h *MCPHandler) handleReadResource(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var req struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return sendErrorResponse(conn, "readResource", err)
	}

	content, mimeType, err := h.server.ReadResource(ctx, req.URI)
	if err != nil {
		return sendErrorResponse(conn, "readResource", err)
	}

	resp := struct {
		Content  []byte `json:"content"`
		MimeType string `json:"mimeType"`
	}{
		Content:  content,
		MimeType: mimeType,
	}

	return sendResponse(conn, "readResource", resp)
}

// handlePing handles ping requests
func (h *MCPHandler) handlePing(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var req types.PingRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return sendErrorResponse(conn, "ping", err)
	}

	resp, err := h.server.Ping(ctx, &req)
	if err != nil {
		return sendErrorResponse(conn, "ping", err)
	}

	return sendResponse(conn, "ping", resp)
}

// handleCancel handles cancel requests
func (h *MCPHandler) handleCancel(ctx context.Context, conn *websocket.Conn, msg Message) error {
	var req types.CancelRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return sendErrorResponse(conn, "cancel", err)
	}

	if err := h.server.Cancel(ctx, &req); err != nil {
		return sendErrorResponse(conn, "cancel", err)
	}

	// Cancel is a notification and doesn't require a response
	return nil
}

// sendResponse sends a response message
func sendResponse(conn *websocket.Conn, msgType string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return conn.WriteJSON(Message{
		Type:    msgType + "Response",
		Payload: payload,
	})
}

// sendErrorResponse sends an error response message
func sendErrorResponse(conn *websocket.Conn, msgType string, err error) error {
	var mcpErr *types.Error
	code := 500
	message := err.Error()

	if e, ok := types.IsError(err); ok {
		mcpErr = e
		code = e.Code
		message = e.Message
	} else {
		mcpErr = types.NewError(code, message)
	}

	errPayload, err := json.Marshal(mcpErr)
	if err != nil {
		return err
	}

	return conn.WriteJSON(Message{
		Type:    "error",
		Payload: errPayload,
	})
}
