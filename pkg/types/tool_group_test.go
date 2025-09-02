package types

import (
	"encoding/json"
	"testing"
)

func TestToolGroup(t *testing.T) {
	t.Run("ToolGroup struct creation", func(t *testing.T) {
		group := ToolGroup{
			Name:          "test-group",
			IncludedTools: []string{"tool1", "tool2"},
		}

		if group.Name != "test-group" {
			t.Errorf("Expected Name to be 'test-group', got %s", group.Name)
		}
		if len(group.IncludedTools) != 2 {
			t.Errorf("Expected IncludedTools to have 2 items, got %d", len(group.IncludedTools))
		}
	})

	t.Run("ToolGroup JSON marshaling", func(t *testing.T) {
		group := ToolGroup{
			Name:          "json-group",
			IncludedTools: []string{"json-tool1"},
			Description:   "Group for JSON testing",
		}

		data, err := json.Marshal(group)
		if err != nil {
			t.Fatalf("Failed to marshal ToolGroup: %v", err)
		}

		expected := `{"name":"json-group","included_tools":["json-tool1"],"description":"Group for JSON testing"}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})
}

func TestCreateToolGroupResponse(t *testing.T) {
	t.Run("CreateToolGroupResponse struct creation", func(t *testing.T) {
		response := CreateToolGroupResponse{
			Endpoint: "/api/tool-groups/test-group",
		}

		if response.Endpoint != "/api/tool-groups/test-group" {
			t.Errorf("Expected Endpoint to be '/api/tool-groups/test-group', got %s", response.Endpoint)
		}
	})

	t.Run("CreateToolGroupResponse JSON marshaling", func(t *testing.T) {
		response := CreateToolGroupResponse{
			Endpoint: "/api/tool-groups/json-group",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CreateToolGroupResponse: %v", err)
		}

		expected := `{"endpoint":"/api/tool-groups/json-group"}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})
}

func TestGetToolGroupResponse(t *testing.T) {
	t.Run("GetToolGroupResponse struct creation", func(t *testing.T) {
		toolGroup := &ToolGroup{
			Name:          "get-group",
			IncludedTools: []string{"get-tool1"},
			Description:   "Group for get testing",
		}

		response := GetToolGroupResponse{
			ToolGroup: toolGroup,
			Endpoint:  "/api/tool-groups/get-group",
		}

		if response.ToolGroup != toolGroup {
			t.Error("Expected ToolGroup pointer to match")
		}
		if response.Endpoint != "/api/tool-groups/get-group" {
			t.Errorf("Expected Endpoint to be '/api/tool-groups/get-group', got %s", response.Endpoint)
		}
	})
}
