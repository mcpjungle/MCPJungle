package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/internal/model"
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var initResp InitServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &initResp, nil
}
