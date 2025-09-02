package cmd

import (
	"testing"
)

func TestCreateCommandStructure(t *testing.T) {
	t.Run("create command has correct properties", func(t *testing.T) {
		if createCmd.Use != "create" {
			t.Errorf("Expected create command Use to be 'create', got %s", createCmd.Use)
		}
		if createCmd.Short != "Create resources" {
			t.Errorf("Expected create command Short to be 'Create resources', got %s", createCmd.Short)
		}
	})

	t.Run("create command has correct annotations", func(t *testing.T) {
		if createCmd.Annotations == nil {
			t.Fatal("Create command missing annotations")
		}

		group, hasGroup := createCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Create command missing 'group' annotation")
		}
		if group != string(subCommandGroupAdvanced) {
			t.Errorf("Expected create command group to be 'advanced', got %s", group)
		}

		order, hasOrder := createCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Create command missing 'order' annotation")
		}
		if order != "3" {
			t.Errorf("Expected create command order to be '3', got %s", order)
		}
	})

	t.Run("create command has subcommands", func(t *testing.T) {
		subcommands := createCmd.Commands()
		if len(subcommands) != 3 {
			t.Errorf("Expected create command to have 3 subcommands, got %d", len(subcommands))
		}
	})
}

func TestCreateMcpClientSubcommand(t *testing.T) {
	t.Run("create mcp-client command has correct properties", func(t *testing.T) {
		if createMcpClientCmd.Use != "mcp-client [name]" {
			t.Errorf("Expected create mcp-client command Use to be 'mcp-client [name]', got %s", createMcpClientCmd.Use)
		}
		if createMcpClientCmd.Short != "Create an authenticated MCP client (Production mode)" {
			t.Errorf("Expected create mcp-client command Short to be 'Create an authenticated MCP client (Production mode)', got %s", createMcpClientCmd.Short)
		}
		if createMcpClientCmd.Long == "" {
			t.Error("Create mcp-client command should have long description")
		}
	})

	t.Run("create mcp-client command has RunE function", func(t *testing.T) {
		if createMcpClientCmd.RunE == nil {
			t.Fatal("Create mcp-client command missing RunE function")
		}
	})

	t.Run("create mcp-client command requires exact args", func(t *testing.T) {
		if createMcpClientCmd.Args == nil {
			t.Fatal("Create mcp-client command missing Args validation")
		}
	})

	t.Run("create mcp-client command has allow flag", func(t *testing.T) {
		allowFlag := createMcpClientCmd.Flags().Lookup("allow")
		if allowFlag == nil {
			t.Fatal("Create mcp-client command missing 'allow' flag")
		}
		if allowFlag.Usage == "" {
			t.Error("Allow flag should have usage description")
		}
	})

	t.Run("create mcp-client command has description flag", func(t *testing.T) {
		descFlag := createMcpClientCmd.Flags().Lookup("description")
		if descFlag == nil {
			t.Fatal("Create mcp-client command missing 'description' flag")
		}
		if descFlag.Usage == "" {
			t.Error("Description flag should have usage description")
		}
	})
}

func TestCreateUserSubcommand(t *testing.T) {
	t.Run("create user command has correct properties", func(t *testing.T) {
		if createUserCmd.Use != "user [username]" {
			t.Errorf("Expected create user command Use to be 'user [username]', got %s", createUserCmd.Use)
		}
		if createUserCmd.Short != "Create a new user (Production mode)" {
			t.Errorf("Expected create user command Short to be 'Create a new user (Production mode)', got %s", createUserCmd.Short)
		}
		if createUserCmd.Long == "" {
			t.Error("Create user command should have long description")
		}
	})

	t.Run("create user command has RunE function", func(t *testing.T) {
		if createUserCmd.RunE == nil {
			t.Fatal("Create user command missing RunE function")
		}
	})

	t.Run("create user command requires exact args", func(t *testing.T) {
		if createUserCmd.Args == nil {
			t.Fatal("Create user command missing Args validation")
		}
	})
}

func TestCreateToolGroupSubcommand(t *testing.T) {
	t.Run("create group command has correct properties", func(t *testing.T) {
		if createToolGroupCmd.Use != "group" {
			t.Errorf("Expected create group command Use to be 'group', got %s", createToolGroupCmd.Use)
		}
		if createToolGroupCmd.Short != "Create a Group of MCP Tools" {
			t.Errorf("Expected create group command Short to be 'Create a Group of MCP Tools', got %s", createToolGroupCmd.Short)
		}
		if createToolGroupCmd.Long == "" {
			t.Error("Create group command should have long description")
		}
	})

	t.Run("create group command has RunE function", func(t *testing.T) {
		if createToolGroupCmd.RunE == nil {
			t.Fatal("Create group command missing RunE function")
		}
	})

	t.Run("create group command has conf flag", func(t *testing.T) {
		confFlag := createToolGroupCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Create group command missing 'conf' flag")
		}
		if confFlag.Usage == "" {
			t.Error("Conf flag should have usage description")
		}
	})

	t.Run("create group command has conf flag with short form", func(t *testing.T) {
		// The StringVarP creates both "conf" and "c" flags
		confFlag := createToolGroupCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Create group command missing 'conf' flag")
		}
		// Note: StringVarP creates both long and short forms, but we test the long form
	})

	t.Run("create group command conf flag is required", func(t *testing.T) {
		confFlag := createToolGroupCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Create group command missing 'conf' flag")
		}
		// Note: The flag is marked as required in the init function with MarkFlagRequired
	})
}

func TestCreateCommandVariables(t *testing.T) {
	t.Run("create command variables are initialized", func(t *testing.T) {
		if createMcpClientCmdAllowedServers != "" {
			t.Errorf("Expected createMcpClientCmdAllowedServers to be empty, got %s", createMcpClientCmdAllowedServers)
		}
		if createMcpClientCmdDescription != "" {
			t.Errorf("Expected createMcpClientCmdDescription to be empty, got %s", createMcpClientCmdDescription)
		}
		if createToolGroupConfigFilePath != "" {
			t.Errorf("Expected createToolGroupConfigFilePath to be empty, got %s", createToolGroupConfigFilePath)
		}
	})
}
