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
