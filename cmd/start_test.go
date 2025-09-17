package cmd

import (
	"testing"
)

func TestStartCommandStructure(t *testing.T) {
	t.Run("start command has correct properties", func(t *testing.T) {
		if startServerCmd.Use != "start" {
			t.Errorf("Expected start command Use to be 'start', got %s", startServerCmd.Use)
		}
		if startServerCmd.Short != "Start the MCPJungle server" {
			t.Errorf("Expected start command Short to be 'Start the MCPJungle server', got %s", startServerCmd.Short)
		}
	})

	t.Run("start command has correct annotations", func(t *testing.T) {
		if startServerCmd.Annotations == nil {
			t.Fatal("Start command missing annotations")
		}

		group, hasGroup := startServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Start command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected start command group to be 'basic', got %s", group)
		}

		order, hasOrder := startServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Start command missing 'order' annotation")
		}
		if order != "1" {
			t.Errorf("Expected start command order to be '1', got %s", order)
		}
	})
}

func TestStartCommandFlags(t *testing.T) {
	t.Run("start command has port flag", func(t *testing.T) {
		portFlag := startServerCmd.Flags().Lookup("port")
		if portFlag == nil {
			t.Fatal("Start command missing 'port' flag")
		}
		if portFlag.Usage == "" {
			t.Error("Port flag should have usage description")
		}
	})

	t.Run("start command has prod flag", func(t *testing.T) {
		prodFlag := startServerCmd.Flags().Lookup("prod")
		if prodFlag == nil {
			t.Fatal("Start command missing 'prod' flag")
		}
		if prodFlag.Usage == "" {
			t.Error("Prod flag should have usage description")
		}
	})
}
