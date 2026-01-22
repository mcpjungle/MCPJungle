package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

type InitServerResponse struct {
	AdminAccessToken string `json:"admin_access_token"`
}

// InitServer sends a request to initialize the server in enterprise mode
func (c *Client) InitServer() (*InitServerResponse, error) {
	u, _ := url.JoinPath(c.baseURL, "/init")

	// TODO: Replace ModeProd with ModeEnterprise in future.
	// For backward compatibility, the client sends ModeProd to indicate enterprise mode.
	// This is because mcpjungle server versions < 0.2.12 do not recognize ModeEnterprise.
	// We want to avoid breaking the client's compatibility with older server versions.
	// Servers >= 0.2.12 will treat ModeProd as enterprise mode.
	// In future, once we drop support for older server versions, we can switch to ModeEnterprise.
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
		return nil, c.parseErrorResponse(resp)
	}

	var initResp InitServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &initResp, nil
}

// CreateOAuthClient creates a new OAuth client in the registry
func (c *Client) CreateOAuthClient(client *types.CreateOAuthClientRequest) (*model.OAuthClient, error) {
	u, err := c.constructAPIEndpoint("/oauth-clients")
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(client)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(http.MethodPost, u, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseErrorResponse(resp)
	}

	var created model.OAuthClient
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, err
	}
	return &created, nil
}
