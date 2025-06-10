package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
)

func TestClientCreation(t *testing.T) {
	// Create client
	client, err := NewClient(ClientOptions{
		URL:            "ws://localhost:12345", // Invalid URL to avoid actual connection
		ConnectTimeout: time.Second,
		Logger:         logger.NewNopLogger(),
	})
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Test basic properties
	assert.Equal(t, "ws://localhost:12345", client.url.String())
	assert.Equal(t, time.Second, client.connectTimeout)

	// Test connect failure (expected since URL is invalid)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = client.Connect(ctx)
	require.Error(t, err)
}
