package model

import "gorm.io/gorm"

// EntityType is the type of entity, e.g., mcp_server, user, mcp_client, group, etc.
type EntityType string

const (
	EntityTypeMcpServer EntityType = "mcp_server"
	EntityTypeMcpClient EntityType = "mcp_client"
	EntityTypeUser      EntityType = "user"
	EntityTypeGroup     EntityType = "group"
)

// ManagedConfigFile tracks configuration files that represent entities in mcpjungle.
// Only the files inside the auto-synced config directory are tracked.
type ManagedConfigFile struct {
	gorm.Model

	EntityType EntityType `json:"entity_type" gorm:"type:varchar(32);not null;index:idx_managed_config_entity,unique"`
	EntityName string     `json:"entity_name" gorm:"type:varchar(255);not null;index:idx_managed_config_entity,unique"`
	FilePath   string     `json:"file_path" gorm:"type:text;not null;index:idx_managed_config_file,unique"`
	FileHash   string     `json:"file_hash" gorm:"type:varchar(128);not null"`
}
