package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// Options represents client configuration options
type Options struct {
	// ServerURL is the URL of the MCP server (for HTTP transport)
	ServerURL string

	// Reader is the reader for stdio transport
	Reader io.Reader

	// Writer is the writer for stdio transport
	Writer io.Writer

	// ClientInfo contains client implementation details
	ClientInfo types.Implementation

	// HTTPClient is the HTTP client to use (for HTTP transport)
	HTTPClient *http.Client
}

// Client represents an MCP client
type Client struct {
	serverURL    string
	clientInfo   types.Implementation
	httpClient   *http.Client
	serverInfo   *types.Implementation
	capabilities *types.ServerCapabilities

	// stdio transport
	reader io.Reader
	writer io.Writer
}

// New creates a new MCP client
func New(opts Options) *Client {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		serverURL:  opts.ServerURL,
		clientInfo: opts.ClientInfo,
		httpClient: httpClient,
		reader:     opts.Reader,
		writer:     opts.Writer,
	}
}

// Initialize initializes the client with the server
func (c *Client) Initialize(ctx context.Context) error {
	req := types.InitializeRequest{
		ProtocolVersion: "1.0",
		ClientInfo:      c.clientInfo,
		Capabilities:    types.ClientCapabilities{},
	}

	var resp types.InitializeResponse
	if err := c.call(ctx, "initialize", req, &resp); err != nil {
		return errors.Wrap(err, "failed to initialize")
	}

	c.serverInfo = &resp.ServerInfo
	c.capabilities = &resp.Capabilities
	return nil
}

// Initialized notifies the server that the client has completed initialization
func (c *Client) Initialized(ctx context.Context) error {
	// The initialized notification has an empty payload
	notification := types.InitializedNotification{}

	// This is a notification, not a request, so we don't expect a response
	if err := c.call(ctx, "initialized", notification, nil); err != nil {
		return errors.Wrap(err, "failed to send initialized notification")
	}

	return nil
}

// Ping sends a ping request to test server connectivity
func (c *Client) Ping(ctx context.Context) (*types.PingResponse, error) {
	req := types.PingRequest{
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	var resp types.PingResponse
	if err := c.call(ctx, "ping", req, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to ping server")
	}

	return &resp, nil
}

// Cancel sends a request to cancel an ongoing operation
func (c *Client) Cancel(ctx context.Context, requestID string) error {
	req := types.CancelRequest{
		ID: requestID,
	}

	// This is a notification, not a request, so we don't expect a response
	if err := c.call(ctx, "cancel", req, nil); err != nil {
		return errors.Wrap(err, "failed to send cancellation request")
	}

	return nil
}

// ListTools lists available tools from the server
func (c *Client) ListTools(ctx context.Context) ([]types.Tool, error) {
	var tools []types.Tool
	if err := c.call(ctx, "listTools", nil, &tools); err != nil {
		return nil, errors.Wrap(err, "failed to list tools")
	}
	return tools, nil
}

// CallTool calls a tool on the server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	req := struct {
		Name string                 `json:"name"`
		Args map[string]interface{} `json:"args"`
	}{
		Name: name,
		Args: args,
	}

	var result interface{}
	if err := c.call(ctx, "callTool", req, &result); err != nil {
		return nil, errors.Wrap(err, "failed to call tool")
	}
	return result, nil
}

// ListPrompts lists available prompts from the server
func (c *Client) ListPrompts(ctx context.Context) ([]types.Prompt, error) {
	var prompts []types.Prompt
	if err := c.call(ctx, "listPrompts", nil, &prompts); err != nil {
		return nil, errors.Wrap(err, "failed to list prompts")
	}
	return prompts, nil
}

// GetPrompt gets a prompt from the server
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (*types.Prompt, error) {
	req := struct {
		Name string                 `json:"name"`
		Args map[string]interface{} `json:"args"`
	}{
		Name: name,
		Args: args,
	}

	var prompt types.Prompt
	if err := c.call(ctx, "getPrompt", req, &prompt); err != nil {
		return nil, errors.Wrap(err, "failed to get prompt")
	}
	return &prompt, nil
}

// ListResources lists available resources from the server
func (c *Client) ListResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	if err := c.call(ctx, "listResources", nil, &resources); err != nil {
		return nil, errors.Wrap(err, "failed to list resources")
	}
	return resources, nil
}

