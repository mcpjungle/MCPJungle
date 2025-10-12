package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestListPrompts(t *testing.T) {
	t.Parallel()

	t.Run("list all prompts", func(t *testing.T) {
		expected := []model.Prompt{
			{Name: "prompt1", Description: "desc1"},
			{Name: "prompt2", Description: "desc2"},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/prompts") {
				t.Errorf("Expected /prompts, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("server") != "" {
				t.Errorf("Expected no server filter")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.ListPrompts("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(result) != len(expected) {
			t.Errorf("Expected %d prompts, got %d", len(expected), len(result))
		}
	})

	t.Run("list prompts with server filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("server") != "srv" {
				t.Errorf("Expected server=srv, got %s", r.URL.Query().Get("server"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]model.Prompt{})
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		_, err := client.ListPrompts("srv")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("fail"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.ListPrompts("")
		if err == nil || result != nil {
			t.Error("Expected error and nil result")
		}
	})
}

func TestGetPrompt(t *testing.T) {
	t.Parallel()

	t.Run("get prompt by name", func(t *testing.T) {
		expected := model.Prompt{Name: "prompt1", Description: "desc"}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("name") != "prompt1" {
				t.Errorf("Expected name=prompt1, got %s", r.URL.Query().Get("name"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.GetPrompt("prompt1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.Name != expected.Name {
			t.Errorf("Expected %s, got %s", expected.Name, result.Name)
		}
	})

	t.Run("prompt not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.GetPrompt("missing")
		if err == nil || result != nil {
			t.Error("Expected error and nil result")
		}
	})
}

func TestGetPromptWithArgs(t *testing.T) {
	t.Parallel()

	t.Run("render prompt with args", func(t *testing.T) {
		expectedMessages := []types.PromptMessage{
			{
				Role: "assistant", Content: map[string]any{"text": "Hello, Alice!"},
			},
		}
		expected := &types.PromptResult{
			Description: "Test prompt result",
			Messages:    expectedMessages,
			Meta:        nil,
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/prompts/render") {
				t.Errorf("Expected /prompts/render, got %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req types.PromptGetRequest
			_ = json.Unmarshal(body, &req)
			if req.Name != "greet" || req.Arguments["name"] != "Alice" {
				t.Errorf("Unexpected request: %+v", req)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.GetPromptWithArgs("greet", map[string]string{"name": "Alice"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.Description != expected.Description {
			t.Errorf("Expected description %s, got %s", expected.Description, result.Description)
		}
		if len(result.Messages) != len(expected.Messages) {
			t.Fatalf("Expected %d messages, got %d", len(expected.Messages), len(result.Messages))
		}
		for i, msg := range result.Messages {
			if msg.Role != expected.Messages[i].Role {
				t.Errorf("Expected role %s, got %s", expected.Messages[i].Role, msg.Role)
			}
			if msg.Content["text"] != expected.Messages[i].Content["text"] {
				t.Errorf("Expected content %s, got %s", expected.Messages[i].Content["text"], msg.Content["text"])
			}
		}
		if result.Meta != nil {
			t.Errorf("Expected nil Meta, got %+v", result.Meta)
		}
	})

	t.Run("render error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad args"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.GetPromptWithArgs("greet", map[string]string{"name": "Bob"})
		if err == nil || result != nil {
			t.Error("Expected error and nil result")
		}
	})
}

func TestEnableDisablePrompts(t *testing.T) {
	t.Parallel()

	t.Run("enable prompts", func(t *testing.T) {
		expected := []string{"prompt1", "prompt2"}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/prompts/enable") {
				t.Errorf("Expected /prompts/enable, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("entity") != "test-entity" {
				t.Errorf("Expected entity=test-entity, got %s", r.URL.Query().Get("entity"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.EnablePrompts("test-entity")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(result) != len(expected) {
			t.Errorf("Expected %d prompts, got %d", len(expected), len(result))
		}
	})

	t.Run("disable prompts", func(t *testing.T) {
		expected := []string{"prompt1"}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/prompts/disable") {
				t.Errorf("Expected /prompts/disable, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("entity") != "test-entity" {
				t.Errorf("Expected entity=test-entity, got %s", r.URL.Query().Get("entity"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.DisablePrompts("test-entity")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(result) != len(expected) {
			t.Errorf("Expected %d prompts, got %d", len(expected), len(result))
		}
	})

	t.Run("enable prompts error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("fail"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.EnablePrompts("fail-entity")
		if err == nil || result != nil {
			t.Error("Expected error and nil result")
		}
	})

	t.Run("disable prompts error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("fail"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "token", &http.Client{})
		result, err := client.DisablePrompts("fail-entity")
		if err == nil || result != nil {
			t.Error("Expected error and nil result")
		}
	})
}
