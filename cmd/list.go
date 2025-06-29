package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
}

var listToolsCmdServerName string

var listToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available tools",
	Long:  "List tools available either from a specific MCP server or across all MCP servers registered in the registry.",
	RunE:  runListTools,
}

var listServersCmd = &cobra.Command{
	Use:   "servers",
	Short: "List registered MCP servers",
	RunE:  runListServers,
}

func init() {
	listToolsCmd.Flags().StringVar(
		&listToolsCmdServerName,
		"server",
		"",
		"Filter tools by server name",
	)

	listCmd.AddCommand(listToolsCmd)
	listCmd.AddCommand(listServersCmd)
	rootCmd.AddCommand(listCmd)
}

func runListTools(cmd *cobra.Command, args []string) error {
	tools, err := apiClient.ListTools(listToolsCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if len(tools) == 0 {
		fmt.Println("There are no tools in the registry")
		return nil
	}
	for i, t := range tools {
		fmt.Printf("%d. %s\n", i+1, t.Name)
		fmt.Println(t.Description)
		fmt.Println()
	}

	fmt.Println("Run 'usage <tool name>' to see a tool's usage or 'invoke <tool name>' to call one")

	return nil
}

func runListServers(cmd *cobra.Command, args []string) error {
	servers, err := apiClient.ListServers()
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("There are no MCP servers in the registry")
		return nil
	}
	for i, s := range servers {
		fmt.Printf("%d. %s\n", i+1, s.Name)
		fmt.Println(s.URL)
		fmt.Println(s.Description)
		if i < len(servers)-1 {
			fmt.Println()
		}
	}

	return nil
}
