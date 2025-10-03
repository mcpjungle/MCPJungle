package mcp

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestConvertMCPResponse(t *testing.T) {
	tests := []struct {
		name           string
		input          *mcp.CallToolResult
		expectedResult *types.ToolInvokeResult
		expectedError  string
	}{
		{
			name: "simple text content",
			input: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Hello, World!",
					},
				},
				IsError:           false,
				Result:            mcp.Result{Meta: nil},
				StructuredContent: map[string]string{"key": "value"},
			},
			expectedResult: &types.ToolInvokeResult{
				Content: []map[string]any{
					{
						"type": "text",
						"text": "Hello, World!",
					},
				},
				IsError:           false,
				Meta:              nil,
				StructuredContent: map[string]string{"key": "value"},
			},
			expectedError: "",
		},
		{
			name: "multiple content items",
			input: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "First item",
					},
					mcp.TextContent{
						Type: "text",
						Text: "Second item",
					},
				},
				IsError:           false,
				Result:            mcp.Result{Meta: nil},
				StructuredContent: []int{1, 2, 3},
			},
			expectedResult: &types.ToolInvokeResult{
				Content: []map[string]any{
					{
						"type": "text",
						"text": "First item",
					},
					{
						"type": "text",
						"text": "Second item",
					},
				},
				IsError:           false,
				Meta:              nil,
				StructuredContent: []int{1, 2, 3},
			},
			expectedError: "",
		},
		{
			name: "error response",
			input: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error occurred",
					},
				},
				IsError:           true,
				Result:            mcp.Result{Meta: nil},
				StructuredContent: nil,
			},
			expectedResult: &types.ToolInvokeResult{
				Content: []map[string]any{
					{
						"type": "text",
						"text": "Error occurred",
					},
				},
				IsError:           true,
				Meta:              nil,
				StructuredContent: nil,
			},
			expectedError: "",
		},
		{
			name: "empty content",
			input: &mcp.CallToolResult{
				Content:           []mcp.Content{},
				IsError:           false,
				Result:            mcp.Result{Meta: nil},
				StructuredContent: true,
			},
			expectedResult: &types.ToolInvokeResult{
				Content:           []map[string]any{},
				IsError:           false,
				Meta:              nil,
				StructuredContent: true,
			},
			expectedError: "",
		},
		{
			name: "with meta - progress token only",
			input: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Content with meta",
					},
				},
				IsError: false,
				Result: mcp.Result{
					Meta: &mcp.Meta{
						ProgressToken: "token123",
					},
				},
				StructuredContent: "Hello world",
			},
			expectedResult: &types.ToolInvokeResult{
				Content: []map[string]any{
					{
						"type": "text",
						"text": "Content with meta",
					},
				},
				IsError: false,
				Meta: map[string]any{
					"progressToken": "token123",
				},
				StructuredContent: "Hello world",
			},
			expectedError: "",
		},
		{
			name: "with meta - additional fields",
			input: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Content with meta fields",
					},
				},
				IsError: false,
				Result: mcp.Result{
					Meta: &mcp.Meta{
						AdditionalFields: map[string]any{
							"custom_field": "custom_value",
							"number_field": 42,
						},
					},
				},
			},
			expectedResult: &types.ToolInvokeResult{
				Content: []map[string]any{
					{
						"type": "text",
						"text": "Content with meta fields",
					},
				},
				IsError: false,
				Meta: map[string]any{
					"custom_field": "custom_value",
					"number_field": 42,
				},
			},
			expectedError: "",
		},
		{
			name: "with meta - both progress token and additional fields",
			input: &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Content with full meta",
					},
				},
				IsError: false,
				Result: mcp.Result{
					Meta: &mcp.Meta{
						ProgressToken: "token456",
						AdditionalFields: map[string]any{
							"source":   "test",
							"priority": 1,
						},
					},
				},
			},
			expectedResult: &types.ToolInvokeResult{
				Content: []map[string]any{
					{
						"type": "text",
						"text": "Content with full meta",
					},
				},
				IsError: false,
				Meta: map[string]any{
					"progressToken": "token456",
					"source":        "test",
					"priority":      1,
				},
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MCPService{}
			result, err := m.convertToolCallResToAPIRes(tt.input)

			// Check error expectations
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, but got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
				return
			}

			// Check result
			if result.IsError != tt.expectedResult.IsError {
				t.Errorf("Expected IsError %v, got %v", tt.expectedResult.IsError, result.IsError)
			}

			if !reflect.DeepEqual(result.StructuredContent, tt.expectedResult.StructuredContent) {
				expectedJSON, _ := json.Marshal(tt.expectedResult.StructuredContent)
				actualJSON, _ := json.Marshal(result.StructuredContent)
				t.Errorf("Expected StructuredContent %s, got %s", expectedJSON, actualJSON)
			}

			// Check content length
			if len(result.Content) != len(tt.expectedResult.Content) {
				t.Errorf("Expected content length %d, got %d", len(tt.expectedResult.Content), len(result.Content))
				return
			}

			// Check content items
			for i, expectedContent := range tt.expectedResult.Content {
				actualContent := result.Content[i]

				// Compare as JSON to handle nested structures
				expectedJSON, _ := json.Marshal(expectedContent)
				actualJSON, _ := json.Marshal(actualContent)

				if string(expectedJSON) != string(actualJSON) {
					t.Errorf("Content[%d] mismatch:\nExpected: %s\nActual: %s", i, expectedJSON, actualJSON)
				}
			}

			// Check meta
			if tt.expectedResult.Meta == nil && result.Meta != nil {
				t.Errorf("Expected nil meta, got %v", result.Meta)
			} else if tt.expectedResult.Meta != nil && result.Meta == nil {
				t.Errorf("Expected meta %v, got nil", tt.expectedResult.Meta)
			} else if tt.expectedResult.Meta != nil && result.Meta != nil {
				expectedMetaJSON, _ := json.Marshal(tt.expectedResult.Meta)
				actualMetaJSON, _ := json.Marshal(result.Meta)

				if string(expectedMetaJSON) != string(actualMetaJSON) {
					t.Errorf("Meta mismatch:\nExpected: %s\nActual: %s", expectedMetaJSON, actualMetaJSON)
				}
			}
		})
	}
}

