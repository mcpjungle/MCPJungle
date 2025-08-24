package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcpjungle/mcpjungle/internal/model"
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
