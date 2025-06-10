package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Adapter provides Fiber adapter for MCP server
type Adapter struct {
	transport *transport.HTTPTransport
}

// New creates a new Fiber adapter
func New(srv *server.Server, logger types.Logger) *Adapter {
	return &Adapter{
		transport: transport.NewHTTPTransport(srv, logger),
	}
}

// Handler returns a Fiber handler function
func (a *Adapter) Handler() fiber.Handler {
	return adaptor.HTTPHandler(a.transport.Handler())
}

// RegisterRoutes registers MCP routes with a Fiber app
func (a *Adapter) RegisterRoutes(app *fiber.App) {
	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// MCP endpoint
	app.Post("/mcp", a.Handler())
}
