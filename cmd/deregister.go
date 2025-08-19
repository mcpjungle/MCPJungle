package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var deregisterMCPServerCmd = &cobra.Command{
	Use:   "deregister",
	Short: "Deregister an MCP Server",
	Long:  "Remove an MCP server from the registry. This also deregisters all tools provided by the server.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeregisterMCPServer,
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "6",
	},
}

func init() {
	rootCmd.AddCommand(deregisterMCPServerCmd)
}

func runDeregisterMCPServer(cmd *cobra.Command, args []string) error {
	server := args[0]
	if err := apiClient.DeregisterServer(server); err != nil {
		return fmt.Errorf("failed to deregister MCP server %s: %w", server, err)
	}
	fmt.Printf("Successfully deregistered MCP server %s\n", server)
	fmt.Println("The tools provided by this server have also been deregistered.")
	// TODO: Output the list of tools that were deregistered.
	return nil
}
