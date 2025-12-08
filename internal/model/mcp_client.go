package model

import (
	"encoding/json"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// McpClient represents MCP clients and their access to the MCP Servers provided MCPJungle MCP server
type McpClient struct {
	gorm.Model

	Name        string `json:"name" gorm:"uniqueIndex;not null"`
	Description string `json:"description"`

	AccessToken string `json:"access_token" gorm:"unique; not null"`

	// AllowList contains a list of MCP Server names that this client is allowed to view and call
	// storing the list of server names as a JSON array is a convenient way for now.
	// In the future, this will be removed in favor of a separate table for ACLs.
	AllowList datatypes.JSON `json:"allow_list" gorm:"type:jsonb; not null"`
}

// CheckHasServerAccess returns true if this client has access to the specified MCP server.
// If not, it returns false.
func (c *McpClient) CheckHasServerAccess(serverName string) bool {
	if c.AllowList == nil {
		return false
	}
	var allowedServers []string
	if err := json.Unmarshal(c.AllowList, &allowedServers); err != nil {
		return false
	}
	for _, allowed := range allowedServers {
		// If the client's allow list contains wildcard, then it is allowed to access all mcp servers
		if allowed == types.AllowAllMcpServers || allowed == serverName {
			return true
		}
	}
	return false
}
