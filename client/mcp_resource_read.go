package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// GetResource retrieves resource metadata by URI, optionally scoped to a server.
func (c *Client) GetResource(uri string, server string) (*types.Resource, error) {
	u, err := c.constructAPIEndpoint("/resource")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	parsed, _ := url.Parse(u)
	q := parsed.Query()
	q.Set("uri", uri)
	if server != "" {
		q.Set("server", server)
	}
	parsed.RawQuery = q.Encode()
	u = parsed.String()

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var resource types.Resource
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resource, nil
}

// ReadResource reads live resource content through MCPJungle, optionally scoped to a server.
func (c *Client) ReadResource(uri string, server string) (*types.ResourceReadResult, error) {
	u, err := c.constructAPIEndpoint("/resources/read")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	request := types.ResourceReadRequest{
		URI:    uri,
		Server: server,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, u, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var result types.ResourceReadResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
