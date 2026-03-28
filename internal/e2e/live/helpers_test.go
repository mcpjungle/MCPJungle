package live_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

func liveMCPClient(t *testing.T, endpoint string) *mcp.ClientSession {
	t.Helper()
	return liveMCPClientWithToken(t, endpoint, "")
}

func liveMCPClientWithToken(t *testing.T, endpoint string, token string) *mcp.ClientSession {
	t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "live-test", Version: "1.0"}, nil)

	var httpClient *http.Client
	if token != "" {
		httpClient = &http.Client{Transport: &authRoundTripper{token: token, base: http.DefaultTransport}}
	}

	// SSE endpoints (/sse, /v0/groups/:name/sse) need SSEClientTransport.
	// Streamable HTTP endpoints (/mcp, /v0/groups/:name/mcp) need StreamableClientTransport.
	var transport mcp.Transport
	if strings.Contains(endpoint, "/sse") {
		tr := &mcp.SSEClientTransport{Endpoint: endpoint}
		if httpClient != nil {
			tr.HTTPClient = httpClient
		}
		transport = tr
	} else {
		tr := &mcp.StreamableClientTransport{
			Endpoint:             endpoint,
			DisableStandaloneSSE: true,
		}
		if httpClient != nil {
			tr.HTTPClient = httpClient
		}
		transport = tr
	}

	cs, err := c.Connect(context.Background(), transport, nil)
	require.NoError(t, err, "connect to %s", endpoint)
	t.Cleanup(func() { cs.Close() })
	return cs
}
