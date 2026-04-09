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

// mockPromptResolver implements PromptResolver for testing
type mockPromptResolver struct {
	serverPrompts map[string][]Prompt
}

func (m *mockPromptResolver) ListPromptsByServer(serverName string) ([]Prompt, error) {
	if prompts, exists := m.serverPrompts[serverName]; exists {
		return prompts, nil
	}
	return []Prompt{}, nil
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

func TestToolGroup_GetPrompts(t *testing.T) {
	t.Run("with prompts", func(t *testing.T) {
		prompts := []string{"prompt1", "prompt2"}
		promptsJSON, _ := json.Marshal(prompts)

		group := &ToolGroup{
			IncludedPrompts: datatypes.JSON(promptsJSON),
		}

		result, err := group.GetPrompts()
		if err != nil {
			t.Fatalf("GetPrompts() failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(result))
		}
		if result[0] != "prompt1" || result[1] != "prompt2" {
			t.Errorf("Expected [prompt1, prompt2], got %v", result)
		}
	})

	t.Run("nil returns empty slice", func(t *testing.T) {
		group := &ToolGroup{}

		result, err := group.GetPrompts()
		if err != nil {
			t.Fatalf("GetPrompts() failed: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected 0 prompts, got %d", len(result))
		}
	})
}

func TestToolGroup_GetExcludedPrompts(t *testing.T) {
	t.Run("with excluded prompts", func(t *testing.T) {
		prompts := []string{"excluded1", "excluded2"}
		promptsJSON, _ := json.Marshal(prompts)

		group := &ToolGroup{
			ExcludedPrompts: datatypes.JSON(promptsJSON),
		}

		result, err := group.GetExcludedPrompts()
		if err != nil {
			t.Fatalf("GetExcludedPrompts() failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 excluded prompts, got %d", len(result))
		}
		if result[0] != "excluded1" || result[1] != "excluded2" {
			t.Errorf("Expected [excluded1, excluded2], got %v", result)
		}
	})

	t.Run("nil returns empty slice", func(t *testing.T) {
		group := &ToolGroup{}

		result, err := group.GetExcludedPrompts()
		if err != nil {
			t.Fatalf("GetExcludedPrompts() failed: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected 0 excluded prompts, got %d", len(result))
		}
	})
}

func TestToolGroup_ResolveEffectivePrompts(t *testing.T) {
	resolver := &mockPromptResolver{
		serverPrompts: map[string][]Prompt{
			"github": {
				{Name: "github__code-review"},
				{Name: "github__security-audit"},
				{Name: "github__summarize"},
			},
			"slack": {
				{Name: "slack__compose-message"},
				{Name: "slack__daily-standup"},
			},
		},
	}

	t.Run("IncludedPrompts only", func(t *testing.T) {
		prompts := []string{"manual__prompt1", "manual__prompt2"}
		promptsJSON, _ := json.Marshal(prompts)

		group := &ToolGroup{
			IncludedPrompts: datatypes.JSON(promptsJSON),
		}

		result, err := group.ResolveEffectivePrompts(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectivePrompts() failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(result))
		}

		promptMap := make(map[string]bool)
		for _, p := range result {
			promptMap[p] = true
		}
		if !promptMap["manual__prompt1"] || !promptMap["manual__prompt2"] {
			t.Errorf("Expected manual prompts, got %v", result)
		}
	})

	t.Run("IncludedServers only", func(t *testing.T) {
		servers := []string{"github"}
		serversJSON, _ := json.Marshal(servers)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
		}

		result, err := group.ResolveEffectivePrompts(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectivePrompts() failed: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 prompts from github server, got %d", len(result))
		}

		promptMap := make(map[string]bool)
		for _, p := range result {
			promptMap[p] = true
		}
		expected := []string{"github__code-review", "github__security-audit", "github__summarize"}
		for _, e := range expected {
			if !promptMap[e] {
				t.Errorf("Expected prompt %s not found in result %v", e, result)
			}
		}
	})

	t.Run("IncludedServers with ExcludedPrompts", func(t *testing.T) {
		servers := []string{"github", "slack"}
		serversJSON, _ := json.Marshal(servers)

		excluded := []string{"github__summarize", "slack__daily-standup"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedServers: datatypes.JSON(serversJSON),
			ExcludedPrompts: datatypes.JSON(excludedJSON),
		}

		result, err := group.ResolveEffectivePrompts(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectivePrompts() failed: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 prompts (5 from servers - 2 excluded), got %d", len(result))
		}

		promptMap := make(map[string]bool)
		for _, p := range result {
			promptMap[p] = true
		}

		expected := []string{"github__code-review", "github__security-audit", "slack__compose-message"}
		for _, e := range expected {
			if !promptMap[e] {
				t.Errorf("Expected prompt %s not found in result %v", e, result)
			}
		}

		unexpected := []string{"github__summarize", "slack__daily-standup"}
		for _, u := range unexpected {
			if promptMap[u] {
				t.Errorf("Unexpected excluded prompt %s found in result %v", u, result)
			}
		}
	})

	t.Run("Mixed IncludedPrompts and IncludedServers with ExcludedPrompts", func(t *testing.T) {
		prompts := []string{"manual__custom-prompt"}
		promptsJSON, _ := json.Marshal(prompts)

		servers := []string{"slack"}
		serversJSON, _ := json.Marshal(servers)

		excluded := []string{"slack__daily-standup"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedPrompts: datatypes.JSON(promptsJSON),
			IncludedServers: datatypes.JSON(serversJSON),
			ExcludedPrompts: datatypes.JSON(excludedJSON),
		}

		result, err := group.ResolveEffectivePrompts(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectivePrompts() failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 prompts (1 manual + 2 from slack - 1 excluded), got %d", len(result))
		}

		promptMap := make(map[string]bool)
		for _, p := range result {
			promptMap[p] = true
		}

		if !promptMap["manual__custom-prompt"] || !promptMap["slack__compose-message"] {
			t.Errorf("Expected manual and slack prompts, got %v", result)
		}
		if promptMap["slack__daily-standup"] {
			t.Errorf("Unexpected excluded prompt slack__daily-standup found in result %v", result)
		}
	})

	t.Run("Same prompt in IncludedPrompts and ExcludedPrompts", func(t *testing.T) {
		prompts := []string{"manual__prompt1", "github__code-review"}
		promptsJSON, _ := json.Marshal(prompts)

		excluded := []string{"github__code-review"}
		excludedJSON, _ := json.Marshal(excluded)

		group := &ToolGroup{
			IncludedPrompts: datatypes.JSON(promptsJSON),
			ExcludedPrompts: datatypes.JSON(excludedJSON),
		}

		result, err := group.ResolveEffectivePrompts(resolver)
		if err != nil {
			t.Fatalf("ResolveEffectivePrompts() failed: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("Expected 1 prompt (manual__prompt1), got %d", len(result))
		}
		if result[0] != "manual__prompt1" {
			t.Errorf("Expected manual__prompt1, got %v", result)
		}
	})
}

func TestToolGroup_ResolveEffectivePrompts_EmptyGroup(t *testing.T) {
	resolver := &mockPromptResolver{
		serverPrompts: map[string][]Prompt{},
	}

	group := &ToolGroup{}

	result, err := group.ResolveEffectivePrompts(resolver)
	if err != nil {
		t.Fatalf("ResolveEffectivePrompts() failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 prompts for empty group, got %d", len(result))
	}
}
