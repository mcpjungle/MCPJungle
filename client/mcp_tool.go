package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListTools fetches the list of tools, optionally filtered by server name.
// If server is an empty string, this method fetches all tools.
func (c *Client) ListTools(server string) ([]*types.Tool, error) {
	u, _ := c.constructAPIEndpoint("/tools")
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var tools []*types.Tool
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return tools, nil
}

// EnableTools enables a tool or all tools provided by an MCP server.
func (c *Client) EnableTools(name string) ([]string, error) {
	u, _ := c.constructAPIEndpoint("/tools/enable")
	req, err := c.newRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("entity", name)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var tools []string
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	return tools, nil
}

// DisableTools disables a tool or all tools provided by an MCP server.
func (c *Client) DisableTools(name string) ([]string, error) {
	u, _ := c.constructAPIEndpoint("/tools/disable")
	req, err := c.newRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("entity", name)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var tools []string
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	return tools, nil
}

// GetTool fetches a specific tool by its name.
func (c *Client) GetTool(name string) (*types.Tool, error) {
	u, _ := c.constructAPIEndpoint("/tool")
	req, _ := c.newRequest(http.MethodGet, u, nil)
	q := req.URL.Query()
	q.Add("name", name)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var tool types.Tool
	if err := json.NewDecoder(resp.Body).Decode(&tool); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &tool, nil
}

// InvokeTool sends a JSON payload to invoke a tool.
// For now, this function only supports invoking tools that return a string response.
func (c *Client) InvokeTool(name string, input map[string]any) (*types.ToolInvokeResult, error) {
	// We need to insert the tool name into the POST payload
	// In order not to mutate the user-supplied input, create a shallow copy of the input
	// and add the name field to it.
	payload := make(map[string]any, len(input)+1)
	for k, v := range input {
		payload[k] = v
	}
	payload["name"] = name

	body, _ := json.Marshal(payload)
	u, _ := c.constructAPIEndpoint("/tools/invoke")
	req, err := c.newRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to server failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, string(respBody))
	}

	var result *types.ToolInvokeResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
