package model

import "gorm.io/gorm"

// McpServerTaskCapability stores the tasks capability advertised by an upstream MCP server
// during initialization. Each McpServer has at most one record in this table.
type McpServerTaskCapability struct {
	gorm.Model

	// List indicates the upstream server supports tasks/list.
	List bool `json:"list" gorm:"default:false"`
	// Cancel indicates the upstream server supports tasks/cancel.
	Cancel bool `json:"cancel" gorm:"default:false"`
	// ToolCallTasks indicates the upstream server supports tools/call augmented with task metadata.
	ToolCallTasks bool `json:"tool_call_tasks" gorm:"default:false"`

	// ServerID is the ID of the MCP server that provides this task capability.
	ServerID uint      `json:"-" gorm:"not null"`
	Server   McpServer `json:"-" gorm:"foreignKey:ServerID;references:ID"`
}
