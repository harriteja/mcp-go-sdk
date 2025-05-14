package stdio

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/server"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

func TestTransport_Initialize(t *testing.T) {
	// Create test server
	srv := server.New(server.Options{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Create test pipes
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()

	// Create transport
	transport := New(srv, Options{
		Reader: inputReader,
		Writer: outputWriter,
		Logger: zap.NewNop(),
	})

	// Start transport in goroutine
	done := make(chan error)
	go func() {
		done <- transport.Start()
	}()

	// Write initialize request
	req := struct {
		Method string                  `json:"method"`
		Params types.InitializeRequest `json:"params"`
	}{
		Method: "initialize",
		Params: types.InitializeRequest{
			ProtocolVersion: "1.0",
			ClientInfo: types.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		},
	}
	if err := json.NewEncoder(inputWriter).Encode(req); err != nil {
		t.Fatal(err)
	}

	// Read response
	var resp struct {
		Result types.InitializeResponse `json:"result"`
		Error  *types.Error             `json:"error,omitempty"`
	}
	if err := json.NewDecoder(outputReader).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Verify response
	assert.Nil(t, resp.Error)
	assert.Equal(t, "test-server", resp.Result.ServerInfo.Name)
	assert.Equal(t, "1.0.0", resp.Result.ServerInfo.Version)

	// Close input to signal EOF
	inputWriter.Close()

	// Wait for transport to stop
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("transport did not stop")
	}

	// Clean up
	outputReader.Close()
	outputWriter.Close()
}

func TestTransport_ListTools(t *testing.T) {
	// Create test server
	srv := server.New(server.Options{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Register test tool
	srv.OnListTools(func(ctx context.Context) ([]types.Tool, error) {
		return []types.Tool{
			{
				Name:        "test-tool",
				Description: "A test tool",
			},
		}, nil
	})

	// Create test pipes
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()

	// Create transport
	transport := New(srv, Options{
		Reader: inputReader,
		Writer: outputWriter,
		Logger: zap.NewNop(),
	})

	// Start transport in goroutine
	done := make(chan error)
	go func() {
		done <- transport.Start()
	}()

	// Write list tools request
	req := struct {
		Method string `json:"method"`
	}{
		Method: "listTools",
	}
	if err := json.NewEncoder(inputWriter).Encode(req); err != nil {
		t.Fatal(err)
	}

	// Read response
	var resp struct {
		Result []types.Tool `json:"result"`
		Error  *types.Error `json:"error,omitempty"`
	}
	if err := json.NewDecoder(outputReader).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Verify response
	assert.Nil(t, resp.Error)
	assert.Len(t, resp.Result, 1)
	assert.Equal(t, "test-tool", resp.Result[0].Name)
	assert.Equal(t, "A test tool", resp.Result[0].Description)

	// Close input to signal EOF
	inputWriter.Close()

	// Wait for transport to stop
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("transport did not stop")
	}

	// Clean up
	outputReader.Close()
	outputWriter.Close()
}

func TestTransport_Error(t *testing.T) {
	// Create test server
	srv := server.New(server.Options{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Create test pipes
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()

	// Create transport
	transport := New(srv, Options{
		Reader: inputReader,
		Writer: outputWriter,
		Logger: zap.NewNop(),
	})

	// Start transport in goroutine
	done := make(chan error)
	go func() {
		done <- transport.Start()
	}()

	// Write invalid request
	if _, err := inputWriter.Write([]byte("invalid json\n")); err != nil {
		t.Fatalf("failed to write data: %v", err)
	}

	// Read error response
	var resp struct {
		Result interface{}  `json:"result,omitempty"`
		Error  *types.Error `json:"error"`
	}
	if err := json.NewDecoder(outputReader).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Verify error
	assert.NotNil(t, resp.Error)
	assert.Equal(t, 500, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "failed to parse request")

	// Close input to signal EOF
	inputWriter.Close()

	// Wait for transport to stop
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("transport did not stop")
	}

	// Clean up
	outputReader.Close()
	outputWriter.Close()
}