func TestConvertMCPContent(t *testing.T) {
	tests := []struct {
		name           string
		input          []mcp.Content
		expectedResult []map[string]any
		expectedError  string
	}{
		{
			name:           "empty content",
			input:          []mcp.Content{},
			expectedResult: []map[string]any{},
			expectedError:  "",
		},
		{
			name: "single text content",
			input: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Test content",
				},
			},
			expectedResult: []map[string]any{
				{
					"type": "text",
					"text": "Test content",
				},
			},
			expectedError: "",
		},
		{
			name: "multiple different content types",
			input: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Text content",
				},
				mcp.ImageContent{
					Type:     "image",
					Data:     "base64data",
					MIMEType: "image/png",
				},
			},
			expectedResult: []map[string]any{
				{
					"type": "text",
					"text": "Text content",
				},
				{
					"type":     "image",
					"data":     "base64data",
					"mimeType": "image/png",
				},
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MCPService{}
			result, err := m.convertToolCallRespContent(tt.input)

			// Check error expectations
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, but got nil", tt.expectedError)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
				return
			}

			// Check result length
			if len(result) != len(tt.expectedResult) {
				t.Errorf("Expected length %d, got %d", len(tt.expectedResult), len(result))
				return
			}

			// Check each item
			for i, expected := range tt.expectedResult {
				actual := result[i]

				expectedJSON, _ := json.Marshal(expected)
				actualJSON, _ := json.Marshal(actual)

				if string(expectedJSON) != string(actualJSON) {
					t.Errorf("Item[%d] mismatch:\nExpected: %s\nActual: %s", i, expectedJSON, actualJSON)
				}
			}
		})
	}
}

func TestConvertMCPMeta(t *testing.T) {
	tests := []struct {
		name           string
		input          *mcp.Meta
		expectedResult map[string]any
	}{
		{
			name:           "nil meta",
			input:          nil,
			expectedResult: nil,
		},
		{
			name:           "empty meta",
			input:          &mcp.Meta{},
			expectedResult: nil,
		},
		{
			name: "progress token only",
			input: &mcp.Meta{
				ProgressToken: "test-token",
			},
			expectedResult: map[string]any{
				"progressToken": "test-token",
			},
		},
		{
			name: "additional fields only",
			input: &mcp.Meta{
				AdditionalFields: map[string]any{
					"key1": "value1",
					"key2": 42,
				},
			},
			expectedResult: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
		},
		{
			name: "both progress token and additional fields",
			input: &mcp.Meta{
				ProgressToken: "token123",
				AdditionalFields: map[string]any{
					"custom": "data",
					"count":  10,
				},
			},
			expectedResult: map[string]any{
				"progressToken": "token123",
				"custom":        "data",
				"count":         10,
			},
		},
		{
			name: "additional fields with nil map",
			input: &mcp.Meta{
				ProgressToken:    "token456",
				AdditionalFields: nil,
			},
			expectedResult: map[string]any{
				"progressToken": "token456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MCPService{}
			result := m.convertMCPMetaToMap(tt.input)

			if tt.expectedResult == nil && result != nil {
				t.Errorf("Expected nil result, got %v", result)
			} else if tt.expectedResult != nil && result == nil {
				t.Errorf("Expected result %v, got nil", tt.expectedResult)
			} else if tt.expectedResult != nil && result != nil {
				expectedJSON, _ := json.Marshal(tt.expectedResult)
				actualJSON, _ := json.Marshal(result)

				if string(expectedJSON) != string(actualJSON) {
					t.Errorf("Result mismatch:\nExpected: %s\nActual: %s", expectedJSON, actualJSON)
				}
			}
		})
	}
}
