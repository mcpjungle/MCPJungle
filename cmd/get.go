package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get information about a specific resource",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "7",
	},
}

var getGroupCmd = &cobra.Command{
	Use:   "group [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get information about a specific Tool Group",
	Long: "Get information about a specific Tool Group by name.\n" +
		"This returns the configuration of the Tool Group including which tools are included.\n",
	RunE: runGetGroup,
}

func init() {
	getCmd.AddCommand(getGroupCmd)

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
