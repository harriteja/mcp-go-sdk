package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func TestServer_Initialize(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	srv := New(Options{
		Name:         "test-server",
		Version:      "1.0.0",
		Instructions: "Test server",
		Logger:       logger,
	})

	req := &types.InitializeRequest{
		ProtocolVersion: "1.0",
		ClientInfo: types.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Capabilities: types.ClientCapabilities{},
	}

	resp, err := srv.Initialize(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-server", resp.ServerInfo.Name)
	assert.Equal(t, "1.0.0", resp.ServerInfo.Version)
	assert.Equal(t, "Test server", resp.Instructions)
}

func TestServer_ListTools(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	srv := New(Options{
		Name:    "test-server",
		Version: "1.0.0",
		Logger:  logger,
	})

	// Test without handler
	tools, err := srv.ListTools(context.Background())
	assert.Error(t, err)
	assert.Nil(t, tools)

	// Test with handler
	expectedTools := []types.Tool{
		{
			Name:        "test-tool",
			Description: "A test tool",
		},
	}

	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return expectedTools, nil
	})

	tools, err = srv.ListTools(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedTools, tools)
}

func TestServer_CallTool(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	srv := New(Options{
		Name:    "test-server",
		Version: "1.0.0",
		Logger:  logger,
	})

	// Test without handler
	result, err := srv.CallTool(context.Background(), "test-tool", nil)
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test with handler
	expectedResult := map[string]interface{}{
		"result": "success",
	}

	srv.OnCallTool(func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
		assert.Equal(t, "test-tool", name)
		return expectedResult, nil
	})

	result, err = srv.CallTool(context.Background(), "test-tool", nil)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestServer_GetSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	srv := New(Options{
		Name:    "test-server",
		Version: "1.0.0",
		Logger:  logger,
	})

	// Test non-existent session
	session, err := srv.getSession("non-existent")
	assert.Error(t, err)
	assert.Nil(t, session)

	// Test existing session
	req := &types.InitializeRequest{
		ProtocolVersion: "1.0",
		ClientInfo: types.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Capabilities: types.ClientCapabilities{},
	}

	resp, err := srv.Initialize(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Get session from map
	sessions := make([]string, 0)
	srv.mu.RLock()
	for id := range srv.sessions {
		sessions = append(sessions, id)
	}
	srv.mu.RUnlock()

	assert.Len(t, sessions, 1)
	session, err = srv.getSession(sessions[0])
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "test-client", session.ClientInfo().Name)
}
