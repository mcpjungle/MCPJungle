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
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var metadata types.ServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
