package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable MCP resources globally",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "2",
	},
}

var enableToolsCmd = &cobra.Command{
	Use:   "tool [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Enable one or more MCP tools globally",
	Long: "Specify the name of a tool or MCP server to enable it in the mcp proxy.\n" +
		"If a server is specified, all tools provided by that server will be enabled.\n" +
		"If a tool is enabled, it can be viewed and called by mcp clients.",
	RunE: runEnableTools,
}

var enablePromptsCmd = &cobra.Command{
	Use:   "prompt [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Enable one or more MCP prompts globally",
	Long: "Specify the name of a prompt or MCP server to enable it in the mcp proxy.\n" +
		"If a server is specified, all prompts provided by that server will be enabled.\n" +
		"If a prompt is enabled, it can be viewed and used by mcp clients.",
	RunE: runEnablePrompts,
}

func init() {
	enableCmd.AddCommand(enableToolsCmd)
	enableCmd.AddCommand(enablePromptsCmd)
	rootCmd.AddCommand(enableCmd)
}

func runEnableTools(cmd *cobra.Command, args []string) error {
	name := args[0]
	toolsEnabled, err := apiClient.EnableTools(name)
	if err != nil {
		return fmt.Errorf("failed to enable %s: %w", name, err)
	}
	if len(toolsEnabled) == 1 {
		cmd.Printf("MCP tool '%s' enabled successfully!\n", toolsEnabled[0])
		return nil
	}
	cmd.Println("Following MCP tools have been enabled successfully:")
	for _, tool := range toolsEnabled {
		cmd.Printf("- %s\n", tool)
	}
	return nil
}

func runEnablePrompts(cmd *cobra.Command, args []string) error {
	name := args[0]
	promptsEnabled, err := apiClient.EnablePrompts(name)
	if err != nil {
		return fmt.Errorf("failed to enable %s: %w", name, err)
	}
	if len(promptsEnabled) == 1 {
		cmd.Printf("MCP prompt '%s' enabled successfully!\n", promptsEnabled[0])
		return nil
	}
	cmd.Println("Following MCP prompts have been enabled successfully:")
	for _, prompt := range promptsEnabled {
		cmd.Printf("- %s\n", prompt)
	}
	return nil
}
