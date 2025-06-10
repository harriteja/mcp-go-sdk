# MCP Server Examples

This directory contains examples of different ways to run MCP servers using the Go SDK.

## Simple Servers

- [simple-tool](./simple-tool): Basic example of a standalone MCP server with a simple tool
- [chat-server](./chat-server): Example of an MCP server that provides a chat interface
- [load-balancer](./load-balancer): Example of load balancing across multiple MCP servers
- [filetransfer](./filetransfer): Example of using MCP for file transfer operations

## Integration Examples

These examples show how to integrate an MCP server with existing web frameworks:

### Fiber Integration

The [fiber-integration](./fiber-integration) example demonstrates how to add MCP capabilities to an existing Fiber application. This is useful when you already have a Fiber-based API and want to expose some of its functionality as MCP tools.

Key features:
- Attaching an MCP server to an existing Fiber application
- Using the Fiber adapter to integrate with the Fiber router
- Sharing the same HTTP server for both your API and MCP endpoints

To run the example:
```bash
go run examples/servers/fiber-integration/main.go
```

### Gin Integration

The [gin-integration](./gin-integration) example shows how to integrate MCP with an existing Gin application. This is helpful when you have a Gin-based API and want to add MCP support.

Key features:
- Adding MCP capabilities to an existing Gin application
- Using the Gin adapter to integrate with the Gin router
- Routing both regular HTTP endpoints and MCP endpoints through the same server

To run the example:
```bash
go run examples/servers/gin-integration/main.go
```

## Best Practices

When integrating an MCP server with an existing web application:

1. **Separation of Concerns**: Keep MCP-specific code separate from your regular API handlers.
2. **Shared Resources**: Consider how to share resources (e.g., database connections) between your API and MCP tools.
3. **Error Handling**: Implement consistent error handling between MCP and regular API endpoints.
4. **Authentication**: If needed, apply the same authentication mechanism to both API and MCP endpoints.
5. **Logging**: Use a consistent logging approach for both MCP and API code. 