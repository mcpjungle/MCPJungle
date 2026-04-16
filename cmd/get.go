package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get entities like Prompts and Tool Groups",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "1",
	},
}

var getPromptArgs map[string]string

var getGroupCmd = &cobra.Command{
	Use:   "group [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get information about a specific Tool Group",
	Long: "Get information about a specific Tool Group by name.\n" +
		"This returns the configuration of the Tool Group including which tools are included.\n",
	RunE: runGetGroup,
}

var getPromptCmd = &cobra.Command{
	Use:   "prompt [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get a prompt template",
	Long: "Retrieve a prompt template from an MCP server with optional arguments.\n" +
		"The prompt will be rendered with the provided arguments and returned as structured messages.",
	Example: `  # Get a basic prompt
  mcpjungle get prompt github__code-review

  # Get a prompt with arguments
  mcpjungle get prompt github__code-review --arg code="def hello(): print('world')" --arg language="python"`,
	RunE: runGetPrompt,
}

func init() {
	getPromptCmd.Flags().StringToStringVar(
		&getPromptArgs,
		"arg",
		nil,
		"Arguments to pass to the prompt in the form of 'key=value' (this flag can be specified multiple times)",
	)

	getCmd.AddCommand(getGroupCmd)
	getCmd.AddCommand(getPromptCmd)
	rootCmd.AddCommand(getCmd)
}

func runGetGroup(cmd *cobra.Command, args []string) error {
	name := args[0]
	group, err := apiClient.GetToolGroup(name)
	if err != nil {
		return fmt.Errorf("failed to get tool group: %w", err)
	}

	cmd.Println(group.Name)
	if group.Description != "" {
		cmd.Println()
		cmd.Println("Description: " + group.Description)
	}

	cmd.Println()
	cmd.Println("MCP Server streamable http endpoint:")
	cmd.Println(group.StreamableHTTPEndpoint)
	cmd.Println()
	cmd.Println("MCP server SSE endpoints:")
	cmd.Println(group.SSEEndpoint)
	cmd.Println(group.SSEMessageEndpoint)
	cmd.Println()

	if len(group.IncludedTools) == 0 {
		cmd.Println("Included Tools: None")
	} else {
		cmd.Println("Included Tools:")
		for i, t := range group.IncludedTools {
			cmd.Printf("%d. %s\n", i+1, t)
			// TODO: Also show whether the tool is still active, disabled, or deleted at the moment
			// ie, is it practically available as part of this group?
		}
	}
	cmd.Println()

	if len(group.IncludedServers) == 0 {
		cmd.Println("Included Servers: None")
	} else {
		cmd.Println("Included Servers:")
		for i, s := range group.IncludedServers {
			cmd.Printf("%d. %s\n", i+1, s)
		}
	}
	cmd.Println()

	if len(group.ExcludedTools) == 0 {
		cmd.Println("Excluded Tools: None")
	} else {
		cmd.Println("Excluded Tools:")
		for i, t := range group.ExcludedTools {
			cmd.Printf("%d. %s\n", i+1, t)
		}
	}
	cmd.Println()

	cmd.Println(
		"NOTE: If a tool in this group is disabled globally or has been deleted, " +
			"then it will not be available via the group's MCP endpoint.",
	)

	return nil
}

func runGetPrompt(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Convert CLI args to proper format
	arguments := make(map[string]string)
	for k, v := range getPromptArgs {
		arguments[k] = v
	}

	result, err := apiClient.GetPromptWithArgs(name, arguments)
	if err != nil {
		return prettifyGetPromptError(name, err)
	}

	// Pretty print the result
	cmd.Printf("Prompt: %s\n", name)
	if result.Description != "" {
		cmd.Printf("Description: %s\n", result.Description)
	}
	cmd.Println("\nGenerated Messages:")
	cmd.Println("=" + strings.Repeat("=", 50))

	for i, message := range result.Messages {
		cmd.Printf("\nMessage %d (%s):\n", i+1, message.Role)
		cmd.Println("-" + strings.Repeat("-", 30))

		// Format the content nicely
		contentBytes, err := json.MarshalIndent(message.Content, "", "  ")
		if err != nil {
			cmd.Printf("Content: %+v\n", message.Content)
		} else {
			cmd.Printf("Content: %s\n", string(contentBytes))
		}
	}

	return nil
}


// prettifyGetPromptError rewrites low-level errors from GetPromptWithArgs into
// something a CLI user can act on. It:
//  1. Drops the redundant outer "failed to get prompt:" wrapping that
//     duplicates the server's message.
//  2. Adds a usage hint when the upstream MCP server reports invalid
//     arguments (MCP JSON-RPC error -32602), because the raw error payload
//     is a JSON array that looks like noise to someone running the command.
func prettifyGetPromptError(name string, err error) error {
	msg := err.Error()
	// Strip any stacked "failed to get prompt: " prefixes the wrappers add.
	const prefix = "failed to get prompt: "
	for strings.HasPrefix(msg, prefix) {
		msg = msg[len(prefix):]
	}

	// Detect the "Invalid arguments for prompt" shape from the MCP protocol
	// (error -32602) and point the user at --arg so they can retry.
	if strings.Contains(msg, "-32602") || strings.Contains(msg, "Invalid arguments for prompt") {
		return fmt.Errorf(
			"%s\n\nHint: prompt %q rejected the arguments you supplied. "+
				"Pass required arguments with --arg key=value, e.g. "+
				"`mcpjungle get prompt %q --arg key=value`.",
			msg, name, name,
		)
	}
	return fmt.Errorf("%s", msg)
}
