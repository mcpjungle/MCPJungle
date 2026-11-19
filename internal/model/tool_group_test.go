package model

import (
	"encoding/json"
	"fmt"
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

// errorToolResolver implements ToolResolver for testing the tolerant resolver.
// It returns the configured tools for known servers and an error for any other
// server name, simulating a server that has been deregistered.
type errorToolResolver struct {
	serverTools map[string][]Tool
}

func (m *errorToolResolver) ListToolsByServer(serverName string) ([]Tool, error) {
	if tools, exists := m.serverTools[serverName]; exists {
		return tools, nil
	}
	return nil, fmt.Errorf("server %s not found", serverName)
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

func TestToolGroup_ResolveEffectiveToolsTolerant(t *testing.T) {
	resolver := &errorToolResolver{
		serverTools: map[string][]Tool{
			"time": {
				{Name: "time__get_current_time"},
				{Name: "time__convert_time"},
			},
			"deepwiki": {
				{Name: "deepwiki__read_wiki_contents"},
				{Name: "deepwiki__search_wiki"},
			},
		},
	}

	t.Run("One server missing keeps the surviving server's tools", func(t *testing.T) {
		servers := []string{"time", "missing"}
		serversJSON, _ := json.Marshal(servers)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
		}

		result, skipped, err := group.ResolveEffectiveToolsTolerant(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveToolsTolerant() failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 tools from time server, got %d (%v)", len(result), result)
		}

		toolMap := make(map[string]bool)
		for _, tool := range result {
			toolMap[tool] = true
		}
		if !toolMap["time__get_current_time"] || !toolMap["time__convert_time"] {
			t.Errorf("Expected time tools, got %v", result)
		}

		if len(skipped) != 1 || skipped[0] != "missing" {
			t.Errorf("Expected skipped=[missing], got %v", skipped)
		}
	})

	t.Run("All servers missing yields empty tools and lists all skipped", func(t *testing.T) {
		servers := []string{"missing1", "missing2"}
		serversJSON, _ := json.Marshal(servers)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
		}

		result, skipped, err := group.ResolveEffectiveToolsTolerant(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveToolsTolerant() failed: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("Expected 0 tools when all servers are missing, got %d (%v)", len(result), result)
		}

		if len(skipped) != 2 {
			t.Errorf("Expected 2 skipped servers, got %d (%v)", len(skipped), skipped)
		}
	})

	t.Run("All servers valid matches strict resolver and reports no skips", func(t *testing.T) {
		servers := []string{"time", "deepwiki"}
		serversJSON, _ := json.Marshal(servers)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
		}

		tolerant, skipped, err := group.ResolveEffectiveToolsTolerant(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveToolsTolerant() failed: %v", err)
		}
		if len(skipped) != 0 {
			t.Errorf("Expected no skipped servers, got %v", skipped)
		}

		strict, err := group.ResolveEffectiveTools(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveTools() failed: %v", err)
		}

		if len(tolerant) != len(strict) {
			t.Errorf("Expected tolerant result (%d) to match strict result (%d)", len(tolerant), len(strict))
		}

		strictMap := make(map[string]bool)
		for _, tool := range strict {
			strictMap[tool] = true
		}
		for _, tool := range tolerant {
			if !strictMap[tool] {
				t.Errorf("Tolerant result contains tool %s not present in strict result %v", tool, strict)
			}
		}
	})

	t.Run("Excluded tools are still applied", func(t *testing.T) {
		servers := []string{"time", "missing"}
		serversJSON, _ := json.Marshal(servers)

		excluded := []string{"time__convert_time"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
			ExcludedTools:   datatypes.JSON(excludedJSON),
		}

		result, skipped, err := group.ResolveEffectiveToolsTolerant(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectiveToolsTolerant() failed: %v", err)
		}

		toolMap := make(map[string]bool)
		for _, tool := range result {
			toolMap[tool] = true
		}
		if !toolMap["time__get_current_time"] {
			t.Errorf("Expected time__get_current_time in result %v", result)
		}
		if toolMap["time__convert_time"] {
			t.Errorf("Expected excluded tool time__convert_time to be absent from result %v", result)
		}
		if len(skipped) != 1 || skipped[0] != "missing" {
			t.Errorf("Expected skipped=[missing], got %v", skipped)
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
