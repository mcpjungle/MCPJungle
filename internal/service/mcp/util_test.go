package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

func TestValidateServerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "server_1", false},
		{"valid name", "server_2_multiple_underscores", false},
		{"valid hyphen", "server-2", false},
		{"trailing underscore", "_server_", true},
		{"only underscore", "_", true},
		{"invalid slash", "server/3", true},
		{"invalid special char", "server$", true},
		{"double underscore", "server__name", true},
		{"double underscore", "__server", true},
		{"double underscore", "server__", true},
		{"only double underscore", "__", true},
		{"triple underscore", "server___name", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServerName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, apierrors.ErrInvalidInput) {
				t.Errorf("validateServerName(%q) error = %v, want ErrInvalidInput", tt.input, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com", false},
		{"valid uppercase scheme", "HTTP://example.com", false},
		{"valid with path", "http://example.com/mcp", false},
		{"valid with port", "http://localhost:8080/mcp", false},
		{"no scheme", "blahblahblah", true},
		{"ftp scheme", "ftp://example.com", true},
		{"empty", "", true},
		{"no host", "http://", true},
		{"scheme only", "https://", true},
		{"missing host with path", "http:///foo", true},
		{"missing host without slashes", "https:example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, apierrors.ErrInvalidInput) {
				t.Errorf("validateURL(%q) error = %v, want ErrInvalidInput", tt.input, err)
			}
		})
	}
}

func TestMergeServerToolNames(t *testing.T) {
	tests := []struct {
		server string
		tool   string
		want   string
	}{
		{"myserver", "mytool", "myserver__mytool"},
		{"myserver", "my/tool", "myserver__my/tool"},
		{"_myserver", "mytool", "_myserver__mytool"},
		{"my_server", "my_tool", "my_server__my_tool"},
		{"my-server", "my-tool", "my-server__my-tool"},
	}
	for _, tt := range tests {
		caseName := fmt.Sprintf("server:%s,tool: %s", tt.server, tt.tool)
		t.Run(caseName, func(t *testing.T) {
			got := mergeServerToolNames(tt.server, tt.tool)
			if got != tt.want {
				t.Errorf("mergeServerToolNames(%q, %q) = %q, want %q", tt.server, tt.tool, got, tt.want)
			}
		})
	}
}

func TestSplitServerToolName(t *testing.T) {
	tests := []struct {
		input      string
		wantServer string
		wantTool   string
		wantOK     bool
	}{
		{"server__tool", "server", "tool", true},
		{"a__b/c", "a", "b/c", true},
		{"a__b__c", "a", "b__c", true},
		{"_abc__def", "_abc", "def", true},
		{"no_separator", "no_separator", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			server, tool, ok := splitServerToolName(tt.input)
			if server != tt.wantServer || tool != tt.wantTool || ok != tt.wantOK {
				t.Errorf("splitServerToolName(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, server, tool, ok, tt.wantServer, tt.wantTool, tt.wantOK)
			}
		})
	}
}

