package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

/*
   Usage:
     enable [servername]
       (Deprecated, for backward-compatibility) Enable all tools from a mcp server
     enable [toolname]
       (Deprecated, for backward compatibility) Enable a specific mcp tool
     enable tool [servername]
       Enable all tools from a mcp server
     enable tool [toolname]
       Enable a specific mcp tool
     enable prompt [servername]
       Enable all prompts from a mcp server
     enable prompt [promptname]
       Enable a specific prompt
     enable server [servername]
       Enable all tools and prompts from a mcp server
*/

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable MCP entities like tools & prompts globally",
	Long: "Enable one or more tools or prompts globally.\n" +
		"If an entity is enabled in mcpjungle, it can be consumed by mcp clients via the gateway.\n\n" +
		"NOTE: For backward-compatibility, you can still run 'enable [name]' to enable a tool or all tools from a mcp server.\n" +
		"But the recommended way to achieve this now is 'enable tool [name]'.",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "3",
	},
	RunE: runEnable,
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

var enableServerCmd = &cobra.Command{
	Use:   "server [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Enable all tools and prompts from a MCP server globally",
	Long: "Specify the name of a MCP server to enable all its tools and prompts in the mcp proxy.\n" +
		"If a server is enabled, all its tools and prompts can be viewed and used by mcp clients.",
	RunE: runEnableServer,
}

func init() {
	enableCmd.AddCommand(enableToolsCmd)
	enableCmd.AddCommand(enablePromptsCmd)
	enableCmd.AddCommand(enableServerCmd)

	rootCmd.AddCommand(enableCmd)
}

// runEnable checks if the command is called as `mcpjungle enable [name]`
// and redirects to `mcpjungle enable tool [name]`.
// This is to maintain backward compatibility with older versions of the CLI that only supported enabling tools & servers.
func runEnable(cmd *cobra.Command, args []string) error {
	if len(args) == 1 && cmd.CalledAs() == "enable" {
		cmd.Println(
			"Warning: 'enable [name]' is deprecated. Please use 'enable tool [name]' or 'enable server [name]' instead.",
		)
		cmd.Println()
		return runEnableTools(cmd, args)
	}
	// Otherwise, just show help message
	return cmd.Help()
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

func runEnableServer(cmd *cobra.Command, args []string) error {
	name := args[0]
	resp, err := apiClient.EnableServer(name)
	if err != nil {
		return fmt.Errorf("failed to enable server %s: %w", name, err)
	}

	cmd.Printf("MCP server '%s' enabled successfully!\n", resp.Name)

	if len(resp.ToolsAffected) > 0 {
		cmd.Println()
		cmd.Println("Following MCP tools have been enabled:")
		for _, tool := range resp.ToolsAffected {
			cmd.Printf("    - %s\n", tool)
		}
	}

	if len(resp.PromptsAffected) > 0 {
		cmd.Println()
		cmd.Println("Following MCP prompts have been enabled:")
		for _, prompt := range resp.PromptsAffected {
			cmd.Printf("    - %s\n", prompt)
		}
	}

	cmd.Println()
	return nil
}
