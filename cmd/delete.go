package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "4",
	},
}

var deleteMcpClientCmd = &cobra.Command{
	Use:   "mcp-client [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Delete an MCP client (Production mode)",
	Long: "Delete an MCP client from the registry. This instantly revokes all access of this client.\n" +
		"This command is only available in Production mode.",
	RunE: runDeleteMcpClient,
}

var deleteUserCmd = &cobra.Command{
	Use:   "user [username]",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a user (Production mode)",
	Long:  "Delete a user from mcpjungle.\nThis instantly revokes all access of this user.",
	RunE:  runDeleteUser,
}

func init() {
	deleteCmd.AddCommand(deleteMcpClientCmd)
	deleteCmd.AddCommand(deleteUserCmd)

	rootCmd.AddCommand(deleteCmd)
}

func runDeleteMcpClient(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := apiClient.DeleteMcpClient(name); err != nil {
		return fmt.Errorf("failed to delete the client: %w", err)
	}
	fmt.Printf("MCP client '%s' deleted successfully (if it existed)!\n", name)
	return nil
}

func runDeleteUser(cmd *cobra.Command, args []string) error {
	username := args[0]
	if err := apiClient.DeleteUser(username); err != nil {
		return fmt.Errorf("failed to delete the user: %w", err)
	}
	cmd.Printf("User '%s' deleted successfully (if they existed)\n", username)
	return nil
}
