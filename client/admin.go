package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"io"
	"net/http"
	"net/url"
)

type InitServerResponse struct {
	AdminAccessToken string `json:"admin_access_token"`
}

// InitServer sends a request to initialize the server in production mode
func (c *Client) InitServer() (*InitServerResponse, error) {
	u, _ := url.JoinPath(c.baseURL, "/init")

	payload := struct {
		Mode string `json:"mode"`
	}{
		Mode: string(model.ModeProd),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(u, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var initResp InitServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &initResp, nil
}

type CreateUserResponse struct {
	UserAccessToken string `json:"user_access_token"`
}

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
