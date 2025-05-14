package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client represents an HTTP transport client
type Client struct {
	client  *http.Client
	baseURL string
	logger  *zap.Logger
}

// ClientOptions represents client configuration options
type ClientOptions struct {
	// BaseURL for the API
	BaseURL string
	// Timeout for requests
	Timeout time.Duration
	// Logger instance
	Logger *zap.Logger
	// Transport for the HTTP client
	Transport http.RoundTripper
}

// NewClient creates a new HTTP transport client
func NewClient(opts ClientOptions) *Client {
	if opts.Logger == nil {
		opts.Logger, _ = zap.NewProduction()
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: opts.Transport,
	}

	return &Client{
		client:  client,
		baseURL: opts.BaseURL,
		logger:  opts.Logger,
	}
}

// Request represents an HTTP request
type Request struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, req.Path)

	var body io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Set custom headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Log request
	c.logger.Debug("Sending request",
		zap.String("method", req.Method),
		zap.String("url", url),
		zap.Any("headers", httpReq.Header),
	)

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response
	c.logger.Debug("Received response",
		zap.Int("status", resp.StatusCode),
		zap.Int("body_size", len(respBody)),
		zap.Any("headers", resp.Header),
	)

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// Get sends a GET request
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodGet,
		Path:    path,
		Headers: headers,
	})
}

// Post sends a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodPost,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Put sends a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodPut,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Delete sends a DELETE request
func (c *Client) Delete(ctx context.Context, path string, headers map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:  http.MethodDelete,
		Path:    path,
		Headers: headers,
	})
}
