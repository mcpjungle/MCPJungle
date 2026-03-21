package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListResources fetches the list of resources, optionally filtered by server name.
// If server is an empty string, this method fetches all resources.
func (c *Client) ListResources(server string) ([]*types.Resource, error) {
	u, _ := c.constructAPIEndpoint("/resources")
	req, _ := c.newRequest(http.MethodGet, u, nil)
	if server != "" {
		q := req.URL.Query()
		q.Add("server", server)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var resources []*types.Resource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return resources, nil
}
