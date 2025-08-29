package client

import (
	"github.com/mcpjungle/mcpjungle/internal/api"
	"io"
	"net/http"
	"net/url"
)

// Client represents a client for interacting with the MCPJungle HTTP API
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

func NewClient(baseURL string, accessToken string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient:  httpClient,
	}
}

// constructAPIEndpoint constructs the full API endpoint URL where a request must be sent
func (c *Client) constructAPIEndpoint(suffixPath string) (string, error) {
	return url.JoinPath(c.baseURL, api.V0ApiPathPrefix, suffixPath)
}

// newRequest creates a new HTTP request with the specified method, URL, and body.
// It automatically adds the Authorization header if an access token is present.
func (c *Client) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	return req, nil
}
