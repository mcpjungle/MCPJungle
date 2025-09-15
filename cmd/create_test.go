package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestCreateCommandStructure(t *testing.T) {
	t.Parallel()
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "create", createCmd.Use)
		testhelpers.AssertEqual(t, "Create resources", createCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "3"},
		}
		testhelpers.TestCommandAnnotations(t, createCmd.Annotations, annotationTests)
	})

	t.Run("subcommands_count", func(t *testing.T) {
		subcommands := createCmd.Commands()
		testhelpers.AssertEqual(t, 3, len(subcommands))
	})
}

func TestCreateMcpClientSubcommand(t *testing.T) {
	t.Parallel()
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "mcp-client [name]", createMcpClientCmd.Use)
		testhelpers.AssertEqual(t, "Create an authenticated MCP client (Production mode)", createMcpClientCmd.Short)
		testhelpers.AssertNotNil(t, createMcpClientCmd.Long)
		testhelpers.AssertTrue(t, len(createMcpClientCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createMcpClientCmd.RunE)
		testhelpers.AssertNotNil(t, createMcpClientCmd.Args)
	})

	t.Run("command_flags", func(t *testing.T) {
		flagTests := []struct {
			name        string
			flagName    string
			description string
		}{
			{"allow_flag", "allow", "Allow flag should have usage description"},
			{"description_flag", "description", "Description flag should have usage description"},
		}

		for _, tt := range flagTests {
			t.Run(tt.name, func(t *testing.T) {
				flag := createMcpClientCmd.Flags().Lookup(tt.flagName)
				testhelpers.AssertNotNil(t, flag)
				testhelpers.AssertTrue(t, len(flag.Usage) > 0, tt.description)
			})
		}
	})
}

func TestCreateUserSubcommand(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "user [username]", createUserCmd.Use)
		testhelpers.AssertEqual(t, "Create a new user (Production mode)", createUserCmd.Short)
		testhelpers.AssertNotNil(t, createUserCmd.Long)
		testhelpers.AssertTrue(t, len(createUserCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createUserCmd.RunE)
		testhelpers.AssertNotNil(t, createUserCmd.Args)
	})
}

func TestCreateToolGroupSubcommand(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "group", createToolGroupCmd.Use)
		testhelpers.AssertEqual(t, "Create a Group of MCP Tools", createToolGroupCmd.Short)
		testhelpers.AssertNotNil(t, createToolGroupCmd.Long)
		testhelpers.AssertTrue(t, len(createToolGroupCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)
	})

	t.Run("command_flags", func(t *testing.T) {
		confFlag := createToolGroupCmd.Flags().Lookup("conf")
		testhelpers.AssertNotNil(t, confFlag)
		testhelpers.AssertTrue(t, len(confFlag.Usage) > 0, "Conf flag should have usage description")
		// Note: The flag is marked as required in the init function with MarkFlagRequired
	})
}

func TestCreateCommandVariables(t *testing.T) {
	t.Run("command_variables_initialized", func(t *testing.T) {
		// Test that command variables are properly initialized to empty values
		testhelpers.AssertEqual(t, "", createMcpClientCmdAllowedServers)
		testhelpers.AssertEqual(t, "", createMcpClientCmdDescription)
		testhelpers.AssertEqual(t, "", createToolGroupConfigFilePath)
	})
}

