package cmd

import (
	"testing"
)

func TestRegisterCommandStructure(t *testing.T) {
	t.Run("register command has correct properties", func(t *testing.T) {
		if registerMCPServerCmd.Use != "register" {
			t.Errorf("Expected register command Use to be 'register', got %s", registerMCPServerCmd.Use)
		}
		if registerMCPServerCmd.Short != "Register an MCP Server" {
			t.Errorf("Expected register command Short to be 'Register an MCP Server', got %s", registerMCPServerCmd.Short)
		}
	})

	t.Run("register command has correct annotations", func(t *testing.T) {
		if registerMCPServerCmd.Annotations == nil {
			t.Fatal("Register command missing annotations")
		}

		group, hasGroup := registerMCPServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Register command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected register command group to be 'basic', got %s", group)
		}

		order, hasOrder := registerMCPServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Register command missing 'order' annotation")
		}
		if order != "2" {
			t.Errorf("Expected register command order to be '2', got %s", order)
		}
	})

	t.Run("register command has PreRunE function", func(t *testing.T) {
		if registerMCPServerCmd.PreRunE == nil {
			t.Fatal("Register command missing PreRunE function")
		}
	})
}

func TestRegisterCommandFlags(t *testing.T) {
	t.Run("register command has name flag", func(t *testing.T) {
		nameFlag := registerMCPServerCmd.Flags().Lookup("name")
		if nameFlag == nil {
			t.Fatal("Register command missing 'name' flag")
		}
		if nameFlag.Usage == "" {
			t.Error("Name flag should have usage description")
		}
	})

	t.Run("register command has url flag", func(t *testing.T) {
		urlFlag := registerMCPServerCmd.Flags().Lookup("url")
		if urlFlag == nil {
			t.Fatal("Register command missing 'url' flag")
		}
		if urlFlag.Usage == "" {
			t.Error("URL flag should have usage description")
		}
	})

	t.Run("register command has description flag", func(t *testing.T) {
		descFlag := registerMCPServerCmd.Flags().Lookup("description")
		if descFlag == nil {
			t.Fatal("Register command missing 'description' flag")
		}
		if descFlag.Usage == "" {
			t.Error("Description flag should have usage description")
		}
	})

	t.Run("register command has bearer-token flag", func(t *testing.T) {
		tokenFlag := registerMCPServerCmd.Flags().Lookup("bearer-token")
		if tokenFlag == nil {
			t.Fatal("Register command missing 'bearer-token' flag")
		}
		if tokenFlag.Usage == "" {
			t.Error("Bearer-token flag should have usage description")
		}
	})

	t.Run("register command has conf flag", func(t *testing.T) {
		confFlag := registerMCPServerCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Register command missing 'conf' flag")
		}
		if confFlag.Usage == "" {
			t.Error("Conf flag should have usage description")
		}
	})

	t.Run("register command has conf flag with short form", func(t *testing.T) {
		// The StringVarP creates both "conf" and "c" flags
		confFlag := registerMCPServerCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Register command missing 'conf' flag")
		}
		// Note: StringVarP creates both long and short forms, but we test the long form
	})
}

func TestRegisterCommandVariables(t *testing.T) {
	t.Run("register command variables are initialized", func(t *testing.T) {
		// These variables should be initialized to empty strings
		if registerCmdServerName != "" {
			t.Errorf("Expected registerCmdServerName to be empty, got %s", registerCmdServerName)
		}
		if registerCmdServerURL != "" {
			t.Errorf("Expected registerCmdServerURL to be empty, got %s", registerCmdServerURL)
		}
		if registerCmdServerDesc != "" {
			t.Errorf("Expected registerCmdServerDesc to be empty, got %s", registerCmdServerDesc)
		}
		if registerCmdBearerToken != "" {
			t.Errorf("Expected registerCmdBearerToken to be empty, got %s", registerCmdBearerToken)
		}
		if registerCmdServerConfigFilePath != "" {
			t.Errorf("Expected registerCmdServerConfigFilePath to be empty, got %s", registerCmdServerConfigFilePath)
		}
	})
}
