package types

import (
	"encoding/json"
	"testing"
)

func TestMcpClient(t *testing.T) {
	t.Parallel()

	// Test struct creation
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
}

func TestMcpClientZeroValues(t *testing.T) {
	t.Parallel()

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
}

func TestMcpClientJSONMarshaling(t *testing.T) {
	t.Parallel()

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
}

func TestMcpClientJSONUnmarshaling(t *testing.T) {
	t.Parallel()

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
}

func TestMcpClientEdgeCases(t *testing.T) {
	t.Parallel()

	// Test with empty allow list
	client := McpClient{
		AllowList: []string{},
	}
	if len(client.AllowList) != 0 {
		t.Errorf("Expected empty AllowList, got %d items", len(client.AllowList))
	}

	// Test with nil allow list
	client = McpClient{
		AllowList: nil,
	}
	if client.AllowList != nil {
		t.Error("Expected AllowList to be nil")
	}

	// Test JSON unmarshaling with missing allow list
	jsonData := `{"name":"missing-allow-list-client","description":"Client with missing allow list"}`
	var unmarshalClient McpClient

	err := json.Unmarshal([]byte(jsonData), &unmarshalClient)
	if err != nil {
		t.Fatalf("Failed to unmarshal McpClient with missing allow list: %v", err)
	}

	if unmarshalClient.Name != "missing-allow-list-client" {
		t.Errorf("Expected Name 'missing-allow-list-client', got %s", unmarshalClient.Name)
	}
	if unmarshalClient.AllowList != nil {
		t.Error("Expected AllowList to be nil when missing from JSON")
	}
}