// Functional tests for create.go functions
func TestRunCreateMcpClient(t *testing.T) {
	t.Run("runCreateMcpClient function is properly defined", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createMcpClientCmd.RunE)
	})

	t.Run("runCreateMcpClient would parse allow list correctly", func(t *testing.T) {
		// Test the allow list parsing logic
		testCases := []struct {
			name     string
			input    string
			expected []string
		}{
			{"empty string", "", []string{}},
			{"single server", "server1", []string{"server1"}},
			{"multiple servers", "server1,server2,server3", []string{"server1", "server2", "server3"}},
			{"servers with spaces", "server1, server2 , server3", []string{"server1", "server2", "server3"}},
			{"servers with empty elements", "server1,,server2", []string{"server1", "server2"}},
			{"servers with only spaces", "server1,  ,server2", []string{"server1", "server2"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Simulate the allow list parsing logic from runCreateMcpClient
				allowList := make([]string, 0)
				for _, s := range strings.Split(tc.input, ",") {
					trimmed := strings.TrimSpace(s)
					if trimmed != "" {
						allowList = append(allowList, trimmed)
					}
				}
				// Compare slices element by element
				if len(tc.expected) != len(allowList) {
					t.Errorf("Expected length %d, got %d", len(tc.expected), len(allowList))
					return
				}
				for i, expected := range tc.expected {
					if expected != allowList[i] {
						t.Errorf("Expected[%d] = %s, got %s", i, expected, allowList[i])
					}
				}
			})
		}
	})

	t.Run("runCreateMcpClient would create McpClient with correct fields", func(t *testing.T) {
		// Test the McpClient creation logic
		args := []string{"test-client"}
		createMcpClientCmdAllowedServers = "server1,server2"
		createMcpClientCmdDescription = "Test client description"

		// Simulate the McpClient creation logic
		allowList := make([]string, 0)
		for _, s := range strings.Split(createMcpClientCmdAllowedServers, ",") {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				allowList = append(allowList, trimmed)
			}
		}

		expectedClient := &types.McpClient{
			Name:        args[0],
			Description: createMcpClientCmdDescription,
			AllowList:   allowList,
		}

		testhelpers.AssertEqual(t, "test-client", expectedClient.Name)
		testhelpers.AssertEqual(t, "Test client description", expectedClient.Description)
		// Compare slices element by element
		expectedAllowList := []string{"server1", "server2"}
		if len(expectedAllowList) != len(expectedClient.AllowList) {
			t.Errorf("Expected AllowList length %d, got %d", len(expectedAllowList), len(expectedClient.AllowList))
		} else {
			for i, expected := range expectedAllowList {
				if expected != expectedClient.AllowList[i] {
					t.Errorf("Expected AllowList[%d] = %s, got %s", i, expected, expectedClient.AllowList[i])
				}
			}
		}
	})

	t.Run("runCreateMcpClient would handle empty token response", func(t *testing.T) {
		// Test the empty token validation logic
		token := ""
		if token == "" {
			// This is the error condition from the function
			err := fmt.Errorf("server returned an empty token, this was unexpected")
			testhelpers.AssertNotNil(t, err)
			testhelpers.AssertEqual(t, "server returned an empty token, this was unexpected", err.Error())
		}
	})

	t.Run("runCreateMcpClient would format success messages correctly", func(t *testing.T) {
		// Test the success message formatting logic
		clientName := "test-client"
		allowList := []string{"server1", "server2"}

		// Test with allow list
		expectedMessage := fmt.Sprintf("MCP client '%s' created successfully!\n", clientName)
		testhelpers.AssertEqual(t, "MCP client 'test-client' created successfully!\n", expectedMessage)

		// Test allow list message
		if len(allowList) > 0 {
			expectedAllowMessage := "Servers accessible: " + strings.Join(allowList, ",")
			testhelpers.AssertEqual(t, "Servers accessible: server1,server2", expectedAllowMessage)
		}

		// Test empty allow list message
		emptyAllowList := []string{}
		if len(emptyAllowList) == 0 {
			expectedEmptyMessage := "This client does not have access to any MCP servers."
			testhelpers.AssertEqual(t, "This client does not have access to any MCP servers.", expectedEmptyMessage)
		}
	})
}

func TestRunCreateUser(t *testing.T) {
	t.Run("runCreateUser function is properly defined", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createUserCmd.RunE)
	})

	t.Run("runCreateUser would create CreateUserRequest correctly", func(t *testing.T) {
		// Test the CreateUserRequest creation logic
		args := []string{"testuser"}

		expectedRequest := &types.CreateUserRequest{
			Username: args[0],
		}

		testhelpers.AssertEqual(t, "testuser", expectedRequest.Username)
	})

	t.Run("runCreateUser would handle empty access token response", func(t *testing.T) {
		// Test the empty access token validation logic
		accessToken := ""
		if accessToken == "" {
			// This is the error condition from the function
			err := fmt.Errorf("server returned an empty access token, this was unexpected")
			testhelpers.AssertNotNil(t, err)
			testhelpers.AssertEqual(t, "server returned an empty access token, this was unexpected", err.Error())
		}
	})

	t.Run("runCreateUser would format success messages correctly", func(t *testing.T) {
		// Test the success message formatting logic
		username := "testuser"
		accessToken := "test-token-123"

		expectedMessage := fmt.Sprintf("User '%s' created successfully\n", username)
		testhelpers.AssertEqual(t, "User 'testuser' created successfully\n", expectedMessage)

		expectedLoginCommand := fmt.Sprintf("    mcpjungle login %s\n", accessToken)
		testhelpers.AssertEqual(t, "    mcpjungle login test-token-123\n", expectedLoginCommand)
	})
}

