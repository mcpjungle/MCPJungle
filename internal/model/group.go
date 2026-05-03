package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Group struct {
	gorm.Model
	Name        string         `json:"name" gorm:"uniqueIndex;not null"`
	Description string         `json:"description"`
	AllowList   datatypes.JSON `json:"allow_list" gorm:"type:jsonb"`
}
