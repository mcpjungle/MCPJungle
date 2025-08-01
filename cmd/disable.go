package cmd

import "github.com/spf13/cobra"

var disableCmd = &cobra.Command{
	Use:   "disable [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Disable one or more MCP tools globally",
	Long: "Specify the name of a tool or MCP server to disable it in the mcp proxy.\n" +
		"If a server is specified, all tools provided by that server will be disabled.\n" +
		"If a tool is disabled, it cannot be viewed or called by mcp clients.",
	RunE: runDisableTools,
}

func init() {
	rootCmd.AddCommand(disableCmd)
}

func runDisableTools(cmd *cobra.Command, args []string) error {
	name := args[0]
	toolsDisabled, err := apiClient.DisableTools(name)
	if err != nil {
		return err
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
