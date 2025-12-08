package types

// McpClient represents an MCP client that is authorized to access the MCPJungle MCP Proxy server.
type McpClient struct {
	// Name is the name of the client that uniquely identifies it within mcpungle.
	Name        string `json:"name"`
	Description string `json:"description"`

	// AllowList is a list of MCP Servers that this client is allowed to access from MCPJungle.
	AllowList []string `json:"allow_list"`
}

// AllowAllMcpServers is a wildcard operator used to indicate that a mcp client has access to all mcp servers
// in mcpjungle.
const AllowAllMcpServers = "*"
