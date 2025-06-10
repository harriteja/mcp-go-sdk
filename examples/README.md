# MCP Go SDK Examples

This directory contains example implementations of MCP servers and clients using the Go SDK.

## Server Examples

### Simple Tool Server

A basic MCP server that provides a calculator tool with basic arithmetic operations.

To run:
```bash
cd servers/simple-tool
go run main.go
```

The server will start on `http://localhost:8080` with the following endpoints:
- `GET /health` - Health check endpoint
- `POST /mcp` - MCP protocol endpoint

## Client Examples

### Simple Calculator Client

A client that connects to the simple tool server and performs arithmetic calculations.

To run (make sure the server is running first):
```bash
cd clients/simple-calculator
go run main.go
```

The client will:
1. Connect to the server
2. List available tools
3. Perform example calculations using the calculator tool

## Framework Examples

The examples demonstrate the framework-agnostic nature of the SDK by providing implementations using different HTTP frameworks:

### Gin Framework
The simple tool server uses the Gin framework, showing how to:
- Set up routes
- Handle requests
- Use middleware
- Integrate with the MCP SDK

### Fiber Framework
An alternative implementation could easily be created using the Fiber framework by:
1. Importing the Fiber adapter instead of Gin
2. Using Fiber's routing and middleware
3. Everything else remains the same

This demonstrates how the SDK's transport layer abstraction allows for easy framework switching without changing the core business logic. 

# MCP-GO-SDK Client Examples

## HTTP Client Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Replace with actual client implementation when available
type HTTPClient struct {
	url string
}

func NewHTTPClient(url string) *HTTPClient {
	return &HTTPClient{url: url}
}

func (c *HTTPClient) Initialize(ctx context.Context, req *types.InitializeRequest) (*types.InitializeResponse, error) {
	// Mock implementation
	return &types.InitializeResponse{
		ServerInfo: types.Implementation{
			Name:    "example-server",
			Version: "1.0.0",
		},
	}, nil
}

func (c *HTTPClient) Initialized(ctx context.Context, notification *types.InitializedNotification) error {
	return nil
}

func (c *HTTPClient) ListTools(ctx context.Context) ([]types.Tool, error) {
	return []types.Tool{
		{
			Name:        "calculator",
			Description: "A simple calculator",
		},
	}, nil
}

func (c *HTTPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"result": 12,
	}, nil
}

func (c *HTTPClient) Ping(ctx context.Context, req *types.PingRequest) (*types.PingResponse, error) {
	return &types.PingResponse{
		Timestamp:       req.Timestamp,
		ServerTimestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}, nil
}

func (c *HTTPClient) Close() error {
	return nil
}