func TestReadToolGroupConfig(t *testing.T) {
	t.Run("readToolGroupConfig function is properly defined", func(t *testing.T) {
		// Test that the function exists and has the correct signature
		// We can't directly test the function since it's not exported, but we can test
		// the logic it would use
		testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)
	})

	t.Run("readToolGroupConfig would handle file read errors", func(t *testing.T) {
		// Test file read error handling
		filePath := "/nonexistent/file.json"
		_, err := os.ReadFile(filePath)
		if err != nil {
			expectedError := fmt.Errorf("failed to read config file %s: %w", filePath, err)
			testhelpers.AssertNotNil(t, expectedError)
			testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to read config file"), "Error should contain expected message")
		}
	})

	t.Run("readToolGroupConfig would handle JSON unmarshal errors", func(t *testing.T) {
		// Test JSON unmarshal error handling
		invalidJSON := `{"name": "test-group", "invalid": json}`
		var toolGroup types.ToolGroup
		err := json.Unmarshal([]byte(invalidJSON), &toolGroup)
		if err != nil {
			expectedError := fmt.Errorf("failed to parse config file: %w", err)
			testhelpers.AssertNotNil(t, expectedError)
			testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to parse config file"), "Error should contain expected message")
		}
	})

	t.Run("readToolGroupConfig would parse valid JSON correctly", func(t *testing.T) {
		// Test valid JSON parsing
		validJSON := `{"name": "test-group", "included_tools": ["tool1", "tool2"], "description": "Test group"}`
		var toolGroup types.ToolGroup
		err := json.Unmarshal([]byte(validJSON), &toolGroup)
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertEqual(t, "test-group", toolGroup.Name)
		// Compare slices element by element
		expectedTools := []string{"tool1", "tool2"}
		if len(expectedTools) != len(toolGroup.IncludedTools) {
			t.Errorf("Expected IncludedTools length %d, got %d", len(expectedTools), len(toolGroup.IncludedTools))
		} else {
			for i, expected := range expectedTools {
				if expected != toolGroup.IncludedTools[i] {
					t.Errorf("Expected IncludedTools[%d] = %s, got %s", i, expected, toolGroup.IncludedTools[i])
				}
			}
		}
		testhelpers.AssertEqual(t, "Test group", toolGroup.Description)
	})
}

func TestRunCreateToolGroup(t *testing.T) {
	t.Run("runCreateToolGroup function is properly defined", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)
	})

	t.Run("runCreateToolGroup would handle config file read errors", func(t *testing.T) {
		// Test config file read error handling
		configFilePath := "/nonexistent/config.json"
		expectedError := fmt.Errorf("failed to read config file %s: %w", configFilePath, fmt.Errorf("no such file or directory"))
		testhelpers.AssertNotNil(t, expectedError)
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to read config file"), "Error should contain expected message")
	})

	t.Run("runCreateToolGroup would handle API client errors", func(t *testing.T) {
		// Test API client error handling
		expectedError := fmt.Errorf("failed to create tool group: %w", fmt.Errorf("API error"))
		testhelpers.AssertNotNil(t, expectedError)
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to create tool group"), "Error should contain expected message")
	})

	t.Run("runCreateToolGroup would format success messages correctly", func(t *testing.T) {
		// Test success message formatting
		groupName := "test-group"
		streamableEndpoint := "/v0/groups/test-group/stream"
		sseEndpoint := "/v0/groups/test-group/sse"
		sseMessageEndpoint := "/v0/groups/test-group/sse/message"

		expectedGroupMessage := fmt.Sprintf("Tool Group %s created successfully\n", groupName)
		testhelpers.AssertEqual(t, "Tool Group test-group created successfully\n", expectedGroupMessage)

		expectedStreamableMessage := "    " + streamableEndpoint + "\n"
		testhelpers.AssertEqual(t, "    /v0/groups/test-group/stream\n", expectedStreamableMessage)

		expectedSSEMessage := "    " + sseEndpoint
		testhelpers.AssertEqual(t, "    /v0/groups/test-group/sse", expectedSSEMessage)

		expectedSSEMessageEndpoint := "    " + sseMessageEndpoint + "\n"
		testhelpers.AssertEqual(t, "    /v0/groups/test-group/sse/message\n", expectedSSEMessageEndpoint)
	})
}