func TestIsLoopbackURL(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		want   bool
	}{
		// IPv4 loopback
		{"IPv4 127.0.0.1", "http://127.0.0.1:8080", true},
		{"IPv4 127.0.0.1 no port", "http://127.0.0.1", true},
		{"IPv4 127.0.0.2", "http://127.0.0.2", true}, // 127.0.0.0/8 is loopback
		{"IPv4 127.255.255.255", "http://127.255.255.255", true},
		{"IPv4 0.0.0.0", "http://0.0.0.0:9000", false}, // 0.0.0.0 is not loopback, it's "any"
		// IPv6 loopback
		{"IPv6 ::1", "http://[::1]:8080", true},
		{"IPv6 ::1 no port", "http://[::1]", true},
		// Hostname loopback
		{"localhost", "http://localhost:8080", true},
		{"localhost no port", "http://localhost", true},
		{"LOCALHOST uppercase", "http://LOCALHOST", true},
		// Non-loopback IPv4
		{"IPv4 public", "http://8.8.8.8:8080", false},
		{"IPv4 private", "http://192.168.1.1", false},
		// Non-loopback IPv6
		{"IPv6 public", "http://[2001:4860:4860::8888]:443", false},
		// Hostname non-loopback
		{"example.com", "http://example.com", false},
		{"sub.domain.com", "http://sub.domain.com:1234", false},
		// Malformed URLs
		{"empty string", "", false},
		{"no scheme", "127.0.0.1:8080", false},
		{"garbage", "not a url", false},
		// Edge cases
		{"IPv4 with userinfo", "http://user:pass@127.0.0.1:8080", true},
		{"IPv6 with userinfo", "http://user:pass@[::1]:8080", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLoopbackURL(tt.rawURL)
			if got != tt.want {
				t.Errorf("isLoopbackURL(%q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestConvertToolModelToMcpObject_Success(t *testing.T) {
	in := &model.Tool{
		Name:        "mytool",
		Description: "a tool",
		InputSchema: []byte("{}"), // valid empty JSON object
		Annotations: nil,
	}

	got, err := convertToolModelToMcpObject(in)
	if err != nil {
		t.Fatalf("convertToolModelToMcpObject() error = %v, want nil", err)
	}

	if got.Name != in.Name {
		t.Errorf("Name = %q, want %q", got.Name, in.Name)
	}
	if got.Description != in.Description {
		t.Errorf("Description = %q, want %q", got.Description, in.Description)
	}

	// Expect an unmarshalled (zero) ToolInputSchema when provided "{}"
	if !reflect.DeepEqual(got.InputSchema, mcp.ToolInputSchema{}) {
		t.Errorf("InputSchema = %+v, want zero value %+v", got.InputSchema, mcp.ToolInputSchema{})
	}
}

func TestConvertToolModelToMcpObject_AnnotationsSuccess(t *testing.T) {
	in := &model.Tool{
		Name:        "annToolOk",
		Description: "tool with annotations",
		InputSchema: []byte("{}"),                 // valid schema
		Annotations: []byte(`{"title":"foobar"}`), // valid annotations JSON
	}

	got, err := convertToolModelToMcpObject(in)
	if err != nil {
		t.Fatalf("convertToolModelToMcpObject() error = %v, want nil", err)
	}

	// Expect annotations to be set (not the zero value)
	if reflect.DeepEqual(got.Annotations, mcp.ToolAnnotation{}) {
		t.Fatalf("Annotations = %+v, want non-zero value when annotations are valid", got.Annotations)
	}

	// Ensure the annotations contain the expected key/value when marshalled back to JSON
	b, err := json.Marshal(got.Annotations)
	if err != nil {
		t.Fatalf("failed to marshal got.Annotations: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"title"`) || !strings.Contains(s, `"foobar"`) {
		t.Errorf("Annotations JSON = %s, want it to contain %q:%q", s, "title", "foobar")
	}
}

func TestConvertToolModelToMcpObject_InvalidInputSchema(t *testing.T) {
	in := &model.Tool{
		Name:        "badtool",
		Description: "bad schema",
		InputSchema: []byte("not valid json"),
	}

	_, err := convertToolModelToMcpObject(in)
	if err == nil {
		t.Fatalf("convertToolModelToMcpObject() error = nil, want non-nil for invalid input schema")
	}
}

func TestConvertToolModelToMcpObject_InvalidAnnotationsLoggedButNoError(t *testing.T) {
	in := &model.Tool{
		Name:        "annTool",
		Description: "tool with bad annotations",
		InputSchema: []byte("{}"),            // valid schema
		Annotations: []byte("{ not: json }"), // invalid annotations JSON
	}

	got, err := convertToolModelToMcpObject(in)
	if err != nil {
		t.Fatalf("convertToolModelToMcpObject() error = %v, want nil (annotation errors are logged)", err)
	}

	// When annotations fail to unmarshal, function should not set them and should not error.
	if !reflect.DeepEqual(got.Annotations, mcp.ToolAnnotation{}) {
		t.Errorf("Annotations = %+v, want zero value %+v when annotations are invalid", got.Annotations, mcp.ToolAnnotation{})
	}
}

func TestPrepareSHTTPClientOptions_NoHeadersNoBearer(t *testing.T) {
	conf := &model.StreamableHTTPConfig{
		URL:         "http://example.com",
		BearerToken: "",
		Headers:     nil,
	}
	opts := prepareSHTTPClientOptions("srv", conf)
	// The proxy-aware HTTP client option is always present, even when no headers
	// and no bearer token are configured.
	if len(opts) != 1 {
		t.Fatalf("expected 1 option (proxy client) when no headers and no bearer token, got %d", len(opts))
	}
}

func TestPrepareSHTTPClientOptions_HeaderOnly(t *testing.T) {
	conf := &model.StreamableHTTPConfig{
		URL:         "http://example.com",
		BearerToken: "",
		Headers: map[string]string{
			"X-Custom": "value",
		},
	}
	opts := prepareSHTTPClientOptions("srv", conf)
	// proxy client option + headers option
	if len(opts) != 2 {
		t.Fatalf("expected 2 options (proxy client + headers) when headers present, got %d", len(opts))
	}
}

func TestPrepareSHTTPClientOptions_BearerOnly(t *testing.T) {
	conf := &model.StreamableHTTPConfig{
		URL:         "http://example.com",
		BearerToken: "secrettoken",
		Headers:     nil,
	}
	opts := prepareSHTTPClientOptions("srv", conf)
	// proxy client option + bearer-token-as-header option
	if len(opts) != 2 {
		t.Fatalf("expected 2 options (proxy client + bearer header) when bearer token present, got %d", len(opts))
	}
}

func TestPrepareSHTTPClientOptions_BearerWithCustomAuthorization(t *testing.T) {
	conf := &model.StreamableHTTPConfig{
		URL:         "http://example.com",
		BearerToken: "should_be_ignored",
		Headers: map[string]string{
			"Authorization": "Custom auth-value",
		},
	}

	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	opts := prepareSHTTPClientOptions("my-server", conf)
	// proxy client option + custom Authorization header option
	if len(opts) != 2 {
		t.Fatalf("expected 2 options (proxy client + custom Authorization header) when custom Authorization header present, got %d", len(opts))
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "custom Authorization header will be used for MCP server my-server; bearer_token ignored") {
		t.Fatalf("expected log to mention bearer_token ignored when custom Authorization header present, got: %q", logOutput)
	}
}

// clearProxyEnv unsets every proxy-related variable (both upper and lower case forms)
// that httpproxy.FromEnvironment consults, so tests are not influenced by the ambient
// CI/developer environment.
func clearProxyEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"HTTP_PROXY", "http_proxy",
		"HTTPS_PROXY", "https_proxy",
		"NO_PROXY", "no_proxy",
	} {
		t.Setenv(k, "")
	}
}

func TestProxyAwareHTTPClient_HonorsHTTPSProxy(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("HTTPS_PROXY", "http://proxy.example.com:8080")

	c := proxyAwareHTTPClient()
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("expected a non-nil Proxy function on the transport")
	}

	req, err := http.NewRequest(http.MethodGet, "https://upstream.example.com/mcp", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL == nil {
		t.Fatal("expected request to be proxied, got nil proxy URL")
	}
	if proxyURL.Host != "proxy.example.com:8080" {
		t.Fatalf("expected proxy host proxy.example.com:8080, got %s", proxyURL.Host)
	}
}

func TestProxyAwareHTTPClient_NoProxyEnvDefault(t *testing.T) {
	clearProxyEnv(t)

	c := proxyAwareHTTPClient()
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.Transport)
	}

	req, err := http.NewRequest(http.MethodGet, "https://upstream.example.com/mcp", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL != nil {
		t.Fatalf("expected no proxy when no env vars set, got %s", proxyURL)
	}
}

func TestProxyAwareHTTPClient_NoProxyHonored(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("HTTPS_PROXY", "http://proxy.example.com:8080")
	t.Setenv("NO_PROXY", "internal.example.com")

	c := proxyAwareHTTPClient()
	tr := c.Transport.(*http.Transport)

	// A URL matching NO_PROXY must bypass the proxy.
	bypassReq, _ := http.NewRequest(http.MethodGet, "https://internal.example.com/mcp", nil)
	proxyURL, err := tr.Proxy(bypassReq)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL != nil {
		t.Fatalf("expected NO_PROXY host to bypass proxy, got %s", proxyURL)
	}

	// A non-matching URL must still be proxied.
	proxiedReq, _ := http.NewRequest(http.MethodGet, "https://upstream.example.com/mcp", nil)
	proxyURL, err = tr.Proxy(proxiedReq)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.Host != "proxy.example.com:8080" {
		t.Fatalf("expected non-matching host to be proxied via proxy.example.com:8080, got %v", proxyURL)
	}
}

func waitForLogMessage(t *testing.T, buf *bytes.Buffer, want string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected log to contain %q, got %q", want, buf.String())
}

func newTestStdioClient(t *testing.T) (*client.Client, *os.File, *os.File) {
	t.Helper()

	stdoutReader, stdoutWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	t.Cleanup(func() {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		_ = stdinReader.Close()
		_ = stdinWriter.Close()
		_ = stderrReader.Close()
		_ = stderrWriter.Close()
	})

	stdio := transport.NewIO(stdoutReader, stdinWriter, stderrReader)
	return client.NewClient(stdio), stderrReader, stderrWriter
}

func TestCaptureStdioServerStderr_LogsGracefulExitOnEOF(t *testing.T) {
	c, _, stderrWriter := newTestStdioClient(t)

	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	captureStdioServerStderr("demo", c)
	_ = stderrWriter.Close()

	waitForLogMessage(t, &buf, "['demo' MCP Server] [DEBUG] server process has exited gracefully")
}

func TestCaptureStdioServerStderr_LogsClientShutdownOnClosedPipe(t *testing.T) {
	c, stderrReader, _ := newTestStdioClient(t)

	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	captureStdioServerStderr("demo", c)
	_ = stderrReader.Close()

	waitForLogMessage(t, &buf, "['demo' MCP Server] [DEBUG] stderr pipe closed during client shutdown")
}
