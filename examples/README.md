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