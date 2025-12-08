package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create entities in mcpjungle",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "4",
	},
}

var createMcpClientCmd = &cobra.Command{
	Use:   "mcp-client [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Create an authenticated MCP client (Enterprise mode)",
	Long: "Create an MCP client that can make authenticated requests to the MCPJungle MCP Proxy.\n" +
		"This returns an access token which should be sent by your client in the " +
		"`Authorization: Bearer {token}` http header.\n" +
		"Use the --allow option to control which MCP servers the client can access:\n" +
		"    --allow \"server1, server2, server3\" | --allow \"*\"\n" +
		"This command is only available in Enterprise mode.",
	RunE: runCreateMcpClient,
}

var createUserCmd = &cobra.Command{
	Use:   "user [username]",
	Args:  cobra.ExactArgs(1),
	Short: "Create a new user (Enterprise mode)",
	Long: "Create a new standard user in MCPJungle.\n" +
		"A user can make authenticated requests to the MCPJungle API server and perform limited actions like:\n" +
		"- List and view MCP servers & tools\n" +
		"- Check tool usage and invoke them",
	RunE: runCreateUser,
}

var createToolGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Create a Group of MCP Tools",
	Long: "Create a new Group of MCP Tools by supplying a configuration file.\n" +
		"A group lets you expose only a handful of Tools that you choose.\n" +
		"This limits the number of tools your MCP client sees, increasing calling accuracy of the LLM.\n\n" +
		"You can include tools by:\n" +
		"  - Specifying individual tools with 'included_tools'\n" +
		"  - Including all tools from servers with 'included_servers'\n" +
		"  - Excluding specific tools with 'excluded_tools'\n\n" +
		"Once you create a tool group, it is accessible as a streamable http MCP server at the following endpoint:\n" +
		"    /v0/groups/{group_name}/mcp\n",
	RunE: runCreateToolGroup,
}

var (
	createMcpClientCmdAllowedServers string
	createMcpClientCmdDescription    string

	createToolGroupConfigFilePath string
)

func init() {
	createMcpClientCmd.Flags().StringVar(
		&createMcpClientCmdAllowedServers,
		"allow",
		"",
		"Comma-separated list of MCP servers that this client is allowed to access.\n"+
			"By default, the list is empty, meaning the client cannot access any MCP servers.",
	)
	createMcpClientCmd.Flags().StringVar(
		&createMcpClientCmdDescription,
		"description",
		"",
		"Description of the MCP client. This is optional and can be used to provide additional context.",
	)

	createToolGroupCmd.Flags().StringVarP(
		&createToolGroupConfigFilePath,
		"conf",
		"c",
		"",
		"Path to a JSON configuration file for the Group",
	)
	_ = createToolGroupCmd.MarkFlagRequired("conf")

	createCmd.AddCommand(createMcpClientCmd)
	createCmd.AddCommand(createUserCmd)
	createCmd.AddCommand(createToolGroupCmd)

	rootCmd.AddCommand(createCmd)
}

func runCreateMcpClient(cmd *cobra.Command, args []string) error {
	// convert the comma-separated list of allowed servers into a slice
	allowList := make([]string, 0)
	for _, s := range strings.Split(createMcpClientCmdAllowedServers, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			allowList = append(allowList, trimmed)
		}
		if trimmed == types.AllowAllMcpServers {
			cmd.Println("NOTE: This client will have access to all MCP Servers because a wildcard is used.")
			cmd.Println("This practice is highly discouraged!")
			cmd.Println()
		}
	}

	c := &types.McpClient{
		Name:        args[0],
		Description: createMcpClientCmdDescription,
		AllowList:   allowList,
	}

	token, err := apiClient.CreateMcpClient(c)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}
	if token == "" {
		return fmt.Errorf("server returned an empty token, this was unexpected")
	}

	cmd.Printf("MCP client '%s' created successfully!\n", c.Name)

	if len(c.AllowList) > 0 {
		cmd.Println("Servers accessible: " + strings.Join(c.AllowList, ","))
	} else {
		cmd.Println("This client does not have access to any MCP servers.")
	}

	cmd.Printf("\nAccess token: %s\n", token)
	cmd.Println("Your client should send this token in the `Authorization: Bearer {token}` HTTP header.")

	return nil
}

func runCreateUser(cmd *cobra.Command, args []string) error {
	u := &types.CreateUserRequest{
		Username: args[0],
	}
	resp, err := apiClient.CreateUser(u)
	if err != nil {
		return err
	}
	if resp.AccessToken == "" {
		return fmt.Errorf("server returned an empty access token, this was unexpected")
	}

	cmd.Printf("User '%s' created successfully\n", u.Username)
	cmd.Println("The user should now run the following command to log into mcpjungle:")
	cmd.Println()
	cmd.Printf("    mcpjungle login %s\n", resp.AccessToken)
	cmd.Println()

	return nil
}

func readToolGroupConfig(filePath string) (*types.ToolGroup, error) {
	var input types.ToolGroup

	data, err := os.ReadFile(filePath)
	if err != nil {
		return &input, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return &input, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &input, nil
}

func runCreateToolGroup(cmd *cobra.Command, args []string) error {
	group, err := readToolGroupConfig(createToolGroupConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", createToolGroupConfigFilePath, err)
	}

	resp, err := apiClient.CreateToolGroup(group)
	if err != nil {
		return fmt.Errorf("failed to create tool group: %w", err)
	}

	cmd.Printf("Tool Group %s created successfully\n", group.Name)
	cmd.Print("It is now accessible at the following streamable http endpoint:\n\n")
	cmd.Println("    " + resp.StreamableHTTPEndpoint + "\n")

	cmd.Print("Tools using the SSE (server-sent events) transport are accessible at:\n\n")
	cmd.Println("    " + resp.SSEEndpoint)
	cmd.Println("    " + resp.SSEMessageEndpoint + "\n")

	return nil
}
