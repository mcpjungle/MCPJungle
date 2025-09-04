package cmd

import (
	"testing"
)

func TestRootCommandStructure(t *testing.T) {
	if rootCmd.Use != "mcpjungle" {
		t.Errorf("Expected root command Use to be 'mcpjungle', got %s", rootCmd.Use)
	}
	if rootCmd.Short != "MCP Gateway for AI Agents" {
		t.Errorf("Expected root command Short to be 'MCP Gateway for AI Agents', got %s", rootCmd.Short)
	}
}

func TestSubCommandGroups(t *testing.T) {
	if subCommandGroupBasic != "basic" {
		t.Errorf("Expected subCommandGroupBasic to be 'basic', got %s", subCommandGroupBasic)
	}
	if subCommandGroupAdvanced != "advanced" {
		t.Errorf("Expected subCommandGroupAdvanced to be 'advanced', got %s", subCommandGroupAdvanced)
	}
}
