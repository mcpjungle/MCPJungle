package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
	"os"
)

var (
	registerCmdServerName  string
	registerCmdServerURL   string
	registerCmdServerDesc  string
	registerCmdBearerToken string

	registerCmdServerConfigFilePath string
)

var registerMCPServerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an MCP Server",
	Long: "Register a MCP Server with the registry.\nA server name is unique across the registry and " +
		"must not contain any whitespaces, special characters or multiple consecutive underscores '__'.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip flag validation if config file is provided
		if registerCmdServerConfigFilePath != "" {
			return nil
		}
		// Otherwise, validate required flags
		if registerCmdServerName == "" {
			return fmt.Errorf("required flag \"name\" not set")
		}
		if registerCmdServerURL == "" {
			return fmt.Errorf("required flag \"url\" not set")
		}
		return nil
	},
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
	registerMCPServerCmd.Flags().StringVarP(
		&registerCmdServerConfigFilePath,
		"conf",
		"c",
		"",
		"Path to a JSON configuration file for the MCP server.\n"+
			"If provided, the mcp server will be registered using the configuration in the file.\n"+
			"All other flags will be ignored.",
	)

	rootCmd.AddCommand(registerMCPServerCmd)
}

func runRegisterMCPServer(cmd *cobra.Command, args []string) error {
	var input types.RegisterServerInput

	if registerCmdServerConfigFilePath == "" {
		// If no config file is provided, use the flags to create the input for server registration
		input = types.RegisterServerInput{
			Name:        registerCmdServerName,
			URL:         registerCmdServerURL,
			Description: registerCmdServerDesc,
			BearerToken: registerCmdBearerToken,
		}
	} else {
		// If a config file is provided, read the configuration from the file
		data, err := os.ReadFile(registerCmdServerConfigFilePath)
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", registerCmdServerConfigFilePath, err)
		}

		// Parse JSON config
		if err := json.Unmarshal(data, &input); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}

		// Validate required fields
		if input.Name == "" {
			return fmt.Errorf("missing required field 'name' in config file")
		}
	}

	s, err := apiClient.RegisterServer(&input)
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
	for i, tool := range tools {
		fmt.Printf("%d. %s: %s\n\n", i, tool.Name, tool.Description)
	}

	return nil
}
