package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// CreateUser sends a request to create a new authenticated, human user in mcpjungle
func (c *Client) CreateUser(user *types.CreateOrUpdateUserRequest) (*types.CreateOrUpdateUserResponse, error) {
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
		return nil, c.parseErrorResponse(resp)
	}

	var createResp types.CreateOrUpdateUserResponse
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
		return c.parseErrorResponse(resp)
	}

	return nil
}

func (c *Client) UpdateUser(user *types.CreateOrUpdateUserRequest) (*types.CreateOrUpdateUserResponse, error) {
	u, _ := c.constructAPIEndpoint("/users/" + user.Username)

	body, err := json.Marshal(user)
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

	var updateResp types.CreateOrUpdateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &updateResp, nil
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
		return nil, c.parseErrorResponse(resp)
	}

	var users []*types.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return users, nil
}

// GetUserConfigs returns the configurations of all users.
// The returned configs can be used to recreate the users elsewhere.
// Access tokens are not included as the list endpoint does not return them.
func (c *Client) GetUserConfigs() ([]types.UserConfig, error) {
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}

	configs := make([]types.UserConfig, 0, len(users))
	for _, u := range users {
		// Populate access_token_ref with a hint so the operator knows to supply the
		// token via environment variable before reimporting.
		envVarHint := "MCPJUNGLE_USER_TOKEN_" + strings.ToUpper(strings.ReplaceAll(u.Username, "-", "_"))
		configs = append(configs, types.UserConfig{
			Username:       u.Username,
			AccessTokenRef: types.AccessTokenRef{Env: envVarHint},
		})
	}
	return configs, nil
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
		return nil, c.parseErrorResponse(resp)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &user, nil
}
