# MCP CLI Tool

The MCP CLI tool provides utilities for developing MCP servers and clients using the Go SDK.

## Installation

```bash
# From source
git clone https://github.com/harriteja/mcp-go-sdk
cd mcp-go-sdk
make install
```

## Commands

### Initialize a New Project

Create a new MCP project with basic structure and configuration:

```bash
mcp init my-project
cd my-project
go mod tidy
```

This will create:
- A basic MCP server implementation
- go.mod with required dependencies
- Example tool implementation

### Development Server

Run the server with hot reload during development:

```bash
mcp dev
```

This will:
- Start the MCP server
- Watch for file changes
- Automatically restart the server when files change

### Build for Deployment

Build the server for deployment:

```bash
mcp build
```

This will:
- Create a `build` directory
- Build an optimized binary
- Output the binary location

## Project Structure

A typical MCP project created with `mcp init` has the following structure:

```
my-project/
├── main.go        # Server implementation
└── go.mod         # Go module file
```

## Development

The CLI tool is built using:
- [cobra](https://github.com/spf13/cobra) for command-line interface
- [fsnotify](https://github.com/fsnotify/fsnotify) for file watching
- Standard Go libraries for project scaffolding

## Contributing

Contributions are welcome! Please read our [Contributing Guide](../../CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests. 