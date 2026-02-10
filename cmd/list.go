package cmd

import (
	"fmt"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources like MCP servers, tools, etc",
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "3",
	},
}

var (
	listToolsCmdServerName string
	listToolsCmdGroupName  string
)

var listPromptsCmdServerName string

var listToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available tools",
	Long: "List tools available either from a specific MCP server, tool group, or across " +
		"all MCP servers registered in mcpjungle.\n\n" +
		"NOTE: When using --group flag, this command only displays tools that currently exist " +
		"in mcpjungle and are part of the group.\n" +
		"So if, for example, the group includes a tool that has been deleted, this command won't display it.\n" +
		"To get the full list of tools included in a group, use the `get group` command instead.",
	RunE: runListTools,
}

var listPromptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "List available prompts",
	Long:  "List prompt templates available either from a specific MCP server or across all MCP servers in mcpjungle.",
	RunE:  runListPrompts,
}

var listServersCmd = &cobra.Command{
	Use:   "servers",
	Short: "List registered MCP servers",
	RunE:  runListServers,
}

var listMcpClientsCmd = &cobra.Command{
	Use:   "mcp-clients",
	Short: "List MCP clients (Enterprise mode)",
	Long: "List MCP clients that are authorized to access the MCP Proxy server.\n" +
		"This command is only available in Enterprise mode.",
	RunE: runListMcpClients,
}

var listUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users (Enterprise mode)",
	Long:  "List users that are authorized to access MCPJungle.",
	RunE:  runListUsers,
}

var listGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List tool groups",
	RunE:  runListGroups,
}

func init() {
	listToolsCmd.Flags().StringVar(
		&listToolsCmdServerName,
		"server",
		"",
		"Filter tools by server name",
	)
	listToolsCmd.Flags().StringVar(
		&listToolsCmdGroupName,
		"group",
		"",
		"Filter tools by tool group name",
	)

	listPromptsCmd.Flags().StringVar(
		&listPromptsCmdServerName,
		"server",
		"",
		"Filter prompts by server name",
	)

	listCmd.AddCommand(listToolsCmd)
	listCmd.AddCommand(listPromptsCmd)
	listCmd.AddCommand(listServersCmd)
	listCmd.AddCommand(listMcpClientsCmd)
	listCmd.AddCommand(listUsersCmd)
	listCmd.AddCommand(listGroupsCmd)

	rootCmd.AddCommand(listCmd)
}

