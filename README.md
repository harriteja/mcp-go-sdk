# MCP Go SDK

The Model Context Protocol (MCP) Go SDK provides a framework for building MCP-compatible clients and servers in Go. This SDK implements the MCP specification, enabling seamless communication between AI models and applications.

## Features

- Framework-agnostic design with support for multiple HTTP frameworks (Gin, Fiber)
- Clean architecture with clear separation of concerns
- Type-safe API with generics support
- Async/concurrent request handling
- Comprehensive examples and documentation
- Built-in support for streaming responses
- Flexible middleware system

## Requirements

- Go 1.22 or higher

## Installation

```bash
go get github.com/harriteja/mcp-go-sdk
```

## Quick Start

### Server Example

```go
package main

import (
    "context"
    "log"

    "github.com/harriteja/mcp-go-sdk/pkg/server"
)

func main() {
    srv := server.New(server.Options{
        Name:    "example-server",
        Version: "1.0.0",
    })

    // Register handlers
    srv.OnListTools(func(ctx context.Context) ([]server.Tool, error) {
        return []server.Tool{
            {
                Name:        "example-tool",
                Description: "An example tool",
            },
        }, nil
    })

    if err := srv.ListenAndServe(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

### Client Example

```go
package main

import (
    "context"
    "log"

    "github.com/harriteja/mcp-go-sdk/pkg/client"
)

func main() {
    cli := client.New(client.Options{
        ServerURL: "http://localhost:8080",
    })

    tools, err := cli.ListTools(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, tool := range tools {
        log.Printf("Found tool: %s", tool.Name)
    }
}
```

## Documentation

For detailed documentation, please visit the [docs](./docs) directory.

## Examples

Check out the [examples](./examples) directory for more detailed examples of both client and server implementations.

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 