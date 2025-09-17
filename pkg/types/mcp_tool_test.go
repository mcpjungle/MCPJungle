package types

import (
	"encoding/json"
	"testing"
)

func TestToolInputSchema(t *testing.T) {
	t.Parallel()

	// Test struct creation
	schema := ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	if schema.Type != "object" {
		t.Errorf("Expected Type to be 'object', got %s", schema.Type)
	}
	if len(schema.Properties) != 1 {
		t.Errorf("Expected Properties to have 1 item, got %d", len(schema.Properties))
	}
}

func TestToolInputSchemaJSONMarshaling(t *testing.T) {
	t.Parallel()

	schema := ToolInputSchema{
		Type: "string",
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal ToolInputSchema: %v", err)
	}

	expected := `{"type":"string"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestTool(t *testing.T) {
	t.Parallel()

	// Test struct creation
	tool := Tool{
		Name:        "test-tool",
		Enabled:     true,
		InputSchema: ToolInputSchema{Type: "object"},
	}

	if tool.Name != "test-tool" {
		t.Errorf("Expected Name to be 'test-tool', got %s", tool.Name)
	}
	if !tool.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestToolJSONMarshaling(t *testing.T) {
	t.Parallel()

	tool := Tool{
		Name:        "json-tool",
		Enabled:     false,
		Description: "Tool for JSON testing",
		InputSchema: ToolInputSchema{Type: "string"},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal Tool: %v", err)
	}

	expected := `{"name":"json-tool","enabled":false,"description":"Tool for JSON testing","input_schema":{"type":"string"}}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestToolInvokeResult(t *testing.T) {
	t.Parallel()

	// Test struct creation
	result := ToolInvokeResult{
		Meta:    map[string]any{"status": "success"},
		IsError: false,
		Content: []map[string]any{{"key": "value"}},
	}

	if result.Meta["status"] != "success" {
		t.Errorf("Expected Meta['status'] to be 'success', got %v", result.Meta["status"])
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	if len(result.Content) != 1 {
		t.Errorf("Expected Content to have 1 item, got %d", len(result.Content))
	}
}

func TestToolInvokeResultJSONMarshaling(t *testing.T) {
	t.Parallel()

	result := ToolInvokeResult{
		Meta:    map[string]any{"status": "success"},
		IsError: false,
		Content: []map[string]any{{"result": "successful"}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ToolInvokeResult: %v", err)
	}

	expected := `{"_meta":{"status":"success"},"content":[{"result":"successful"}]}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}
