package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// CreateUser sends a request to create a new authenticated, human user in mcpjungle
func (c *Client) CreateUser(user *types.CreateUserRequest) (*types.CreateUserResponse, error) {
	u, _ := c.constructAPIEndpoint("/users")

	body, err := json.Marshal(user)
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

	var createResp types.CreateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &createResp, nil
}

// DeleteUser sends a request to delete a user from mcpjungle
func (c *Client) DeleteUser(username string) error {
	u, _ := c.constructAPIEndpoint("/users/" + username)

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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	return nil
}

// ListUsers sends a request to list all users in mcpjungle
func (c *Client) ListUsers() ([]*types.User, error) {
	u, _ := c.constructAPIEndpoint("/users")

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var users []*types.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return users, nil
}

// Whoami sends a request to get information about the user associated with the provided access token
func (c *Client) Whoami(accessToken string) (*types.User, error) {
	u, _ := c.constructAPIEndpoint("/users/whoami")

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &user, nil
}