func main() {
	// Get the default logger
	log := logger.New("http-client")
	
	// Create HTTP client
	httpClient := NewHTTPClient("http://localhost:8080/mcp")

	// Initialize connection
	ctx := context.Background()
	initReq := &types.InitializeRequest{
		ProtocolVersion: "1.0",
		ClientInfo: types.Implementation{
			Name:    "example-http-client",
			Version: "1.0.0",
		},
	}
	
	log.Info(ctx, "client", "initialize", "Initializing connection...")
	initResp, err := httpClient.Initialize(ctx, initReq)
	if err != nil {
		log.Panic(ctx, "client", "initialize", "Failed to initialize connection: " + err.Error())
	}
	
	log.Info(ctx, "client", "initialize", fmt.Sprintf("Connected to server: %s v%s", 
		initResp.ServerInfo.Name, 
		initResp.ServerInfo.Version))

	// Send initialized notification
	notif := &types.InitializedNotification{}
	if err := httpClient.Initialized(ctx, notif); err != nil {
		log.Error(ctx, "client", "initialized", "Failed to send initialized notification: " + err.Error())
	}

	// List available tools
	tools, err := httpClient.ListTools(ctx)
	if err != nil {
		log.Error(ctx, "client", "listTools", "Failed to list tools: " + err.Error())
	} else {
		log.Info(ctx, "client", "listTools", fmt.Sprintf("Found %d tools", len(tools)))
		for _, tool := range tools {
			log.Info(ctx, "client", "listTools", fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
		}
	}

	// Call a tool
	log.Info(ctx, "client", "callTool", "Calling calculator tool...")
	result, err := httpClient.CallTool(ctx, "calculator", map[string]interface{}{
		"operation": "add",
		"a": 5,
		"b": 7,
	})
	if err != nil {
		log.Error(ctx, "client", "callTool", "Failed to call tool: " + err.Error())
	} else {
		log.Info(ctx, "client", "callTool", fmt.Sprintf("Tool result: %v", result))
	}

	// Ping the server
	pingResp, err := httpClient.Ping(ctx, &types.PingRequest{
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	})
	if err != nil {
		log.Error(ctx, "client", "ping", "Failed to ping server: " + err.Error())
	} else {
		latency := pingResp.ServerTimestamp - pingResp.Timestamp
		log.Info(ctx, "client", "ping", fmt.Sprintf("Server latency: %dms", latency))
	}

	// Close the client
	if err := httpClient.Close(); err != nil {
		log.Error(ctx, "client", "close", "Error closing client: " + err.Error())
	}
}

## WebSocket Client with Fiber Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	gofiber "github.com/gofiber/fiber/v2"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server"
	fiberadapter "github.com/harriteja/mcp-go-sdk/pkg/server/transport/fiber"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Replace with actual client implementation when available
type WebSocketClient struct {
	url string
}

func NewWebSocketClient(url string) *WebSocketClient {
	return &WebSocketClient{url: url}
}

func (c *WebSocketClient) Initialize(ctx context.Context, req *types.InitializeRequest) (*types.InitializeResponse, error) {
	// Mock implementation
	return &types.InitializeResponse{
		ServerInfo: types.Implementation{
			Name:    "example-server",
			Version: "1.0.0",
		},
	}, nil
}

func (c *WebSocketClient) Initialized(ctx context.Context, notification *types.InitializedNotification) error {
	return nil
}

func (c *WebSocketClient) ListTools(ctx context.Context) ([]types.Tool, error) {
	return []types.Tool{
		{
			Name:        "echo",
			Description: "Echo back the input message",
		},
	}, nil
}

func (c *WebSocketClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	if name == "echo" && args["message"] != nil {
		return args["message"], nil
	}
	return nil, fmt.Errorf("tool not found or invalid arguments")
}

func (c *WebSocketClient) Close() error {
	return nil
}

// Server side setup with Fiber
func setupServer() (*gofiber.App, error) {
	// Create a default logger
	log := logger.New("fiber-server")
	
	// Create a new server
	srv, err := server.New(&server.Options{
		Logger: log,
		ServerInfo: types.Implementation{
			Name:    "example-fiber-server",
			Version: "1.0.0",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Register a sample tool
	srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		if name != "echo" {
			return nil, &types.Error{Code: 404, Message: "Tool not found"}
		}
		
		message, ok := args["message"].(string)
		if !ok {
			return nil, &types.Error{Code: 400, Message: "Message parameter must be a string"}
		}
		return message, nil
	})
	
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "echo",
				Description: "Echo back the input message",
			},
		}, nil
	})

	// Create a Fiber adapter
	adapter := fiberadapter.New(srv, log)

	// Create Fiber app
	app := gofiber.New()

	// Register MCP routes
	adapter.RegisterRoutes(app)

	return app, nil
}

