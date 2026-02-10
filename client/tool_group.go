package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// CreateToolGroup sends API request to create a new Tool Group.
func (c *Client) CreateToolGroup(group *types.ToolGroup) (*types.CreateToolGroupResponse, error) {
	u, _ := c.constructAPIEndpoint("/tool-groups")

	body, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(http.MethodPost, u, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
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

	var createResp types.CreateToolGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &createResp, nil
}

// DeleteToolGroup sends API request to delete a Tool Group by name.
func (c *Client) DeleteToolGroup(name string) error {
	u, _ := c.constructAPIEndpoint("/tool-groups/" + name)

	req, err := c.newRequest(http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to %s: %w", u, err)
	}

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

// ListToolGroups sends API request to list all Tool Groups.
func (c *Client) ListToolGroups() ([]types.ToolGroup, error) {
	u, _ := c.constructAPIEndpoint("/tool-groups")

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var groups []types.ToolGroup
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return groups, nil
}

// GetToolGroup sends API request to get details of a specific Tool Group by name.
func (c *Client) GetToolGroup(name string) (*types.GetToolGroupResponse, error) {
	u, _ := c.constructAPIEndpoint("/tool-groups/" + name)

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var group types.GetToolGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &group, nil
}

// GetToolGroupEffectiveTools sends API request to get effective tools of a specific Tool Group by name.
func (c *Client) GetToolGroupEffectiveTools(name string) ([]string, error) {
	u, _ := c.constructAPIEndpoint("/tool-groups/" + name + "/effective-tools")

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var out struct {
		Tools []string `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return out.Tools, nil
}

func (c *Client) UpdateToolGroup(group *types.ToolGroup) (*types.UpdateToolGroupResponse, error) {
	u, _ := c.constructAPIEndpoint("/tool-groups/" + group.Name)

	body, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(http.MethodPut, u, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var updateResp types.UpdateToolGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &updateResp, nil
}

// GetToolGroupConfigs returns all Tool Group configurations.
// It is just a user-friendly wrapper around ListToolGroups().
func (c *Client) GetToolGroupConfigs() ([]types.ToolGroup, error) {
	return c.ListToolGroups()
}
