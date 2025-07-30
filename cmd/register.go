package cmd

import (
	"fmt"
	"github.com/mcpjungle/mcpjungle/client"
	"github.com/spf13/cobra"
)

var (
	registerCmdServerName  string
	registerCmdServerURL   string
	registerCmdServerDesc  string
	registerCmdBearerToken string
)

var registerMCPServerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an MCP Server",
	Long: "Register a MCP Server with the registry.\nA server name is unique across the registry and " +
		"must not contain any whitespaces, special characters or multiple consecutive underscores '__'.",
	RunE: runRegisterMCPServer,
}

func init() {
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdServerName,
		"name",
		"",
		"MCP server name",
	)
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdServerURL,
		"url",
		"",
		"URL of the MCP server (eg- http://localhost:8000/mcp)",
	)
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdServerDesc,
		"description",
		"",
		"Server description",
	)
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdBearerToken,
		"bearer-token",
		"",
		"If provided, MCPJungle will use this token to authenticate with the MCP server for all requests."+
			" This is useful if the MCP server requires static tokens (eg- your API token) for authentication.",
	)

	// TODO: name should not be mandatory.
	//  If not supplied, name should be read from MCP server metadata by the registry.
	_ = registerMCPServerCmd.MarkFlagRequired("name")
	_ = registerMCPServerCmd.MarkFlagRequired("url")

	rootCmd.AddCommand(registerMCPServerCmd)
}

func runRegisterMCPServer(cmd *cobra.Command, args []string) error {
	input := &client.RegisterServerInput{
		Name:        registerCmdServerName,
		URL:         registerCmdServerURL,
		Description: registerCmdServerDesc,
		BearerToken: registerCmdBearerToken,
	}
	s, err := apiClient.RegisterServer(input)
	if err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}
	fmt.Printf("Server %s registered successfully!\n", s.Name)

	tools, err := apiClient.ListTools(s.Name)
	if err != nil {
		// if we fail to fetch tool list, fail silently because this is not a must-have output
		return nil
	}
	fmt.Println()
	fmt.Println("The following tools are now available from this server:")
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	return nil
}
