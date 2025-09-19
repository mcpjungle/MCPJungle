package cmd

import (
	"testing"
)

func TestUsageCommandStructure(t *testing.T) {
	t.Run("usage command has correct properties", func(t *testing.T) {
		if usageCmd.Use != "usage <name>" {
			t.Errorf("Expected usage command Use to be 'usage <name>', got %s", usageCmd.Use)
		}
		if usageCmd.Short != "Get usage information for a MCP tool" {
			t.Errorf("Expected usage command Short to be 'Get usage information for a MCP tool', got %s", usageCmd.Short)
		}
	})

	t.Run("usage command has correct annotations", func(t *testing.T) {
		if usageCmd.Annotations == nil {
			t.Fatal("Usage command missing annotations")
		}

		group, hasGroup := usageCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Usage command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected usage command group to be 'basic', got %s", group)
		}

		order, hasOrder := usageCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Usage command missing 'order' annotation")
		}
		if order != "4" {
			t.Errorf("Expected usage command order to be '4', got %s", order)
		}
	})

	t.Run("usage command has RunE function", func(t *testing.T) {
		if usageCmd.RunE == nil {
			t.Fatal("Usage command missing RunE function")
		}
	})

	t.Run("usage command requires exact args", func(t *testing.T) {
		if usageCmd.Args == nil {
			t.Fatal("Usage command missing Args validation")
		}
	})
}