// ReadResource reads a resource from the server
func (c *Client) ReadResource(ctx context.Context, uri string) ([]byte, string, error) {
	req := struct {
		URI string `json:"uri"`
	}{
		URI: uri,
	}

	if c.reader != nil && c.writer != nil {
		// Use stdio transport
		var result struct {
			Data     []byte `json:"data"`
			MimeType string `json:"mimeType"`
		}
		if err := c.call(ctx, "readResource", req, &result); err != nil {
			return nil, "", err
		}
		return result.Data, result.MimeType, nil
	}

	// Use HTTP transport
	httpReq, err := c.newRequest(ctx, "readResource", req)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to create request")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var mcpErr types.Error
		if err := json.NewDecoder(resp.Body).Decode(&mcpErr); err != nil {
			return nil, "", errors.Wrap(err, "failed to decode error response")
		}
		return nil, "", &mcpErr
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to read response body")
	}

	return data, resp.Header.Get("Content-Type"), nil
}

// ListResourceTemplates lists available resource templates from the server
func (c *Client) ListResourceTemplates(ctx context.Context) ([]types.ResourceTemplate, error) {
	var templates []types.ResourceTemplate
	if err := c.call(ctx, "listResourceTemplates", nil, &templates); err != nil {
		return nil, errors.Wrap(err, "failed to list resource templates")
	}
	return templates, nil
}

// call makes an RPC call to the server
func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	if c.reader != nil && c.writer != nil {
		return c.callStdio(method, params, result)
	}
	return c.callHTTP(ctx, method, params, result)
}

// callStdio makes an RPC call using stdio transport
func (c *Client) callStdio(method string, params interface{}, result interface{}) error {
	// Write request
	req := struct {
		Method string      `json:"method"`
		Params interface{} `json:"params,omitempty"`
	}{
		Method: method,
		Params: params,
	}

	if err := json.NewEncoder(c.writer).Encode(req); err != nil {
		return errors.Wrap(err, "failed to encode request")
	}

	// Read response
	var rawResp json.RawMessage
	if err := json.NewDecoder(c.reader).Decode(&rawResp); err != nil {
		if err == io.EOF {
			// If we get EOF during initialize, it's an error
			if method == "initialize" {
				return errors.Wrap(err, "connection closed during initialize")
			}
			// For other methods, EOF after writing request means no response
			return nil
		}
		return errors.Wrap(err, "failed to decode response")
	}

	// Try to decode as error response first
	var errResp struct {
		Error *types.Error `json:"error"`
	}
	if err := json.Unmarshal(rawResp, &errResp); err == nil && errResp.Error != nil {
		return errResp.Error
	}

	// If no error, decode as result
	var resultResp struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(rawResp, &resultResp); err != nil {
		return errors.Wrap(err, "failed to decode response")
	}

	// If no result expected, return
	if result == nil {
		return nil
	}

	// Decode result
	if err := json.Unmarshal(resultResp.Result, result); err != nil {
		return errors.Wrap(err, "failed to decode result")
	}

	return nil
}

// callHTTP makes an HTTP request and decodes the response
func (c *Client) callHTTP(ctx context.Context, method string, params interface{}, result interface{}) error {
	// Create request body
	var body io.Reader
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return errors.Wrap(err, "failed to marshal request body")
		}
		body = bytes.NewReader(data)
	}

	// Ensure server URL has trailing slash
	serverURL := strings.TrimSuffix(c.serverURL, "/") + "/"

	// Create request URL
	url := serverURL + method
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}

	// Debug: Print response body
	fmt.Printf("Response body: %s\n", string(respBody))

	// Try to decode as error response first
	var errResp struct {
		Error *types.Error `json:"error"`
	}
	if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != nil {
		// Return the error directly to preserve type information
		return errResp.Error
	}

	// If no error, decode as result
	var resultResp struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &resultResp); err != nil {
		return errors.Wrap(err, "failed to decode response")
	}

	// If result is nil, we don't need to decode further
	if result == nil {
		return nil
	}

	// Decode result into target type
	if err := json.Unmarshal(resultResp.Result, result); err != nil {
		return errors.Wrap(err, "failed to decode response")
	}

	return nil
}

// newRequest creates a new HTTP request
func (c *Client) newRequest(ctx context.Context, method string, params interface{}) (*http.Request, error) {
	body := struct {
		Method string      `json:"method"`
		Params interface{} `json:"params,omitempty"`
	}{
		Method: method,
		Params: params,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request body")
	}

	// Ensure server URL has trailing slash
	serverURL := strings.TrimSuffix(c.serverURL, "/") + "/"

	// Create request URL
	url := serverURL + method
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
