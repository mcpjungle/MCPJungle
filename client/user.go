package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"io"
	"net/http"
	"net/url"
)

// CreateUser sends a request to create a new authenticated, human user in mcpjungle
func (c *Client) CreateUser(user *types.User) (*CreateUserResponse, error) {
	u, _ := url.JoinPath(c.baseURL, "/users")

	body, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewBuffer(body))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var createResp CreateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &createResp, nil
}

// DeleteUser sends a request to delete a user from mcpjungle
func (c *Client) DeleteUser(username string) error {
	u, _ := url.JoinPath(c.baseURL, "/users", username)

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to %s: %w", u, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	return nil
}
