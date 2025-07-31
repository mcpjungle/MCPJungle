package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Server represents an MCP server registered in the MCPJungle registry.
type Server struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// RegisterServerInput is the input structure for registering a new MCP server.

type RegisterServerInput struct {
	// Name is the unique name of an MCP server registered in mcpjungle
	Name string `json:"name"`

	// Transport is the transport protocol used by the MCP server.
	// valid values are "stdio", "streamable_http""
	Transport string `json:"transport"`

	Description string `json:"description"`

	// URL is the URL of the remote mcp server
	// It is mandatory when transport is streamable_http and must be a valid
	//  http/https URL (e.g., https://example.com/mcp).
	URL string `json:"url"`

	// BearerToken is an optional token used for authenticating requests to the remote MCP server.
	// It is useful when the upstream MCP server requires static tokens (e.g., API tokens) for authentication.
	// If the transport is "stdio", this field is ignored.
	BearerToken string `json:"bearer_token"`

	// Command is the command to run the mcp server.
	// It is mandatory when the transport is "stdio".
	Command string `json:"command"`

	// Args is the list of arguments to pass to the command when the transport is "stdio".
	Args []string `json:"args"`

	// Env is the set of environment variables to pass to the mcp server when the transport is "stdio".
	// Both the key and value must be of type string.
	Env map[string]string `json:"env"`
}

// RegisterServer registers a new MCP server with the registry.
func (c *Client) RegisterServer(server *RegisterServerInput) (*Server, error) {
	u, _ := c.constructAPIEndpoint("/servers")
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

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var registeredServer Server
	if err := json.NewDecoder(resp.Body).Decode(&registeredServer); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &registeredServer, nil
}

// ListServers fetches the list of registered servers.
func (c *Client) ListServers() ([]*Server, error) {
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var servers []*Server
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return servers, nil
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status from server: %s, body: %s", resp.Status, body)
	}
	return nil
}
