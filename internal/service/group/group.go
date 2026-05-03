package group

import (
	"encoding/json"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

type GroupService struct {
	db *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{db: db}
}

type UpdateGroupInput struct {
	Description     string
	AllowList       []string
	UpdateAllowList bool
}

func (s *GroupService) CreateGroup(input model.Group) (*model.Group, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	if len(input.AllowList) == 0 {
		encoded, _ := json.Marshal([]string{"*"})
		input.AllowList = encoded
	}
	if err := s.db.Create(&input).Error; err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *GroupService) UpdateGroup(name string, input UpdateGroupInput) (*model.Group, error) {
	var g model.Group
	if err := s.db.Where("name = ?", name).First(&g).Error; err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	updates := map[string]any{"description": input.Description}
	if input.UpdateAllowList {
		encoded, err := json.Marshal(input.AllowList)
		if err != nil {
			return nil, fmt.Errorf("invalid allow_list: %w", err)
		}
		updates["allow_list"] = encoded
	}
	if err := s.db.Model(&g).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *GroupService) DeleteGroup(name string) error {
	var g model.Group
	if err := s.db.Where("name = ?", name).First(&g).Error; err != nil {
		return fmt.Errorf("group not found: %w", err)
	}
	var count int64
	s.db.Model(&model.User{}).Where("group_id = ?", g.ID).Count(&count)
	if count > 0 {
		return fmt.Errorf("cannot delete group %q: it has %d member(s); remove them first", name, count)
	}
	return s.db.Delete(&g).Error
}

func (s *GroupService) ListGroups() ([]model.Group, error) {
	var groups []model.Group
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *GroupService) GetGroup(name string) (*model.Group, error) {
	var g model.Group
	if err := s.db.Where("name = ?", name).First(&g).Error; err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	return &g, nil
}

func (s *GroupService) AssignUser(username string, groupName string) error {
	var g model.Group
	if err := s.db.Where("name = ?", groupName).First(&g).Error; err != nil {
		return fmt.Errorf("group not found: %w", err)
	}
	var u model.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	return s.db.Model(&u).Update("group_id", g.ID).Error
}

func (s *GroupService) RemoveUserFromGroup(username string) error {
	var u model.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	return s.db.Model(&u).Update("group_id", nil).Error
}

// MemberCount returns the number of users in a group.
func (s *GroupService) MemberCount(groupID uint) int {
	var count int64
	s.db.Model(&model.User{}).Where("group_id = ?", groupID).Count(&count)
	return int(count)
}

// GetGroupMembers returns all users belonging to the given group.
func (s *GroupService) GetGroupMembers(groupID uint) ([]model.User, error) {
	var users []model.User
	if err := s.db.Where("group_id = ?", groupID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ResolveAllowList returns the effective allow-list for a user.
// Priority: explicit user allow_list > group allow_list > wildcard ["*"]
func (s *GroupService) ResolveAllowList(user *model.User) []string {
	// User has an explicit allow_list set by admin (non-empty array only).
	// len(user.AllowList) checks raw byte length — "[]" is 2 bytes and passes
	// the length guard, so we must also verify the decoded slice is non-empty.
	if len(user.AllowList) > 0 {
		list := make([]string, 0)
		if err := json.Unmarshal(user.AllowList, &list); err == nil && len(list) > 0 {
			return list
		}
	}
	// User belongs to a group — inherit group allow_list
	if user.GroupID != nil {
		var g model.Group
		if err := s.db.First(&g, *user.GroupID).Error; err == nil && len(g.AllowList) > 0 {
			list := make([]string, 0)
			if err := json.Unmarshal(g.AllowList, &list); err == nil {
				return list
			}
		}
	}
	// Default: wildcard
	return []string{"*"}
}
