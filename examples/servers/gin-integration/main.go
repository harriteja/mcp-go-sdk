package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	mcpLogger "github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server"
	transportgin "github.com/harriteja/mcp-go-sdk/pkg/server/transport/gin"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// MultiplyRequest is a request to the multiply tool
type MultiplyRequest struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

// MultiplyResponse is a response from the multiply tool
type MultiplyResponse struct {
	Result float64 `json:"result"`
}

func main() {
	// Set Gin to release mode in production
	// gin.SetMode(gin.ReleaseMode)

	// Create a new Gin router - this is your existing Gin application
	router := gin.Default()

	// Add middleware
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Set up your existing Gin routes
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to the Gin + MCP integration example!")
	})

	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now(),
		})
	})

	// API group for your existing endpoints
	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, []gin.H{
				{"id": 1, "name": "User 1"},
				{"id": 2, "name": "User 2"},
				{"id": 3, "name": "User 3"},
			})
		})

		apiGroup.GET("/products", func(c *gin.Context) {
			c.JSON(http.StatusOK, []gin.H{
				{"id": 1, "name": "Product 1", "price": 99.99},
				{"id": 2, "name": "Product 2", "price": 149.99},
				{"id": 3, "name": "Product 3", "price": 199.99},
			})
		})
	}

	// Create a new MCP server
	mcpServer, err := server.New(&server.Options{
		Name:    "gin-mcp-example",
		Version: "1.0.0",
		Logger:  mcpLogger.GetDefaultLogger(),
	})
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Register MCP tools
	mcpServer.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "multiply",
				Description: "Multiply two numbers",
				Parameters: &types.Parameters{
					Type: "object",
					Properties: map[string]types.Parameter{
						"a": {
							Type:        "number",
							Description: "First number",
						},
						"b": {
							Type:        "number",
							Description: "Second number",
						},
					},
				},
			},
		}, nil
	})

	mcpServer.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "multiply":
			// Parse arguments
			a, ok := args["a"].(float64)
			if !ok {
				return nil, fmt.Errorf("invalid argument 'a'")
			}
			b, ok := args["b"].(float64)
			if !ok {
				return nil, fmt.Errorf("invalid argument 'b'")
			}

			// Perform calculation
			result := a * b

			// Return response
			return MultiplyResponse{Result: result}, nil
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	})

	// Create a Gin adapter for the MCP server
	ginAdapter := transportgin.New(mcpServer, mcpLogger.GetDefaultLogger())

	// Register MCP routes with your existing Gin router
	// This is the key part - adding MCP routes to your existing app
	ginAdapter.RegisterRoutes(router)

	// Start the server
	log.Println("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
