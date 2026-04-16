package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListResources fetches the list of resources, optionally filtered by server name.
// If server is an empty string, this method fetches all resources.
func (c *Client) ListResources(server string) ([]*types.Resource, error) {
	u, _ := c.constructAPIEndpoint("/resources")
	req, _ := c.newRequest(http.MethodGet, u, nil)
	if server != "" {
		q := req.URL.Query()
		q.Add("server", server)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var resources []*types.Resource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return resources, nil
}

// GetResource retrieves resource metadata by URI.
func (c *Client) GetResource(uri string) (*types.Resource, error) {
	u, err := c.constructAPIEndpoint("/resources/get")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	request := types.ResourceGetRequest{URI: uri}
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

	var resource types.Resource
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resource, nil
}

// ReadResource reads live resource content through MCPJungle.
func (c *Client) ReadResource(uri string) (*types.ResourceReadResult, error) {
	u, err := c.constructAPIEndpoint("/resources/read")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	request := types.ResourceReadRequest{URI: uri}
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