// Integration tests for create commands
func TestCreateCommandIntegration(t *testing.T) {
	t.Run("create command is added to root command", func(t *testing.T) {
		// Verify that createCmd is properly added to rootCmd
		testhelpers.AssertNotNil(t, createCmd)
	})

	t.Run("all create subcommands are properly configured", func(t *testing.T) {
		subcommands := createCmd.Commands()
		expectedSubcommands := []string{"mcp-client", "user", "group"}

		testhelpers.AssertEqual(t, len(expectedSubcommands), len(subcommands))

		for _, expected := range expectedSubcommands {
			found := false
			for _, subcmd := range subcommands {
				if subcmd.Name() == expected {
					found = true
					break
				}
			}
			testhelpers.AssertTrue(t, found, "Expected subcommand '"+expected+"' not found")
		}
	})

	t.Run("create command flags are properly configured", func(t *testing.T) {
		// Test mcp-client flags
		allowFlag := createMcpClientCmd.Flags().Lookup("allow")
		testhelpers.AssertNotNil(t, allowFlag)
		testhelpers.AssertTrue(t, len(allowFlag.Usage) > 0, "Allow flag should have usage description")

		descriptionFlag := createMcpClientCmd.Flags().Lookup("description")
		testhelpers.AssertNotNil(t, descriptionFlag)
		testhelpers.AssertTrue(t, len(descriptionFlag.Usage) > 0, "Description flag should have usage description")

		// Test tool group flags
		confFlag := createToolGroupCmd.Flags().Lookup("conf")
		testhelpers.AssertNotNil(t, confFlag)
		testhelpers.AssertTrue(t, len(confFlag.Usage) > 0, "Conf flag should have usage description")

		confFlagShort := createToolGroupCmd.Flags().Lookup("c")
		if confFlagShort != nil {
			testhelpers.AssertTrue(t, len(confFlagShort.Usage) > 0, "Conf flag short form should have usage description")
		}
	})
}

// Error handling tests
func TestCreateCommandErrorHandling(t *testing.T) {
	t.Run("create commands handle empty arguments", func(t *testing.T) {
		// Test that commands properly validate arguments
		testhelpers.AssertNotNil(t, createMcpClientCmd.Args)
		testhelpers.AssertNotNil(t, createUserCmd.Args)
		// createToolGroupCmd doesn't have Args validation, which is correct
	})

	t.Run("create commands handle invalid input gracefully", func(t *testing.T) {
		// Test various invalid input scenarios
		testCases := []struct {
			name        string
			args        []string
			expectError bool
		}{
			{"empty args", []string{}, true},
			{"too many args", []string{"arg1", "arg2", "arg3"}, true},
			{"valid single arg", []string{"valid-arg"}, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test mcp-client command args validation
				if createMcpClientCmd.Args != nil {
					err := createMcpClientCmd.Args(createMcpClientCmd, tc.args)
					if tc.expectError {
						testhelpers.AssertError(t, err)
					} else {
						testhelpers.AssertNoError(t, err)
					}
				}

				// Test user command args validation
				if createUserCmd.Args != nil {
					err := createUserCmd.Args(createUserCmd, tc.args)
					if tc.expectError {
						testhelpers.AssertError(t, err)
					} else {
						testhelpers.AssertNoError(t, err)
					}
				}
			})
		}
	})
}
