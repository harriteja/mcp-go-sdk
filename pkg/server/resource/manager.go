package resource

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// ResourceHandler handles resource operations
type ResourceHandler interface {
	// Read reads a resource by URI
	Read(ctx context.Context, uri string) ([]byte, string, error)
}

// TemplateHandler handles resource template operations
type TemplateHandler interface {
	// List returns available resource templates
	List(ctx context.Context) ([]types.ResourceTemplate, error)
}

// CacheEntry represents a cached resource
type CacheEntry struct {
	Data       []byte
	MimeType   string
	ExpiresAt  time.Time
	LastAccess time.Time
}

// Manager manages resources and templates with caching
type Manager struct {
	mu sync.RWMutex

	// Handlers
	resourceHandler ResourceHandler
	templateHandler TemplateHandler

	// Cache settings
	cacheEnabled bool
	cacheTTL     time.Duration
	maxCacheSize int64
	currentSize  int64

	// Cache storage
	cache map[string]*CacheEntry

	logger *zap.Logger
}

// Options represents resource manager options
type Options struct {
	ResourceHandler ResourceHandler
	TemplateHandler TemplateHandler
	CacheEnabled    bool
	CacheTTL        time.Duration
	MaxCacheSize    int64
	Logger          *zap.Logger
}

// New creates a new resource manager
func New(opts Options) *Manager {
	if opts.Logger == nil {
		opts.Logger, _ = zap.NewDevelopment()
	}
	if opts.CacheTTL == 0 {
		opts.CacheTTL = 5 * time.Minute
	}
	if opts.MaxCacheSize == 0 {
		opts.MaxCacheSize = 100 * 1024 * 1024 // 100MB default
	}

	return &Manager{
		resourceHandler: opts.ResourceHandler,
		templateHandler: opts.TemplateHandler,
		cacheEnabled:    opts.CacheEnabled,
		cacheTTL:        opts.CacheTTL,
		maxCacheSize:    opts.MaxCacheSize,
		cache:           make(map[string]*CacheEntry),
		logger:          opts.Logger,
	}
}

// Read reads a resource, using cache if enabled
func (m *Manager) Read(ctx context.Context, uri string) ([]byte, string, error) {
	if !m.cacheEnabled {
		return m.resourceHandler.Read(ctx, uri)
	}

	// Check cache
	m.mu.RLock()
	entry, exists := m.cache[uri]
	m.mu.RUnlock()

	if exists {
		if time.Now().Before(entry.ExpiresAt) {
			// Update last access time
			m.mu.Lock()
			entry.LastAccess = time.Now()
			m.mu.Unlock()
			return entry.Data, entry.MimeType, nil
		}
		// Remove expired entry
		m.mu.Lock()
		delete(m.cache, uri)
		m.currentSize -= int64(len(entry.Data))
		m.mu.Unlock()
	}

	// Read from handler
	data, mimeType, err := m.resourceHandler.Read(ctx, uri)
	if err != nil {
		return nil, "", err
	}

	// Add to cache
	if m.cacheEnabled {
		m.addToCache(uri, data, mimeType)
	}

	return data, mimeType, nil
}

// ListTemplates returns available resource templates
func (m *Manager) ListTemplates(ctx context.Context) ([]types.ResourceTemplate, error) {
	if m.templateHandler == nil {
		return nil, errors.New("template handler not registered")
	}
	return m.templateHandler.List(ctx)
}

func (m *Manager) addToCache(uri string, data []byte, mimeType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dataSize := int64(len(data))

	// Check if adding this entry would exceed max cache size
	if m.currentSize+dataSize > m.maxCacheSize {
		m.evictLRU(dataSize)
	}

	// Add new entry
	m.cache[uri] = &CacheEntry{
		Data:       data,
		MimeType:   mimeType,
		ExpiresAt:  time.Now().Add(m.cacheTTL),
		LastAccess: time.Now(),
	}
	m.currentSize += dataSize
}

func (m *Manager) evictLRU(requiredSize int64) {
	type cacheItem struct {
		uri   string
		entry *CacheEntry
	}

	// Get all entries
	entries := make([]cacheItem, 0, len(m.cache))
	for uri, entry := range m.cache {
		entries = append(entries, cacheItem{uri, entry})
	}

	// Sort by last access time (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].entry.LastAccess.After(entries[j].entry.LastAccess) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove entries until we have enough space
	for _, item := range entries {
		if m.currentSize+requiredSize <= m.maxCacheSize {
			break
		}
		delete(m.cache, item.uri)
		m.currentSize -= int64(len(item.entry.Data))
		m.logger.Debug("Evicted resource from cache",
			zap.String("uri", item.uri),
			zap.Time("lastAccess", item.entry.LastAccess))
	}
}
