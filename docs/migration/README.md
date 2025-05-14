# Migration Guide

This guide helps you migrate between different versions of the MCP Go SDK.

## Version Compatibility Matrix

| MCP Go SDK Version | Go Version Required | Protocol Version | Major Changes |
|-------------------|---------------------|------------------|---------------|
| v1.0.0            | 1.22+               | 1.0              | Initial release |
| v1.1.0            | 1.22+               | 1.0              | Added WebSocket transport |
| v1.2.0            | 1.22+               | 1.1              | Added resource templates |
| v2.0.0            | 1.22+               | 2.0              | Breaking changes in API |

## Breaking Changes

### v2.0.0

1. Client API Changes
   ```go
   // Old way
   client.New(serverURL string)

   // New way
   client.New(client.Options{
       ServerURL: serverURL,
   })
   ```

2. Server API Changes
   ```go
   // Old way
   server.OnTool(name string, handler ToolHandler)

   // New way
   server.OnListTools(handler ListToolsHandler)
   server.OnCallTool(handler CallToolHandler)
   ```

3. Error Handling Changes
   ```go
   // Old way
   type Error struct {
       Message string
   }

   // New way
   type Error struct {
       Code    int
       Message string
       Data    map[string]interface{}
   }
   ```

### v1.2.0

1. Resource API Changes
   ```go
   // Old way
   type Resource struct {
       URI      string
       MimeType string
   }

   // New way
   type Resource struct {
       URI         string
       Name        string
       Description string
       MimeType    string
   }
   ```

## Upgrade Steps

### Upgrading to v2.0.0

1. Update your Go dependencies:
   ```bash
   go get -u github.com/harriteja/mcp-go-sdk@v2.0.0
   ```

2. Update client initialization:
   ```go
   // Before
   cli := client.New("http://localhost:8080")

   // After
   cli := client.New(client.Options{
       ServerURL: "http://localhost:8080",
       ClientInfo: types.Implementation{
           Name:    "my-client",
           Version: "1.0.0",
       },
   })
   ```

3. Update server handlers:
   ```go
   // Before
   srv.OnTool("calculator", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
       // ...
   })

   // After
   srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
       return []types.Tool{
           {
               Name:        "calculator",
               Description: "Basic calculator",
           },
       }, nil
   })

   srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
       if name != "calculator" {
           return nil, &types.Error{Code: 404, Message: "Tool not found"}
       }
       // ...
   })
   ```

4. Update error handling:
   ```go
   // Before
   if err != nil {
       return nil, &types.Error{Message: "Something went wrong"}
   }

   // After
   if err != nil {
       return nil, &types.Error{
           Code:    500,
           Message: "Something went wrong",
           Data: map[string]interface{}{
               "details": err.Error(),
           },
       }
   }
   ```

### Upgrading to v1.2.0

1. Update your Go dependencies:
   ```bash
   go get -u github.com/harriteja/mcp-go-sdk@v1.2.0
   ```

2. Update resource implementations:
   ```go
   // Before
   resources := []types.Resource{
       {
           URI:      "file://example.txt",
           MimeType: "text/plain",
       },
   }

   // After
   resources := []types.Resource{
       {
           URI:         "file://example.txt",
           Name:        "Example Text",
           Description: "An example text file",
           MimeType:    "text/plain",
       },
   }
   ```

## Troubleshooting

### Common Issues

1. **Incompatible Protocol Version**
   
   Error:
   ```
   failed to initialize: incompatible protocol version
   ```

   Solution:
   - Check the compatibility matrix above
   - Update both client and server to compatible versions
   - Ensure protocol versions match between client and server

2. **Missing Required Fields**

   Error:
   ```
   failed to create client: missing required field 'ClientInfo'
   ```

   Solution:
   ```go
   cli := client.New(client.Options{
       ServerURL: "http://localhost:8080",
       ClientInfo: types.Implementation{  // Add this
           Name:    "my-client",
           Version: "1.0.0",
       },
   })
   ```

3. **Handler Signature Mismatch**

   Error:
   ```
   cannot use handler (type func(...)) as type Handler
   ```

   Solution:
   - Check the handler signatures in the API documentation
   - Update handler functions to match the new signatures
   - Use type assertions where necessary

### Version-Specific Issues

#### v2.0.0

1. **Middleware Changes**
   - Middleware now receives a context with method information
   - Update middleware to handle new context values

2. **Transport Changes**
   - Custom transports need to implement new interface methods
   - WebSocket transport requires additional configuration

#### v1.2.0

1. **Resource Template Issues**
   - Resource templates require proper URI patterns
   - Check template syntax in documentation

### Getting Help

If you encounter issues not covered here:

1. Check the [GitHub issues](https://github.com/harriteja/mcp-go-sdk/issues)
2. Join our [Discord community](https://discord.gg/mcp)
3. Read the [API documentation](../api/README.md)
4. Contact support at support@mcp.dev 