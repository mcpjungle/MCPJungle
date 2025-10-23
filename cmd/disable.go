package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

/*
   Usage:
     disable [servername]
       (Deprecated, for backward-compatibility) Disable all tools from a mcp server
     disable [toolname]
       (Deprecated, for backward compatibility) Disable a specific mcp tool
     disable tool [servername]
       Disable all tools from a mcp server
     disable tool [toolname]
       Disable a specific mcp tool
     disable prompt [servername]
       Disable all prompts from a mcp server
     disable prompt [promptname]
       Disable a specific prompt
     disable server [servername]
       Disable all tools and prompts from a mcp server
*/

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable MCP entities like tools and prompts globally",
	Long: "Disable one or more tools or prompts globally.\n" +
		"If an entity is disabled in mcpjungle, it CANNOT be consumed by mcp clients via the gateway.\n\n" +
		"NOTE: For backward-compatibility, you can still run 'disable [name]' to disable a tool or all tools from a mcp server.\n" +
		"But the recommended way to achieve this now is 'disable tool [name]'.",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "2",
	},
	RunE: runDisable,
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

var disableServerCmd = &cobra.Command{
	Use:   "server [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Disable all tools and prompts from a MCP server globally",
	Long: "Specify the name of a MCP server to disable all its tools and prompts in the mcp proxy.\n" +
		"If a server is disabled, its tools and prompts CANNOT be viewed or used by mcp clients.",
	RunE: runDisableServer,
}

func init() {
	disableCmd.AddCommand(disableToolsCmd)
	disableCmd.AddCommand(disablePromptsCmd)
	disableCmd.AddCommand(disableServerCmd)
	rootCmd.AddCommand(disableCmd)
}

// runDisable checks if the command is called as `mcpjungle disable [name]`
// and redirects to `mcpjungle disable tool [name]`.
// This is to maintain backward compatibility with older versions of the CLI that only supported disabling tools & servers.
func runDisable(cmd *cobra.Command, args []string) error {
	if len(args) == 1 && cmd.CalledAs() == "disable" {
		cmd.Println(
			"Warning: 'disable [name]' is deprecated. Please use 'disable tool [name]' or 'disable server [name]' instead.",
		)
		cmd.Println()

		// only disable tools, because this was the behaviour before prompts were introduced
		// to disable everything, users should now use `disable server [name]`
		return runDisableTools(cmd, args)
	}
	// Otherwise, just show help message
	return cmd.Help()
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

func runDisableServer(cmd *cobra.Command, args []string) error {
	name := args[0]
	resp, err := apiClient.DisableServer(name)
	if err != nil {
		return fmt.Errorf("failed to disable server %s: %w", name, err)
	}

	cmd.Printf("MCP server '%s' disabled successfully!\n", resp.Name)

	if len(resp.ToolsAffected) > 0 {
		cmd.Println()
		cmd.Println("Following MCP tools have been disabled:")
		for _, tool := range resp.ToolsAffected {
			cmd.Printf("    - %s\n", tool)
		}
	}

	if len(resp.PromptsAffected) > 0 {
		cmd.Println()
		cmd.Println("Following MCP prompts have been disabled:")
		for _, prompt := range resp.PromptsAffected {
			cmd.Printf("    - %s\n", prompt)
		}
	}

	cmd.Println()
	return nil
}
