package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable MCP resources globally",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "3",
	},
}

var disableToolsCmd = &cobra.Command{
	Use:   "tool [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Disable one or more MCP tools globally",
	Long: "Specify the name of a tool or MCP server to disable it in the mcp proxy.\n" +
		"If a server is specified, all tools provided by that server will be disabled.\n" +
		"If a tool is disabled, it cannot be viewed or called by mcp clients.",
	RunE: runDisableTools,
}

var disablePromptsCmd = &cobra.Command{
	Use:   "prompt [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Disable one or more MCP prompts globally",
	Long: "Specify the name of a prompt or MCP server to disable it in the mcp proxy.\n" +
		"If a server is specified, all prompts provided by that server will be disabled.\n" +
		"If a prompt is disabled, it cannot be viewed or used by mcp clients.",
	RunE: runDisablePrompts,
}

func init() {
	disableCmd.AddCommand(disableToolsCmd)
	disableCmd.AddCommand(disablePromptsCmd)
	rootCmd.AddCommand(disableCmd)
}

func runDisableTools(cmd *cobra.Command, args []string) error {
	name := args[0]
	toolsDisabled, err := apiClient.DisableTools(name)
	if err != nil {
		return fmt.Errorf("failed to disable %s: %w", name, err)
	}
	if len(toolsDisabled) == 1 {
		cmd.Printf("MCP tool '%s' disabled successfully!\n", toolsDisabled[0])
		return nil
	}
	cmd.Println("Following MCP tools have been disabled successfully:")
	for _, tool := range toolsDisabled {
		cmd.Printf("- %s\n", tool)
	}
	return nil
}

func runDisablePrompts(cmd *cobra.Command, args []string) error {
	name := args[0]
	promptsDisabled, err := apiClient.DisablePrompts(name)
	if err != nil {
		return fmt.Errorf("failed to disable %s: %w", name, err)
	}
	if len(promptsDisabled) == 1 {
		cmd.Printf("MCP prompt '%s' disabled successfully!\n", promptsDisabled[0])
		return nil
	}
	cmd.Println("Following MCP prompts have been disabled successfully:")
	for _, prompt := range promptsDisabled {
		cmd.Printf("- %s\n", prompt)
	}
	return nil
}