func runListTools(cmd *cobra.Command, args []string) error {
	// If both server and group flags are provided, reject the request.
	if listToolsCmdServerName != "" && listToolsCmdGroupName != "" {
		return fmt.Errorf("using both --server and --group flags together is currently not supported")
	}

	var tools []*types.Tool
	var err error
	var contextInfo string

	if listToolsCmdGroupName != "" {
		// Get tools from specific group
		group, err := apiClient.GetToolGroup(listToolsCmdGroupName)
		if err != nil {
			return fmt.Errorf("failed to get tool group '%s': %w", listToolsCmdGroupName, err)
		}

		effectiveTools, err := apiClient.GetToolGroupEffectiveTools(listToolsCmdGroupName)
		if err != nil {
			return fmt.Errorf("failed to resolve effective tools for group '%s': %w", listToolsCmdGroupName, err)
		}

		// Get all tools first, then filter by group's effective tools.
		// This is necessary because a group might contain tools that do not currently exist in mcpjungle.
		// for eg- the tool was deleted after group creation or the group includes a non-existent tool.
		// ListTools only returns tools that actually exist in mcpjungle, so we must cross-check.
		allTools, err := apiClient.ListTools("")
		if err != nil {
			return fmt.Errorf("failed to list all tools: %w", err)
		}

		// Create a map for efficient lookup
		effectiveToolsMap := make(map[string]bool)
		for _, toolName := range effectiveTools {
			effectiveToolsMap[toolName] = true
		}

		// Filter tools that are in the group
		for _, tool := range allTools {
			if effectiveToolsMap[tool.Name] {
				tools = append(tools, tool)
			}
		}

		contextInfo = fmt.Sprintf("Tools in group '%s'", listToolsCmdGroupName)
		if group.Description != "" {
			contextInfo += fmt.Sprintf(" (%s)", group.Description)
		}
	} else {
		// no group specified, list tools from specific server (if flag is set) or all servers
		tools, err = apiClient.ListTools(listToolsCmdServerName)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}

		if listToolsCmdServerName != "" {
			contextInfo = fmt.Sprintf("Tools from server '%s'", listToolsCmdServerName)
		}
	}

	if len(tools) == 0 {
		if listToolsCmdGroupName != "" {
			cmd.Printf("There are no valid tools in group '%s'\n", listToolsCmdGroupName)
		} else if listToolsCmdServerName != "" {
			cmd.Printf("There are no tools from mcp server '%s'\n", listToolsCmdServerName)
		} else {
			cmd.Println("There are currently no tools in the registry")
		}
		return nil
	}

	// Display context information if filtering is applied
	if contextInfo != "" {
		cmd.Printf("%s:\n\n", contextInfo)
	}

	for i, t := range tools {
		ed := "ENABLED"
		if !t.Enabled {
			ed = "DISABLED"
		}
		cmd.Printf("%d. %s  [%s]\n", i+1, t.Name, ed)
		cmd.Println(t.Description)
		cmd.Println()
	}

	cmd.Println("Run 'usage <tool name>' to see a tool's usage or 'invoke <tool name>' to call one")

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

		if s.Description != "" {
			fmt.Println(s.Description)
		}

		fmt.Println("Transport: " + s.Transport)

		t, _ := types.ValidateTransport(s.Transport)
		if t == types.TransportStreamableHTTP || t == types.TransportSSE {
			fmt.Println("URL: " + s.URL)
		} else {
			if len(s.Args) > 0 {
				fmt.Println("Command: " + s.Command + " " + strings.Join(s.Args, " "))
			} else {
				fmt.Println("Command: " + s.Command)
			}

			if len(s.Env) > 0 {
				fmt.Printf("Environment variables: %s\n", s.Env)
			}
		}

		if i < len(servers)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runListMcpClients(cmd *cobra.Command, args []string) error {
	clients, err := apiClient.ListMcpClients()
	if err != nil {
		return fmt.Errorf("failed to list MCP clients: %w", err)
	}

	if len(clients) == 0 {
		fmt.Println("There are no MCP clients in the registry")
		return nil
	}
	for i, c := range clients {
		fmt.Printf("%d. %s\n", i+1, c.Name)

		if c.Description != "" {
			fmt.Println("Description: ", c.Description)
		}

		if len(c.AllowList) > 0 {
			fmt.Println("Allowed servers: " + strings.Join(c.AllowList, ","))
		} else {
			fmt.Println("This client does not have access to any MCP servers.")
		}

		if i < len(clients)-1 {
			fmt.Println()
		}
	}

	return nil
}

func runListUsers(cmd *cobra.Command, args []string) error {
	users, err := apiClient.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		cmd.Println("There are no users in the registry")
		return nil
	}
	for i, u := range users {
		if u.Role == string(types.UserRoleAdmin) {
			cmd.Printf("%d. %s  [ADMIN]\n", i+1, u.Username)
		} else {
			cmd.Printf("%d. %s\n", i+1, u.Username)
		}

		if i < len(users)-1 {
			cmd.Println()
		}
	}

	return nil
}

func runListGroups(cmd *cobra.Command, args []string) error {
	groups, err := apiClient.ListToolGroups()
	if err != nil {
		return fmt.Errorf("failed to list tool groups: %w", err)
	}

	if len(groups) == 0 {
		cmd.Println("There are no tool groups in the registry")
		return nil
	}
	for i, g := range groups {
		cmd.Printf("%d. %s\n", i+1, g.Name)
		if g.Description != "" {
			cmd.Println(g.Description)
		}

		if i < len(groups)-1 {
			cmd.Println()
		}
	}

	return nil
}

func runListPrompts(cmd *cobra.Command, args []string) error {
	prompts, err := apiClient.ListPrompts(listPromptsCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	if len(prompts) == 0 {
		cmd.Println("No prompts found")
		return nil
	}
	for i, p := range prompts {
		ed := "ENABLED"
		if !p.Enabled {
			ed = "DISABLED"
		}
		cmd.Printf("%d. %s  [%s]\n", i+1, p.Name, ed)
		if p.Description != "" {
			cmd.Println(p.Description)
		}
		cmd.Println()
	}

	cmd.Println("Run 'get prompt <prompt name>' to retrieve a prompt template")

	return nil
}
