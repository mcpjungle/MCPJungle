// Package client provides HTTP client functionality for interacting with the MCPJungle API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/internal/api"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// Client represents a client for interacting with the MCPJungle HTTP API
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

func NewClient(baseURL string, accessToken string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient:  httpClient,
	}
}

// BaseURL returns the base URL of the MCPJungle server
func (c *Client) BaseURL() string {
	return c.baseURL
}

// constructAPIEndpoint constructs the full API endpoint URL where a request must be sent
func (c *Client) constructAPIEndpoint(suffixPath string) (string, error) {
	return url.JoinPath(c.baseURL, api.V0ApiPathPrefix, suffixPath)
}

// newRequest creates a new HTTP request with the specified method, URL, and body.
// It automatically adds the Authorization header if an access token is present.
func (c *Client) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	return req, nil
}

// ErrorResponse represents the JSON structure of error responses from the server
type ErrorResponse struct {
	Error string `json:"error"`
}

// parseErrorResponse parses HTTP error responses (4xx and 5xx) and returns a user-friendly error message
func (c *Client) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("request failed with status: %d (unable to read error details)", resp.StatusCode)
	}

	// For 4xx and 5xx status codes, try to parse as JSON error response
	if resp.StatusCode >= 400 && resp.StatusCode < 600 {
		var errorResp ErrorResponse
		err := json.Unmarshal(body, &errorResp)
		if err != nil || errorResp.Error == "" {
			// If parsing as JSON fails or the error message is empty, return the raw response
			return fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, string(body))
		}
		// Return the parsed error message
		return fmt.Errorf("%s", errorResp.Error)
	}

	// For any other status code, return the full response
	return fmt.Errorf("unexpected response with status: %d, body: %s", resp.StatusCode, string(body))
}

// GetServerMetadata fetches metadata about the MCPJungle server.
func (c *Client) GetServerMetadata(ctx context.Context) (*types.ServerMetadata, error) {
	req, err := c.newRequest(http.MethodGet, c.baseURL+"/metadata", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var metadata types.ServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
