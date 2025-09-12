package cmd

import (
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
