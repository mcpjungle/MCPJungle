// Package user provides user service functionality for the MCPJungle application.
package user

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// UserService provides methods to manage users in the MCPJungle system.
type UserService struct {
	db *gorm.DB
}

type UpdateUserInput struct {
	Username          string
	AccessToken       string
	RotateAccessToken bool
	AllowList         []string // nil = no change, empty slice = wildcard (*), non-empty = restrict
	UpdateAllowList   bool     // explicit flag so empty slice is distinguishable from "not provided"
	ClearAllowList    bool     // if true, set AllowList = NULL (revert to group inheritance)
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// CreateAdminUser creates an admin user in the MCPJungle system.
func (u *UserService) CreateAdminUser() (*model.User, error) {
	token, err := internal.GenerateAccessToken()
	if err != nil {
		return nil, err
	}
	user := model.User{
		Username:    "admin",
		Role:        types.UserRoleAdmin,
		AccessToken: token,
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}
	return &user, nil
}

// GetUserByAccessToken returns a user associated with the provided access token.
// If no user is found, an error is returned.
func (u *UserService) GetUserByAccessToken(token string) (*model.User, error) {
	var user model.User
	if err := u.db.Where("access_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %w", apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	return &user, nil
}

// CreateUser creates a new user with the specified username.
// This method currently only supports creating a standard user, ie, user with the "user" role.
func (u *UserService) CreateUser(input *model.User) (*model.User, error) {
	user := model.User{
		Username:  input.Username,
		Role:      types.UserRoleUser,
		AllowList: input.AllowList, // nil = inherit from group (or wildcard default)
	}
	if input.AccessToken == "" {
		// no custom access token provided, generate a new one
		token, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, err
		}
		user.AccessToken = token
	} else {
		// validate the user-provided custom access token
		if err := internal.ValidateAccessToken(input.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		user.AccessToken = input.AccessToken
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user's access token.
func (u *UserService) UpdateUser(input UpdateUserInput) (*model.User, error) {
	var user model.User
	err := u.db.Where("username = ?", input.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with username %s not found: %w", input.Username, apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if input.AccessToken != "" && input.RotateAccessToken {
		return nil, fmt.Errorf("access_token and rotate_access_token cannot both be set: %w", apierrors.ErrInvalidInput)
	}

	switch {
	case input.AccessToken != "":
		if err := internal.ValidateAccessToken(input.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		user.AccessToken = input.AccessToken
	case input.RotateAccessToken:
		token, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate access token: %w", err)
		}
		user.AccessToken = token
	}

	if input.UpdateAllowList {
		// Explicitly marshal whatever was sent — empty slice = no access, ["*"] = wildcard.
		// Never substitute a default here; the caller is making a deliberate choice.
		encoded, err := json.Marshal(input.AllowList)
		if err != nil {
			return nil, fmt.Errorf("invalid allow_list: %w", err)
		}
		user.AllowList = encoded
	}

	if input.ClearAllowList {
		user.AllowList = nil
	}

	err = u.db.Save(&user).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &user, nil
}

// ListUsers retrieves all users from the database.
func (u *UserService) ListUsers() ([]model.User, error) {
	var users []model.User
	if err := u.db.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// ListUsersWithGroup retrieves all users from the database with their Group preloaded.
func (u *UserService) ListUsersWithGroup() ([]model.User, error) {
	var users []model.User
	if err := u.db.Preload("Group").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// DeleteUser removes a user with the specified username from the database.
// If a user's role is admin, the deletion will be rejected.
func (u *UserService) DeleteUser(username string) error {
	var user model.User
	err := u.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user with username %s not found: %w", username, apierrors.ErrNotFound)
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	if user.Role == types.UserRoleAdmin {
		return fmt.Errorf("cannot delete an admin user: %w", apierrors.ErrInvalidInput)
	}

	err = u.db.Unscoped().Where("username = ?", username).Delete(&model.User{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
