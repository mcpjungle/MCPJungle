package cmd

import (
	"fmt"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var (
	installCmdServerName string
	installCmdArgs       []string
	installCmdEnv        map[string]string
	installCmdVersion    string
)

var installMCPServerCmd = &cobra.Command{
	Use:   "install <server-name>",
	Short: "Install an MCP server from the official registry",
	Long: "Install an MCP server from the official Model Context Protocol servers registry.\n" +
		"This command automatically fetches server configuration and registers it with MCPJungle.\n" +
		"\nExamples:\n" +
		"  mcpjungle install time\n" +
		"  mcpjungle install filesystem --args /path/to/allowed/files\n" +
		"  mcpjungle install github --env GITHUB_TOKEN=your_token\n" +
		"  mcpjungle install memory --version 1.2.3",
	Args: cobra.ExactArgs(1),
	RunE: runInstallMCPServer,
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "4",
	},
}

func init() {
	installMCPServerCmd.Flags().StringSliceVar(
		&installCmdArgs,
		"args",
		[]string{},
		"Additional arguments to pass to the MCP server",
	)
	installMCPServerCmd.Flags().StringToStringVar(
		&installCmdEnv,
		"env",
		map[string]string{},
		"Environment variables to set for the MCP server (format: KEY=VALUE)",
	)
	installMCPServerCmd.Flags().StringVar(
		&installCmdVersion,
		"version",
		"",
		"Specific version of the MCP server to install",
	)

	rootCmd.AddCommand(installMCPServerCmd)
}

func runInstallMCPServer(cmd *cobra.Command, args []string) error {
	serverName := args[0]

	// Validate server name
	if serverName == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Parse environment variables from command line
	envVars := make(map[string]string)
	for key, value := range installCmdEnv {
		envVars[key] = value
	}

	// Create install options
	options := &types.InstallOptions{
		ServerName: serverName,
		Args:       installCmdArgs,
		Env:        envVars,
		Version:    installCmdVersion,
	}

	// Install the server
	server, err := apiClient.InstallServer(options)
	if err != nil {
		return fmt.Errorf("failed to install server: %w", err)
	}

	fmt.Printf("Successfully installed MCP server: %s\n", server.Name)
	if server.Description != "" {
		fmt.Printf("Description: %s\n", server.Description)
	}
	fmt.Printf("Transport: %s\n", server.Transport)

	// Show additional information based on transport type
	switch server.Transport {
	case string(types.TransportStdio):
		if len(server.Args) > 0 {
			fmt.Printf("Command: %s %s\n", server.Command, strings.Join(server.Args, " "))
		} else {
			fmt.Printf("Command: %s\n", server.Command)
		}
		if len(server.Env) > 0 {
			fmt.Printf("Environment variables: %v\n", server.Env)
		}
	case string(types.TransportStreamableHTTP), string(types.TransportSSE):
		fmt.Printf("URL: %s\n", server.URL)
	}

	// Try to list tools provided by the server
	tools, err := apiClient.ListTools(server.Name)
	if err != nil {
		// If we fail to fetch tool list, fail silently because this is not a must-have output
		fmt.Println()
		fmt.Println("Server installed successfully!")
		return nil
	}

	fmt.Println()
	fmt.Println("The following tools are now available from this server:")
	for i, tool := range tools {
		fmt.Printf("%d. %s: %s\n", i+1, tool.Name, tool.Description)
	}

	return nil
}
