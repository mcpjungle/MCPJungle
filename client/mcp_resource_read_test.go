package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestGetResource(t *testing.T) {
	t.Parallel()

	expected := &types.Resource{
		URI:         "system://system/info",
		Name:        "polaro__system_info",
		Enabled:     true,
		Description: "CPU load, memory, disk, and uptime",
		MIMEType:    "application/json",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resource") {
			t.Errorf("Expected path to end with /resource, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("uri") != expected.URI {
			t.Errorf("Expected uri=%s, got %s", expected.URI, r.URL.Query().Get("uri"))
		}
		if r.URL.Query().Get("server") != "polaro" {
			t.Errorf("Expected server=polaro, got %s", r.URL.Query().Get("server"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", &http.Client{})
	resource, err := client.GetResource(expected.URI, "polaro")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resource.URI != expected.URI {
		t.Fatalf("Expected URI %s, got %s", expected.URI, resource.URI)
	}
}

func TestReadResource(t *testing.T) {
	t.Parallel()

	expected := &types.ResourceReadResult{
		Contents: []map[string]any{
			{
				"uri":      "system://system/info",
				"mimeType": "application/json",
				"text":     "{\"uptime\":\"up 1 hour\"}",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/read") {
			t.Errorf("Expected path to end with /resources/read, got %s", r.URL.Path)
		}

		var request types.ResourceReadRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if request.URI != "system://system/info" {
			t.Errorf("Expected URI system://system/info, got %s", request.URI)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", &http.Client{})
	result, err := client.ReadResource("system://system/info", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Contents))
	}
}
