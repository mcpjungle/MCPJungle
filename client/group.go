package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"io"
	"net/http"
)

// CreateToolGroup send API request to create a new Tool Group.
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var createResp types.CreateToolGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &createResp, nil
}
