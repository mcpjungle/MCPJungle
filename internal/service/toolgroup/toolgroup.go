package toolgroup

import (
	"errors"
	"fmt"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// ToolGroupService provides methods to manage tool groups.
type ToolGroupService struct {
	db *gorm.DB
}

func NewToolGroupService(db *gorm.DB) *ToolGroupService {
	return &ToolGroupService{db: db}
}

// CreateToolGroup creates a new tool group in the database.
func (s *ToolGroupService) CreateToolGroup(group *model.ToolGroup) error {
	return s.db.Create(group).Error
}

// GetToolGroup retrieves a tool group by name from the database.
func (s *ToolGroupService) GetToolGroup(name string) (*model.ToolGroup, error) {
	var group model.ToolGroup
	if err := s.db.Where("name = ?", name).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// ListToolGroups retrieves all tool groups from the database.
func (s *ToolGroupService) ListToolGroups() ([]model.ToolGroup, error) {
	var groups []model.ToolGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *ToolGroupService) DeleteToolGroup(name string) error {
	err := s.db.Unscoped().Where("name = ?", name).Delete(&model.ToolGroup{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete toolgroup: %w", err)
	}
	return nil
}
