package resource

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// mockResourceHandler implements ResourceHandler for testing
type mockResourceHandler struct {
	data     []byte
	mimeType string
	err      error
}

func (m *mockResourceHandler) Read(ctx context.Context, uri string) ([]byte, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.data, m.mimeType, nil
}

// mockTemplateHandler implements TemplateHandler for testing
type mockTemplateHandler struct {
	templates []types.ResourceTemplate
	err       error
}

func (m *mockTemplateHandler) List(ctx context.Context) ([]types.ResourceTemplate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.templates, nil
}

func TestManager_Read(t *testing.T) {
	// Create mock handler
	handler := &mockResourceHandler{
		data:     []byte("test data"),
		mimeType: "text/plain",
	}

	// Create manager with cache enabled
	mgr := New(Options{
		ResourceHandler: handler,
		CacheEnabled:    true,
		CacheTTL:        time.Second,
		Logger:          zap.NewNop(),
	})

	// First read should hit the handler
	data, mimeType, err := mgr.Read(context.Background(), "test://uri")
	assert.NoError(t, err)
	assert.Equal(t, handler.data, data)
	assert.Equal(t, handler.mimeType, mimeType)

	// Second read should hit the cache
	data, mimeType, err = mgr.Read(context.Background(), "test://uri")
	assert.NoError(t, err)
	assert.Equal(t, handler.data, data)
	assert.Equal(t, handler.mimeType, mimeType)

	// Wait for cache to expire
	time.Sleep(time.Second)

	// Third read should hit the handler again
	data, mimeType, err = mgr.Read(context.Background(), "test://uri")
	assert.NoError(t, err)
	assert.Equal(t, handler.data, data)
	assert.Equal(t, handler.mimeType, mimeType)
}

func TestManager_CacheEviction(t *testing.T) {
	// Create mock handler
	handler := &mockResourceHandler{
		data:     []byte("test data"),
		mimeType: "text/plain",
	}

	// Create manager with small cache size
	mgr := New(Options{
		ResourceHandler: handler,
		CacheEnabled:    true,
		MaxCacheSize:    20, // Only enough for 2 entries
		Logger:          zap.NewNop(),
	})

	// Fill cache
	uris := []string{"test://1", "test://2", "test://3"}
	for _, uri := range uris {
		_, _, err := mgr.Read(context.Background(), uri)
		assert.NoError(t, err)
	}

	// Check that oldest entry was evicted
	mgr.mu.RLock()
	_, exists := mgr.cache["test://1"]
	mgr.mu.RUnlock()
	assert.False(t, exists, "oldest entry should have been evicted")

	// Check that newest entries are still cached
	mgr.mu.RLock()
	_, exists = mgr.cache["test://2"]
	assert.True(t, exists, "second entry should still be cached")
	_, exists = mgr.cache["test://3"]
	assert.True(t, exists, "newest entry should be cached")
	mgr.mu.RUnlock()
}

func TestManager_ListTemplates(t *testing.T) {
	// Create mock handler
	handler := &mockTemplateHandler{
		templates: []types.ResourceTemplate{
			{
				URITemplate: "test://{name}",
				Name:        "test",
			},
		},
	}

	// Create manager
	mgr := New(Options{
		TemplateHandler: handler,
		Logger:          zap.NewNop(),
	})

	// List templates
	templates, err := mgr.ListTemplates(context.Background())
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "test://{name}", templates[0].URITemplate)
}

func TestManager_NoCache(t *testing.T) {
	// Create mock handler
	handler := &mockResourceHandler{
		data:     []byte("test data"),
		mimeType: "text/plain",
	}

	// Create manager with cache disabled
	mgr := New(Options{
		ResourceHandler: handler,
		CacheEnabled:    false,
		Logger:          zap.NewNop(),
	})

	// Multiple reads should always hit the handler
	for i := 0; i < 3; i++ {
		data, mimeType, err := mgr.Read(context.Background(), "test://uri")
		assert.NoError(t, err)
		assert.Equal(t, handler.data, data)
		assert.Equal(t, handler.mimeType, mimeType)
	}

	// Cache should be empty
	mgr.mu.RLock()
	assert.Empty(t, mgr.cache)
	mgr.mu.RUnlock()
}
