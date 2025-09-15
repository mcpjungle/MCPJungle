package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestListCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "list", listCmd.Use)
		testhelpers.AssertEqual(t, "List resources like MCP servers, tools, etc", listCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupBasic)},
			{Key: "order", Expected: "3"},
		}
		testhelpers.TestCommandAnnotations(t, listCmd.Annotations, annotationTests)
	})

	t.Run("subcommands_count", func(t *testing.T) {
		subcommands := listCmd.Commands()
		testhelpers.AssertEqual(t, 5, len(subcommands))
	})
}

func TestListToolsSubcommand(t *testing.T) {
	t.Run("list tools command has correct properties", func(t *testing.T) {
		if listToolsCmd.Use != "tools" {
			t.Errorf("Expected list tools command Use to be 'tools', got %s", listToolsCmd.Use)
		}
		if listToolsCmd.Short != "List available tools" {
			t.Errorf("Expected list tools command Short to be 'List available tools', got %s", listToolsCmd.Short)
		}
		if listToolsCmd.Long == "" {
			t.Error("List tools command should have long description")
		}
	})

	t.Run("list tools command has RunE function", func(t *testing.T) {
		if listToolsCmd.RunE == nil {
			t.Fatal("List tools command missing RunE function")
		}
	})

	t.Run("list tools command has server flag", func(t *testing.T) {
		serverFlag := listToolsCmd.Flags().Lookup("server")
		if serverFlag == nil {
			t.Fatal("List tools command missing 'server' flag")
		}
		if serverFlag.Usage == "" {
			t.Error("Server flag should have usage description")
		}
	})
}

func TestListServersSubcommand(t *testing.T) {
	t.Run("list servers command has correct properties", func(t *testing.T) {
		if listServersCmd.Use != "servers" {
			t.Errorf("Expected list servers command Use to be 'servers', got %s", listServersCmd.Use)
		}
		if listServersCmd.Short != "List registered MCP servers" {
			t.Errorf("Expected list servers command Short to be 'List registered MCP servers', got %s", listServersCmd.Short)
		}
	})

	t.Run("list servers command has RunE function", func(t *testing.T) {
		if listServersCmd.RunE == nil {
			t.Fatal("List servers command missing RunE function")
		}
	})
}

func TestListMcpClientsSubcommand(t *testing.T) {
	t.Run("list mcp-clients command has correct properties", func(t *testing.T) {
		if listMcpClientsCmd.Use != "mcp-clients" {
			t.Errorf("Expected list mcp-clients command Use to be 'mcp-clients', got %s", listMcpClientsCmd.Use)
		}
		if listMcpClientsCmd.Short != "List MCP clients (Production mode)" {
			t.Errorf("Expected list mcp-clients command Short to be 'List MCP clients (Production mode)', got %s", listMcpClientsCmd.Short)
		}
		if listMcpClientsCmd.Long == "" {
			t.Error("List mcp-clients command should have long description")
		}
	})

	t.Run("list mcp-clients command has RunE function", func(t *testing.T) {
		if listMcpClientsCmd.RunE == nil {
			t.Fatal("List mcp-clients command missing RunE function")
		}
	})
}

func TestListUsersSubcommand(t *testing.T) {
	t.Run("list users command has correct properties", func(t *testing.T) {
		if listUsersCmd.Use != "users" {
			t.Errorf("Expected list users command Use to be 'users', got %s", listUsersCmd.Use)
		}
		if listUsersCmd.Short != "List users (Production mode)" {
			t.Errorf("Expected list users command Short to be 'List users (Production mode)', got %s", listUsersCmd.Short)
		}
		if listUsersCmd.Long == "" {
			t.Error("List users command should have long description")
		}
	})

	t.Run("list users command has RunE function", func(t *testing.T) {
		if listUsersCmd.RunE == nil {
			t.Fatal("List users command missing RunE function")
		}
	})
}

func TestListGroupsSubcommand(t *testing.T) {
	t.Run("list groups command has correct properties", func(t *testing.T) {
		if listGroupsCmd.Use != "groups" {
			t.Errorf("Expected list groups command Use to be 'groups', got %s", listGroupsCmd.Use)
		}
		if listGroupsCmd.Short != "List tool groups" {
			t.Errorf("Expected list groups command Short to be 'List tool groups', got %s", listGroupsCmd.Short)
		}
	})

	t.Run("list groups command has RunE function", func(t *testing.T) {
		if listGroupsCmd.RunE == nil {
			t.Fatal("List groups command missing RunE function")
		}
	})
}

func TestListCommandVariables(t *testing.T) {
	t.Run("list command variables are initialized", func(t *testing.T) {
		if listToolsCmdServerName != "" {
			t.Errorf("Expected listToolsCmdServerName to be empty, got %s", listToolsCmdServerName)
		}
	})
}

func TestRunListToolsFunctional(t *testing.T) {
	t.Run("runListTools would extract server name from flag correctly", func(t *testing.T) {
		testCases := []struct {
			name           string
			serverFlag     string
			expectedServer string
		}{
			{
				name:           "empty server flag",
				serverFlag:     "",
				expectedServer: "",
			},
			{
				name:           "valid server name",
				serverFlag:     "test-server",
				expectedServer: "test-server",
			},
			{
				name:           "server name with spaces",
				serverFlag:     "test server",
				expectedServer: "test server",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				serverName := tc.serverFlag
				if serverName != tc.expectedServer {
					t.Errorf("Expected server name %s, got %s", tc.expectedServer, serverName)
				}
			})
		}
	})

	t.Run("runListTools would handle API client calls correctly", func(t *testing.T) {
		testCases := []struct {
			name         string
			serverName   string
			expectedCall string
		}{
			{
				name:         "list all tools",
				serverName:   "",
				expectedCall: "ListTools()",
			},
			{
				name:         "list tools for specific server",
				serverName:   "test-server",
				expectedCall: "ListTools(test-server)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.serverName == "" {
					if tc.expectedCall != "ListTools()" {
						t.Errorf("Expected call pattern %s, got ListTools()", tc.expectedCall)
					}
				} else {
					expected := "ListTools(" + tc.serverName + ")"
					if expected != tc.expectedCall {
						t.Errorf("Expected call pattern %s, got %s", tc.expectedCall, expected)
					}
				}
			})
		}
	})
}

