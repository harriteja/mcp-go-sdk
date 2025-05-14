# Getting Started with MCP Go SDK

This guide will help you get started with using the MCP Go SDK to build MCP-compatible services.

## Installation

To use the MCP Go SDK in your project, you need Go 1.22 or higher. Install the SDK using:

```bash
go get github.com/harriteja/mcp-go-sdk
```

## Quick Start

### Creating a Simple Server

Here's a minimal example of creating an MCP server:

```go
package main

import (
    "context"
    "log"

    "github.com/harriteja/mcp-go-sdk/pkg/server"
    "github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
    // Create server
    srv := server.New(server.Options{
        Name:    "example-server",
        Version: "1.0.0",
    })

    // Register a simple tool
    srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
        return []types.Tool{
            {
                Name:        "hello",
                Description: "Says hello to the user",
            },
        }, nil
    })

    srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
        if name != "hello" {
            return nil, &types.Error{
                Code:    404,
                Message: "Tool not found",
            }
        }
        return map[string]interface{}{
            "message": "Hello, World!",
        }, nil
    })

    // Start server
    if err := srv.ListenAndServe(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

### Creating a Simple Client

Here's how to create a client to interact with the server:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/harriteja/mcp-go-sdk/pkg/client"
    "github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
    // Create client
    cli := client.New(client.Options{
        ServerURL: "http://localhost:8080",
        ClientInfo: types.Implementation{
            Name:    "example-client",
            Version: "1.0.0",
        },
    })

    // Initialize client
    if err := cli.Initialize(context.Background()); err != nil {
        log.Fatal(err)
    }

    // Call the hello tool
    result, err := cli.CallTool(context.Background(), "hello", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Print result
    fmt.Println(result.(map[string]interface{})["message"])
}
```

## Basic Concepts

### Tools

Tools are the primary way to expose functionality in an MCP server. Each tool:
- Has a unique name
- Has a description
- Can accept input parameters
- Returns a result

### Transport Protocols

The SDK supports multiple transport protocols:
- HTTP/REST
- WebSocket
- Standard I/O
- Custom protocols via the Transport interface

### Error Handling

The SDK uses structured error types for consistent error handling:
```go
type Error struct {
    Code    int
    Message string
    Data    map[string]interface{}
}
```

## Next Steps

- Read the [Core Concepts](../concepts/README.md) guide
- Check out the [Examples](../examples/README.md)
- Learn about [Best Practices](../best-practices/README.md) 