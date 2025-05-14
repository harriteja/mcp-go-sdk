package main

import (
	"context"
	"log"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/stdio"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func main() {
	// Create server
	srv := server.New(server.Options{
		Name:         "simple-stdio-server",
		Version:      "1.0.0",
		Instructions: "A simple MCP server using stdio transport",
	})

	// Register tool handlers
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "echo",
				Description: "Echoes back the input",
			},
		}, nil
	})

	srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		if name != "echo" {
			return nil, &types.Error{
				Code:    404,
				Message: "Tool not found",
			}
		}

		// Echo back all arguments
		return args, nil
	})

	// Create stdio transport
	transport := stdio.New(srv, stdio.Options{})

	// Start server
	if err := transport.Start(); err != nil {
		log.Fatal(err)
	}
}
