package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/server/transport"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Adapter provides Gin adapter for MCP server
type Adapter struct {
	transport *transport.HTTPTransport
}

// New creates a new Gin adapter
func New(srv *server.Server, logger types.Logger) *Adapter {
	return &Adapter{
		transport: transport.NewHTTPTransport(srv, logger),
	}
}

// Handler returns a Gin handler function
func (a *Adapter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		a.transport.Handler().ServeHTTP(c.Writer, c.Request)
	}
}

// RegisterRoutes registers MCP routes with a Gin engine
func (a *Adapter) RegisterRoutes(r *gin.Engine) {
	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// MCP endpoint
	r.POST("/mcp", a.Handler())
}
