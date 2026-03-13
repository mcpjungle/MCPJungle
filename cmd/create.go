package cmd

import (
	"fmt"
	"strings"

	"github.com/mcpjungle/mcpjungle/pkg/accesstoken"
	"github.com/mcpjungle/mcpjungle/pkg/configfiles"
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
	Use: "mcp-client [name] | --conf <file>",
	Args: func(cmd *cobra.Command, args []string) error {
		// if a config file is provided, no positional args are expected
		if createMcpClientCmdConfigFilePath != "" {
			return cobra.ExactArgs(0)(cmd, args)
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	Short: "Create an authenticated MCP client (Enterprise mode)",
	Long: "Create an MCP client that can make authenticated requests to the MCPJungle MCP Proxy.\n" +
		"This returns an access token which should be sent by your client in the " +
		"`Authorization: Bearer {token}` http header.\n" +
		"You can also set a custom access token by using the --access-token flag.\n" +
		"Use the --allow option to control which MCP servers the client can access:\n" +
		"    --allow \"server1, server2, server3\" | --allow \"*\"\n" +
		"It is mandatory to either specify the name or a config file.\n" +
		"This command is only available in Enterprise mode.",
	RunE: runCreateMcpClient,
}

var createUserCmd = &cobra.Command{
	Use: "user [username] | --conf <file>",
	Args: func(cmd *cobra.Command, args []string) error {
		// if a config file is provided, no positional args are expected
		if createUserCmdConfigFilePath != "" {
			return cobra.ExactArgs(0)(cmd, args)
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	Short: "Create a new user (Enterprise mode)",
	Long: "Create a new standard user in MCPJungle.\n" +
		"A user can make authenticated requests to the MCPJungle API server and perform limited actions like:\n" +
		"- List and view MCP servers & tools\n" +
		"- Check tool usage and invoke them\n\n" +
		"This operation generates a unique access token for the user to use when making requests.\n" +
		"It is mandatory to either specify the username or a config file.\n" +
		"This command is only available in Enterprise mode.",
	RunE: runCreateUser,
}

var createToolGroupCmd = &cobra.Command{
	Use:   "group --conf <file>",
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
	createMcpClientCmdAccessToken    string
	createMcpClientCmdConfigFilePath string

	createUserCmdAccessToken    string
	createUserCmdConfigFilePath string

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
	createMcpClientCmd.Flags().StringVar(
		&createMcpClientCmdAccessToken,
		"access-token",
		"",
		"Custom access token for the MCP client. If not provided, a random token will be generated.",
	)
	createMcpClientCmd.Flags().StringVarP(
		&createMcpClientCmdConfigFilePath,
		"conf",
		"c",
		"",
		"Path to a JSON configuration file for the MCP client.\n"+
			"If provided, the client will be created using the configuration in the file.\n"+
			"All other flags will be ignored.",
	)

	createUserCmd.Flags().StringVar(
		&createUserCmdAccessToken,
		"access-token",
		"",
		"Custom access token for the user. If not provided, a random token will be generated.",
	)
	createUserCmd.Flags().StringVarP(
		&createUserCmdConfigFilePath,
		"conf",
		"c",
		"",
		"Path to a JSON configuration file for the user.\n"+
			"If provided, the user will be created using the configuration in the file.\n"+
			"All other flags will be ignored.",
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
	client := &types.McpClient{}

	if createMcpClientCmdConfigFilePath == "" {

		// no config file provided, use command line args
		allowList := parseAllowList(createMcpClientCmdAllowedServers, cmd)
		client = &types.McpClient{
			Name:        args[0],
			Description: createMcpClientCmdDescription,
			AllowList:   allowList,
		}
		if createMcpClientCmdAccessToken != "" {
			client.AccessToken = createMcpClientCmdAccessToken
			client.IsCustomAccessToken = true
		}

	} else {

		// config file provided, ignore command line args and read from file
		config, err := readMcpClientConfig(createMcpClientCmdConfigFilePath)
		if err != nil {
			return err
		}
		if config.Name == "" {
			return fmt.Errorf("config file must define a client name")
		}
		client = &types.McpClient{
			Name:        config.Name,
			Description: config.Description,
			AllowList:   config.AllowMcpServers,
		}
		accessToken, err := accesstoken.Resolve(accesstoken.Input{
			Inline: config.AccessToken,
			Ref:    config.AccessTokenRef,
		})
		if err != nil {
			return err
		}
		if accessToken == "" {
			return fmt.Errorf("config file must supply a custom access token")
		}
		client.AccessToken = accessToken
		client.IsCustomAccessToken = true

		// if a wildcard is used in the allow list, warn the user
		for _, entry := range config.AllowMcpServers {
			if entry == types.AllowAllMcpServers {
				warnAllowAll(cmd)
				break
			}
		}

	}

	token, err := apiClient.CreateMcpClient(client)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}
	if !client.IsCustomAccessToken && token == "" {
		// user didn't supply a custom token and server didn't generate a valid one
		return fmt.Errorf("server returned an empty token, this was unexpected")
	}

	cmd.Printf("MCP client '%s' created successfully!\n", client.Name)

	if len(client.AllowList) > 0 {
		cmd.Println("Servers accessible: " + strings.Join(client.AllowList, ","))
	} else {
		cmd.Println("This client does not have access to any MCP servers.")
	}

	if !client.IsCustomAccessToken {
		// server generated the access token, display it to the user
		cmd.Printf("\nAccess token: %s\n", token)
	}
	cmd.Println("Your client should send the access token in the `Authorization: Bearer {token}` HTTP header.")

	return nil
}

func runCreateUser(cmd *cobra.Command, args []string) error {
	user := &types.CreateOrUpdateUserRequest{}

	if createUserCmdConfigFilePath == "" {
		// no config file provided, use command line args
		user = &types.CreateOrUpdateUserRequest{
			Username:    args[0],
			AccessToken: createUserCmdAccessToken,
		}
	} else {
		// config file provided, ignore command line args and read from file
		config, err := readUserConfig(createUserCmdConfigFilePath)
		if err != nil {
			return err
		}
		if config.Username == "" {
			return fmt.Errorf("config file must define a username")
		}
		accessToken, err := accesstoken.Resolve(accesstoken.Input{
			Inline: config.AccessToken,
			Ref:    config.AccessTokenRef,
		})
		if err != nil {
			return err
		}
		if accessToken == "" {
			return fmt.Errorf("config file must supply a custom access token")
		}
		user = &types.CreateOrUpdateUserRequest{
			Username:    config.Username,
			AccessToken: accessToken,
		}
	}
	resp, err := apiClient.CreateUser(user)
	if err != nil {
		return err
	}
	if resp.AccessToken == "" {
		return fmt.Errorf("server returned an empty access token, this was unexpected")
	}

	cmd.Printf("User '%s' created successfully\n", user.Username)
	cmd.Println("The user should now run the following command to log into mcpjungle:")
	cmd.Println()
	cmd.Printf("    mcpjungle login %s\n", resp.AccessToken)
	cmd.Println()

	return nil
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

// readToolGroupConfig reads the tool group configuration from a JSON file.
func readToolGroupConfig(filePath string) (*types.ToolGroup, error) {
	input, err := configfiles.ReadJSONFile[types.ToolGroup](filePath)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

// readMcpClientConfig reads the MCP client configuration from a JSON file.
func readMcpClientConfig(filePath string) (*types.McpClientConfig, error) {
	input, err := configfiles.ReadJSONFile[types.McpClientConfig](filePath)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

// readUserConfig reads the user configuration from a JSON file.
func readUserConfig(filePath string) (*types.UserConfig, error) {
	input, err := configfiles.ReadJSONFile[types.UserConfig](filePath)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

// parseAllowList parses a comma-separated string of allowed MCP servers into a slice.
func parseAllowList(input string, cmd *cobra.Command) []string {
	allowList := make([]string, 0)
	for _, s := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			allowList = append(allowList, trimmed)
		}
		if trimmed == types.AllowAllMcpServers {
			warnAllowAll(cmd)
		}
	}

	return allowList
}

// warnAllowAll displays a warning message about using a wildcard in the allow list.
func warnAllowAll(cmd *cobra.Command) {
	cmd.Println("NOTE: This client will have access to all MCP Servers because a wildcard is used.")
	cmd.Println("This practice is highly discouraged!")
	cmd.Println()
}
