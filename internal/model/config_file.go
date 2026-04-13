package model

import "gorm.io/gorm"

// ConfigFile tracks configuration files that are managed by mcpjungle.
// These config files represent desired state for their corresponding entities.
type ConfigFile struct {
	gorm.Model

	Path       string `json:"path" gorm:"uniqueIndex;not null"`
	EntityType string `json:"entity_type" gorm:"type:varchar(32);index:idx_config_entity,unique;not null"`
	EntityName string `json:"entity_name" gorm:"type:varchar(255);index:idx_config_entity,unique;not null"`
}
