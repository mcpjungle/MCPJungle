package live_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------
// Shared response types
// -----------------------------------------------------------------------

// toolInvokeResult is the JSON response from POST /api/v0/tools/invoke.
type toolInvokeResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// renderedPromptResult is the JSON response from POST /api/v0/prompts/render.
type renderedPromptResult struct {
	Messages []struct {
		Role    string `json:"role"`
		Content struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"messages"`
}

// -----------------------------------------------------------------------
// HTTP helpers
// -----------------------------------------------------------------------

// drain closes an HTTP response body without reading it.
func drain(r *http.Response) { r.Body.Close() }

// decodeJSON decodes the JSON response body into target and closes the body.
func decodeJSON(t *testing.T, r *http.Response, target any) {
	t.Helper()
	defer r.Body.Close()
	require.NoError(t, json.NewDecoder(r.Body).Decode(target))
}

// readBody reads and closes the response body as a string.
func readBody(r *http.Response) string {
	b, _ := io.ReadAll(r.Body)
	r.Body.Close()
	return string(b)
}

// toolNames extracts the "name" field from a slice of JSON objects.
func toolNames(tools []map[string]any) []string {
	names := make([]string, 0, len(tools))
	for _, tl := range tools {
		if n, ok := tl["name"].(string); ok {
			names = append(names, n)
		}
	}
	return names
}

// promptNames extracts the "name" field from a slice of JSON prompt objects.
func promptNames(prompts []map[string]any) []string {
	names := make([]string, 0, len(prompts))
	for _, p := range prompts {
		if n, ok := p["name"].(string); ok {
			names = append(names, n)
		}
	}
	return names
}

// -----------------------------------------------------------------------
// MCP client helpers
// -----------------------------------------------------------------------

// authRoundTripper adds a Bearer token to every request.
type authRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (a *authRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+a.token)
	return a.base.RoundTrip(r)
}

func liveMCPClient(t *testing.T, endpoint string) *client.Client {
	t.Helper()
	return liveMCPClientWithToken(t, endpoint, "")
}

func liveMCPClientWithToken(t *testing.T, endpoint string, token string) *client.Client {
	t.Helper()

	var httpClient *http.Client
	if token != "" {
		httpClient = &http.Client{Transport: &authRoundTripper{token: token, base: http.DefaultTransport}}
	}

	var c *client.Client
	var err error

	// SSE endpoints (/sse, /v0/groups/:name/sse) need SSEClientTransport.
	// Streamable HTTP endpoints (/mcp, /v0/groups/:name/mcp) need StreamableHttpClient.
	if strings.Contains(endpoint, "/sse") {
		opts := []transport.ClientOption{}
		if httpClient != nil {
			opts = append(opts, transport.WithHTTPClient(httpClient))
		}
		c, err = client.NewSSEMCPClient(endpoint, opts...)
	} else {
		opts := []transport.StreamableHTTPCOption{}
		if httpClient != nil {
			opts = append(opts, transport.WithHTTPBasicClient(httpClient))
		}
		c, err = client.NewStreamableHttpClient(endpoint, opts...)
	}
	require.NoError(t, err, "create client for %s", endpoint)

	ctx := context.Background()
	require.NoError(t, c.Start(ctx), "start client for %s", endpoint)

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ClientInfo: mcp.Implementation{Name: "live-test", Version: "1.0"},
		},
	})
	require.NoError(t, err, "initialize client for %s", endpoint)

	t.Cleanup(func() { c.Close() })
	return c
}
