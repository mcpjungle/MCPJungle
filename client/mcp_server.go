package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// RegisterServer registers a new MCP server with the registry.
func (c *Client) RegisterServer(server *types.RegisterServerInput, force bool) (*types.RegisterServerResult, error) {
	u, _ := c.constructAPIEndpoint("/servers")
	if force {
		parsedURL, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("failed to parse server registration endpoint: %w", err)
		}
		q := parsedURL.Query()
		q.Set("force", "true")
		parsedURL.RawQuery = q.Encode()
		u = parsedURL.String()
	}

	body, err := json.Marshal(server)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize server data into JSON: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, c.parseErrorResponse(resp)
	}

	var result types.RegisterServerResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// CompleteUpstreamOAuthSession finalizes a pending upstream OAuth registration
// after the operator has approved the authorization request in the browser.
func (c *Client) CompleteUpstreamOAuthSession(sessionID string, input *types.CompleteUpstreamOAuthSessionInput) (*types.RegisterServerResult, error) {
	u, err := c.constructAPIEndpoint("/upstream_oauth/sessions/" + sessionID + "/complete")
	if err != nil {
		return nil, fmt.Errorf("failed to construct completion endpoint: %w", err)
	}

	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize OAuth completion payload: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseErrorResponse(resp)
	}

	var result types.RegisterServerResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListServers fetches the list of registered servers.
func (c *Client) ListServers() ([]*types.McpServer, error) {
	u, _ := c.constructAPIEndpoint("/servers")
	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var servers []*types.McpServer
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return servers, nil
}

// GetServerConfigs returns the configurations of all registered MCP servers.
// This is different from ListServers() because it returns the complete configuration used to register the servers.
// This config can be used to register the servers again elsewhere.
func (c *Client) GetServerConfigs() ([]*types.RegisterServerInput, error) {
	u, _ := c.constructAPIEndpoint("/server_configs")
	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var serverConfigs []*types.RegisterServerInput
	if err := json.NewDecoder(resp.Body).Decode(&serverConfigs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return serverConfigs, nil
}

// DeregisterServer deletes a server by name.
func (c *Client) DeregisterServer(name string) error {
	u, _ := c.constructAPIEndpoint("/servers/" + name)
	req, _ := c.newRequest(http.MethodDelete, u, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseErrorResponse(resp)
	}
	return nil
}

// EnableServer sends API request to enable a server by name.
func (c *Client) EnableServer(name string) (*types.EnableDisableServerResult, error) {
	return c.setServerEnabled(name, true)
}

// DisableServer sends API request to disable a server by name.
func (c *Client) DisableServer(name string) (*types.EnableDisableServerResult, error) {
	return c.setServerEnabled(name, false)
}

func (c *Client) setServerEnabled(name string, enabled bool) (*types.EnableDisableServerResult, error) {
	api := "enable"
	if !enabled {
		api = "disable"
	}
	endpoint := fmt.Sprintf("/servers/%s/%s", name, api)

	u, err := c.constructAPIEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	req, err := c.newRequest(http.MethodPost, u, nil)
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

	var result types.EnableDisableServerResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
