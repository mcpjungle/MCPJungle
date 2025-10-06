package model

import (
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
)

// mockToolResolver implements ToolResolver for testing
type mockToolResolver struct {
	serverTools map[string][]Tool
}

func (m *mockToolResolver) ListToolsByServer(serverName string) ([]Tool, error) {
	if tools, exists := m.serverTools[serverName]; exists {
		return tools, nil
	}
	return []Tool{}, nil
}

func TestToolGroup_GetTools(t *testing.T) {
	tools := []string{"tool1", "tool2"}
	toolsJSON, _ := json.Marshal(tools)

	group := &ToolGroup{
		IncludedTools: datatypes.JSON(toolsJSON),
	}

	result, err := group.GetTools()
	if err != nil {
		t.Fatalf("GetTools() failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(result))
	}
	if result[0] != "tool1" || result[1] != "tool2" {
		t.Errorf("Expected [tool1, tool2], got %v", result)
	}
}

func TestToolGroup_GetServers(t *testing.T) {
	servers := []string{"server1", "server2"}
	serversJSON, _ := json.Marshal(servers)

	group := &ToolGroup{
		IncludedServers: datatypes.JSON(serversJSON),
	}

	result, err := group.GetServers()
	if err != nil {
		t.Fatalf("GetServers() failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(result))
	}
	if result[0] != "server1" || result[1] != "server2" {
		t.Errorf("Expected [server1, server2], got %v", result)
	}
}

func TestToolGroup_GetExcludedTools(t *testing.T) {
	tools := []string{"excluded1", "excluded2"}
	toolsJSON, _ := json.Marshal(tools)

	group := &ToolGroup{
		ExcludedTools: datatypes.JSON(toolsJSON),
	}

	result, err := group.GetExcludedTools()
	if err != nil {
		t.Fatalf("GetExcludedTools() failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 excluded tools, got %d", len(result))
	}
	if result[0] != "excluded1" || result[1] != "excluded2" {
		t.Errorf("Expected [excluded1, excluded2], got %v", result)
	}
}

func TestToolGroup_ResolveEffectiveTools(t *testing.T) {
	// Create mock resolver with some test data
	resolver := &mockToolResolver{
		serverTools: map[string][]Tool{
			"time": {
				{Name: "time__get_current_time"},
				{Name: "time__convert_time"},
				{Name: "time__format_time"},
			},
			"deepwiki": {
				{Name: "deepwiki__read_wiki_contents"},
				{Name: "deepwiki__search_wiki"},
			},
		},
	}

	t.Run("IncludedTools only", func(t *testing.T) {
		tools := []string{"manual__tool1", "manual__tool2"}
		toolsJSON, _ := json.Marshal(tools)

		group := &ToolGroup{
			IncludedTools: datatypes.JSON(toolsJSON),
		}

		result, err := group.ResolveEffectiveTools(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveTools() failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(result))
		}

		toolMap := make(map[string]bool)
		for _, tool := range result {
			toolMap[tool] = true
		}

		if !toolMap["manual__tool1"] || !toolMap["manual__tool2"] {
			t.Errorf("Expected manual tools, got %v", result)
		}
	})

	t.Run("IncludedServers only", func(t *testing.T) {
		servers := []string{"time"}
		serversJSON, _ := json.Marshal(servers)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
		}

		result, err := group.ResolveEffectiveTools(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveTools() failed: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 tools from time server, got %d", len(result))
		}

		toolMap := make(map[string]bool)
		for _, tool := range result {
			toolMap[tool] = true
		}

		expectedTools := []string{"time__get_current_time", "time__convert_time", "time__format_time"}
		for _, expectedTool := range expectedTools {
			if !toolMap[expectedTool] {
				t.Errorf("Expected tool %s not found in result %v", expectedTool, result)
			}
		}
	})

	t.Run("IncludedServers with ExcludedTools", func(t *testing.T) {
		servers := []string{"time", "deepwiki"}
		serversJSON, _ := json.Marshal(servers)

		excluded := []string{"time__convert_time", "deepwiki__search_wiki"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
			ExcludedTools:   datatypes.JSON(excludedJSON),
		}

		result, err := group.ResolveEffectiveTools(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveTools() failed: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 tools (5 from servers - 2 excluded), got %d", len(result))
		}

		toolMap := make(map[string]bool)
		for _, tool := range result {
			toolMap[tool] = true
		}

		// Should have these tools
		expectedTools := []string{"time__get_current_time", "time__format_time", "deepwiki__read_wiki_contents"}
		for _, expectedTool := range expectedTools {
			if !toolMap[expectedTool] {
				t.Errorf("Expected tool %s not found in result %v", expectedTool, result)
			}
		}

		// Should NOT have these tools
		unexpectedTools := []string{"time__convert_time", "deepwiki__search_wiki"}
		for _, unexpectedTool := range unexpectedTools {
			if toolMap[unexpectedTool] {
				t.Errorf("Unexpected tool %s found in result %v", unexpectedTool, result)
			}
		}
	})

	t.Run("Mixed IncludedTools and IncludedServers with ExcludedTools", func(t *testing.T) {
		tools := []string{"manual__tool1"}
		toolsJSON, _ := json.Marshal(tools)

		servers := []string{"time"}
		serversJSON, _ := json.Marshal(servers)

		excluded := []string{"time__convert_time"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedTools:   datatypes.JSON(toolsJSON),
			IncludedServers: datatypes.JSON(serversJSON),
			ExcludedTools:   datatypes.JSON(excludedJSON),
		}

		result, err := group.ResolveEffectiveTools(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveTools() failed: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 tools (1 manual + 3 from time - 1 excluded), got %d", len(result))
		}

		toolMap := make(map[string]bool)
		for _, tool := range result {
			toolMap[tool] = true
		}

		// Should have these tools
		expectedTools := []string{"manual__tool1", "time__get_current_time", "time__format_time"}
		for _, expectedTool := range expectedTools {
			if !toolMap[expectedTool] {
				t.Errorf("Expected tool %s not found in result %v", expectedTool, result)
			}
		}

		// Should NOT have this tool
		if toolMap["time__convert_time"] {
			t.Errorf("Unexpected excluded tool time__convert_time found in result %v", result)
		}
	})

	t.Run("Same tool in IncludedTools and ExcludedTools", func(t *testing.T) {
		tools := []string{"manual__tool1", "time__get_current_time"}
		toolsJSON, _ := json.Marshal(tools)

		excluded := []string{"time__get_current_time"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedTools: datatypes.JSON(toolsJSON),
			ExcludedTools: datatypes.JSON(excludedJSON),
		}

		result, err := group.ResolveEffectiveTools(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveTools() failed: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("Expected 1 tool (manual__tool1), got %d", len(result))
		}

		if result[0] != "manual__tool1" {
			t.Errorf("Expected manual__tool1, got %v", result)
		}
	})
}

func TestToolGroup_ResolveEffectiveTools_EmptyGroup(t *testing.T) {
	resolver := &mockToolResolver{
		serverTools: map[string][]Tool{},
	}

	group := &ToolGroup{}

	result, err := group.ResolveEffectiveTools(resolver)
	if err != nil {
		t.Fatalf("ResolveEffectiveTools() failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 tools for empty group, got %d", len(result))
	}
}