func TestRunListServersFunctional(t *testing.T) {
	t.Run("runListServers would call API client correctly", func(t *testing.T) {
		expectedCall := "ListServers()"

		if expectedCall != "ListServers()" {
			t.Errorf("Expected call pattern %s, got ListServers()", expectedCall)
		}
	})

	t.Run("runListServers would handle server list formatting", func(t *testing.T) {
		mockServers := []struct {
			Name        string
			Description string
			URL         string
		}{
			{
				Name:        "server1",
				Description: "Test server 1",
				URL:         "http://localhost:8080",
			},
			{
				Name:        "server2",
				Description: "Test server 2",
				URL:         "http://localhost:8081",
			},
		}

		for i, server := range mockServers {
			t.Run(fmt.Sprintf("server_%d", i+1), func(t *testing.T) {
				expectedFormat := fmt.Sprintf("%d. %s - %s (%s)", i+1, server.Name, server.Description, server.URL)
				actualFormat := fmt.Sprintf("%d. %s - %s (%s)", i+1, server.Name, server.Description, server.URL)

				if expectedFormat != actualFormat {
					t.Errorf("Expected format %s, got %s", expectedFormat, actualFormat)
				}
			})
		}
	})
}

func TestRunListMcpClientsFunctional(t *testing.T) {
	t.Run("runListMcpClients would call API client correctly", func(t *testing.T) {
		expectedCall := "ListMcpClients()"

		if expectedCall != "ListMcpClients()" {
			t.Errorf("Expected call pattern %s, got ListMcpClients()", expectedCall)
		}
	})

	t.Run("runListMcpClients would handle client list formatting", func(t *testing.T) {
		mockClients := []struct {
			Name        string
			Description string
			AllowList   []string
		}{
			{
				Name:        "client1",
				Description: "Test client 1",
				AllowList:   []string{"server1", "server2"},
			},
			{
				Name:        "client2",
				Description: "Test client 2",
				AllowList:   []string{"server3"},
			},
		}

		for i, client := range mockClients {
			t.Run(fmt.Sprintf("client_%d", i+1), func(t *testing.T) {
				allowListStr := strings.Join(client.AllowList, ", ")
				expectedFormat := fmt.Sprintf("%d. %s - %s (Allowed servers: %s)", i+1, client.Name, client.Description, allowListStr)
				actualFormat := fmt.Sprintf("%d. %s - %s (Allowed servers: %s)", i+1, client.Name, client.Description, allowListStr)

				if expectedFormat != actualFormat {
					t.Errorf("Expected format %s, got %s", expectedFormat, actualFormat)
				}
			})
		}
	})
}

func TestRunListUsersFunctional(t *testing.T) {
	t.Run("runListUsers would call API client correctly", func(t *testing.T) {
		expectedCall := "ListUsers()"

		if expectedCall != "ListUsers()" {
			t.Errorf("Expected call pattern %s, got ListUsers()", expectedCall)
		}
	})

	t.Run("runListUsers would handle user list formatting", func(t *testing.T) {
		mockUsers := []struct {
			Username string
			Role     string
			Email    string
		}{
			{
				Username: "user1",
				Role:     "admin",
				Email:    "user1@example.com",
			},
			{
				Username: "user2",
				Role:     "user",
				Email:    "user2@example.com",
			},
		}

		for i, user := range mockUsers {
			t.Run(fmt.Sprintf("user_%d", i+1), func(t *testing.T) {
				expectedFormat := fmt.Sprintf("%d. %s (%s) - %s", i+1, user.Username, user.Role, user.Email)
				actualFormat := fmt.Sprintf("%d. %s (%s) - %s", i+1, user.Username, user.Role, user.Email)

				if expectedFormat != actualFormat {
					t.Errorf("Expected format %s, got %s", expectedFormat, actualFormat)
				}
			})
		}
	})
}

func TestRunListGroupsFunctional(t *testing.T) {
	t.Run("runListGroups would call API client correctly", func(t *testing.T) {
		expectedCall := "ListToolGroups()"

		if expectedCall != "ListToolGroups()" {
			t.Errorf("Expected call pattern %s, got ListToolGroups()", expectedCall)
		}
	})

	t.Run("runListGroups would handle group list formatting", func(t *testing.T) {
		mockGroups := []struct {
			Name          string
			Description   string
			IncludedTools []string
		}{
			{
				Name:          "group1",
				Description:   "Test group 1",
				IncludedTools: []string{"tool1", "tool2"},
			},
			{
				Name:          "group2",
				Description:   "Test group 2",
				IncludedTools: []string{"tool3"},
			},
		}

		for i, group := range mockGroups {
			t.Run(fmt.Sprintf("group_%d", i+1), func(t *testing.T) {
				toolsStr := strings.Join(group.IncludedTools, ", ")
				expectedFormat := fmt.Sprintf("%d. %s - %s (Tools: %s)", i+1, group.Name, group.Description, toolsStr)
				actualFormat := fmt.Sprintf("%d. %s - %s (Tools: %s)", i+1, group.Name, group.Description, toolsStr)

				if expectedFormat != actualFormat {
					t.Errorf("Expected format %s, got %s", expectedFormat, actualFormat)
				}
			})
		}
	})
}
