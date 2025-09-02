package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestMcpClient(t *testing.T) {
	t.Run("McpClient struct creation", func(t *testing.T) {
		client := McpClient{
			Name:        "test-client",
			Description: "A test MCP client",
			AllowList:   []string{"server1", "server2"},
		}

		if client.Name != "test-client" {
			t.Errorf("Expected Name to be 'test-client', got %s", client.Name)
		}
		if client.Description != "A test MCP client" {
			t.Errorf("Expected Description to be 'A test MCP client', got %s", client.Description)
		}
		if len(client.AllowList) != 2 {
			t.Errorf("Expected AllowList to have 2 items, got %d", len(client.AllowList))
		}
		if client.AllowList[0] != "server1" {
			t.Errorf("Expected first AllowList item to be 'server1', got %s", client.AllowList[0])
		}
		if client.AllowList[1] != "server2" {
			t.Errorf("Expected second AllowList item to be 'server2', got %s", client.AllowList[1])
		}
	})

	t.Run("McpClient struct zero values", func(t *testing.T) {
		var client McpClient

		if client.Name != "" {
			t.Errorf("Expected empty Name, got %s", client.Name)
		}
		if client.Description != "" {
			t.Errorf("Expected empty Description, got %s", client.Description)
		}
		if client.AllowList != nil {
			t.Error("Expected AllowList to be nil for zero value, got non-nil")
		}
	})

	t.Run("McpClient with empty allow list", func(t *testing.T) {
		client := McpClient{
			AllowList: []string{},
		}

		if len(client.AllowList) != 0 {
			t.Errorf("Expected empty AllowList, got %d items", len(client.AllowList))
		}
	})

	t.Run("McpClient with nil allow list", func(t *testing.T) {
		client := McpClient{
			AllowList: nil,
		}

		if client.AllowList != nil {
			t.Error("Expected AllowList to be nil")
		}
	})

	t.Run("McpClient JSON marshaling", func(t *testing.T) {
		client := McpClient{
			Name:        "json-client",
			Description: "Client for JSON testing",
			AllowList:   []string{"server1", "server2", "server3"},
		}

		data, err := json.Marshal(client)
		if err != nil {
			t.Fatalf("Failed to marshal McpClient: %v", err)
		}

		expected := `{"name":"json-client","description":"Client for JSON testing","allow_list":["server1","server2","server3"]}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})

	t.Run("McpClient JSON marshaling with empty allow list", func(t *testing.T) {
		client := McpClient{
			Name:        "empty-json-client",
			Description: "Client with empty allow list for JSON testing",
			AllowList:   []string{},
		}

		data, err := json.Marshal(client)
		if err != nil {
			t.Fatalf("Failed to marshal McpClient with empty allow list: %v", err)
		}

		expected := `{"name":"empty-json-client","description":"Client with empty allow list for JSON testing","allow_list":[]}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})

	t.Run("McpClient JSON unmarshaling", func(t *testing.T) {
		jsonData := `{"name":"unmarshal-client","description":"Client from JSON","allow_list":["serverA","serverB"]}`
		var client McpClient

		err := json.Unmarshal([]byte(jsonData), &client)
		if err != nil {
			t.Fatalf("Failed to unmarshal McpClient: %v", err)
		}

		if client.Name != "unmarshal-client" {
			t.Errorf("Expected Name 'unmarshal-client', got %s", client.Name)
		}
		if client.Description != "Client from JSON" {
			t.Errorf("Expected Description 'Client from JSON', got %s", client.Description)
		}
		if len(client.AllowList) != 2 {
			t.Errorf("Expected AllowList to have 2 items, got %d", len(client.AllowList))
		}
		if client.AllowList[0] != "serverA" {
			t.Errorf("Expected first AllowList item 'serverA', got %s", client.AllowList[0])
		}
		if client.AllowList[1] != "serverB" {
			t.Errorf("Expected second AllowList item 'serverB', got %s", client.AllowList[1])
		}
	})

	t.Run("McpClient JSON unmarshaling with empty allow list", func(t *testing.T) {
		jsonData := `{"name":"empty-unmarshal-client","description":"Client with empty allow list","allow_list":[]}`
		var client McpClient

		err := json.Unmarshal([]byte(jsonData), &client)
		if err != nil {
			t.Fatalf("Failed to unmarshal McpClient with empty allow list: %v", err)
		}

		if client.Name != "empty-unmarshal-client" {
			t.Errorf("Expected Name 'empty-unmarshal-client', got %s", client.Name)
		}
		if len(client.AllowList) != 0 {
			t.Errorf("Expected empty AllowList, got %d items", len(client.AllowList))
		}
	})

	t.Run("McpClient JSON unmarshaling with missing allow list", func(t *testing.T) {
		jsonData := `{"name":"missing-allow-list-client","description":"Client with missing allow list"}`
		var client McpClient

		err := json.Unmarshal([]byte(jsonData), &client)
		if err != nil {
			t.Fatalf("Failed to unmarshal McpClient with missing allow list: %v", err)
		}

		if client.Name != "missing-allow-list-client" {
			t.Errorf("Expected Name 'missing-allow-list-client', got %s", client.Name)
		}
		if client.AllowList != nil {
			t.Error("Expected AllowList to be nil when missing from JSON")
		}
	})
}

func TestMcpClientEdgeCases(t *testing.T) {
	t.Run("McpClient with very long name", func(t *testing.T) {
		longName := "a-very-long-client-name-that-exceeds-normal-length-limits-for-testing-purposes"
		client := McpClient{
			Name: longName,
		}

		if client.Name != longName {
			t.Errorf("Expected Name to match long name, got %s", client.Name)
		}
	})

	t.Run("McpClient with very long description", func(t *testing.T) {
		longDesc := "This is a very long description that contains many words and characters to test the behavior of the McpClient struct when dealing with lengthy text content that might be used in real-world scenarios"
		client := McpClient{
			Description: longDesc,
		}

		if client.Description != longDesc {
			t.Errorf("Expected Description to match long description")
		}
	})

	t.Run("McpClient with many allowed servers", func(t *testing.T) {
		manyServers := make([]string, 100)
		for i := 0; i < 100; i++ {
			manyServers[i] = fmt.Sprintf("server%d", i)
		}

		client := McpClient{
			AllowList: manyServers,
		}

		if len(client.AllowList) != 100 {
			t.Errorf("Expected AllowList to have 100 items, got %d", len(client.AllowList))
		}
		if client.AllowList[0] != "server0" {
			t.Errorf("Expected first server to be 'server0', got %s", client.AllowList[0])
		}
		if client.AllowList[99] != "server99" {
			t.Errorf("Expected last server to be 'server99', got %s", client.AllowList[99])
		}
	})

	t.Run("McpClient with special characters in name", func(t *testing.T) {
		specialName := "client-with-special-chars-!@#$%^&*()_+-=[]{}|;':\",./<>?"
		client := McpClient{
			Name: specialName,
		}

		if client.Name != specialName {
			t.Errorf("Expected Name to match special name, got %s", client.Name)
		}
	})
}