// Client side setup with WebSocket
func runClient(serverURL string) error {
	// Create a logger for the client
	log := logger.New("ws-client")
	
	// Create WebSocket client
	ctx := context.Background()
	wsClient := NewWebSocketClient(serverURL)
	
	// Initialize connection
	log.Info(ctx, "client", "initialize", "Initializing connection...")
	initReq := &types.InitializeRequest{
		ProtocolVersion: "1.0",
		ClientInfo: types.Implementation{
			Name:    "example-websocket-client",
			Version: "1.0.0",
		},
	}
	
	initResp, err := wsClient.Initialize(ctx, initReq)
	if err != nil {
		return fmt.Errorf("failed to initialize connection: %w", err)
	}
	
	log.Info(ctx, "client", "initialize", fmt.Sprintf("Connected to server: %s v%s", 
		initResp.ServerInfo.Name, 
		initResp.ServerInfo.Version))

	// Send initialized notification
	notif := &types.InitializedNotification{}
	if err := wsClient.Initialized(ctx, notif); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	// List available tools
	tools, err := wsClient.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	
	log.Info(ctx, "client", "listTools", fmt.Sprintf("Found %d tools", len(tools)))
	for _, tool := range tools {
		log.Info(ctx, "client", "listTools", fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
	}

	// Call the echo tool
	log.Info(ctx, "client", "callTool", "Calling echo tool...")
	result, err := wsClient.CallTool(ctx, "echo", map[string]interface{}{
		"message": "Hello from WebSocket client!",
	})
	if err != nil {
		return fmt.Errorf("failed to call tool: %w", err)
	}
	
	log.Info(ctx, "client", "callTool", fmt.Sprintf("Echo result: %v", result))

	return nil
}

func main() {
	// Create a logger for the main function
	log := logger.New("main")
	ctx := context.Background()
	
	// Setup server
	app, err := setupServer()
	if err != nil {
		log.Panic(ctx, "main", "setup", "Failed to setup server: " + err.Error())
	}

	// Start server in a goroutine
	go func() {
		log.Info(ctx, "main", "server", "Starting server on :3000")
		if err := app.Listen(":3000"); err != nil {
			log.Error(ctx, "main", "server", "Server failed: " + err.Error())
		}
	}()
	
	// Give the server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Run client
	if err := runClient("ws://localhost:3000/mcp"); err != nil {
		log.Error(ctx, "main", "client", "Client error: " + err.Error())
	}

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Info(ctx, "main", "shutdown", "Shutting down...")
	app.Shutdown()
}

## Creating a Custom Logger Implementation

To create your own logger implementation, simply implement the `types.Logger` interface:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// MyCustomLogger implements the types.Logger interface
type MyCustomLogger struct {
	logger    *log.Logger
	verbosity int
	bucket    string
}

// NewMyCustomLogger creates a new custom logger
func NewMyCustomLogger() *MyCustomLogger {
	return &MyCustomLogger{
		logger:    log.New(os.Stdout, "[CUSTOM] ", log.LstdFlags),
		verbosity: 2, // Set default verbosity
		bucket:    "root",
	}
}

// Access implements types.Logger.Access
func (l *MyCustomLogger) Access(ctx context.Context, message string) {
	l.logger.Printf("ACCESS: %s", message)
}

// Info implements types.Logger.Info
func (l *MyCustomLogger) Info(ctx context.Context, bucket, handler, message string) {
	l.logger.Printf("INFO [%s/%s]: %s", bucket, handler, message)
}

// Warn implements types.Logger.Warn
func (l *MyCustomLogger) Warn(ctx context.Context, bucket, handler, message string) {
	l.logger.Printf("WARN [%s/%s]: %s", bucket, handler, message)
}

// Error implements types.Logger.Error
func (l *MyCustomLogger) Error(ctx context.Context, bucket, handler, message string) {
	l.logger.Printf("ERROR [%s/%s]: %s", bucket, handler, message)
}

// Panic implements types.Logger.Panic
func (l *MyCustomLogger) Panic(ctx context.Context, bucket, handler, message string) {
	l.logger.Panicf("PANIC [%s/%s]: %s", bucket, handler, message)
}

// V implements types.Logger.V
func (l *MyCustomLogger) V(n int) bool {
	return l.verbosity >= n
}

// Sub implements types.Logger.Sub
func (l *MyCustomLogger) Sub(name string) types.Logger {
	return &MyCustomLogger{
		logger:    l.logger,
		verbosity: l.verbosity,
		bucket:    fmt.Sprintf("%s.%s", l.bucket, name),
	}
}

// SubWithIncrement implements types.Logger.SubWithIncrement
func (l *MyCustomLogger) SubWithIncrement(name string, n int) types.Logger {
	return &MyCustomLogger{
		logger:    l.logger,
		verbosity: l.verbosity + n,
		bucket:    fmt.Sprintf("%s.%s", l.bucket, name),
	}
}

func main() {
	// Create your custom logger
	customLogger := NewMyCustomLogger()
	
	// Set it as the default logger
	logger.SetDefaultLogger(customLogger)
	
	// Now all components that use the default logger will use your implementation
	log := logger.New("my-app")
	
	ctx := context.Background()
	log.Info(ctx, "main", "start", "Application started with custom logger")
} 