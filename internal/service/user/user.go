package user

import (
	"errors"
	"fmt"
	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// UserService provides methods to manage users in the MCPJungle system.
type UserService struct {
	db *gorm.DB
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

// VerifyAdminToken checks if the provided token belongs to an admin user
func (u *UserService) VerifyAdminToken(token string) (*model.User, error) {
	var user model.User
	if err := u.db.Where("access_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("admin user not found")
		}
		return nil, fmt.Errorf("failed to verify admin token: %w", err)
	}
	if user.Role != types.UserRoleAdmin {
		return nil, fmt.Errorf("user is not an admin")
	}
	return &user, nil
}

// CreateUser creates a new user with the specified username.
// A user is a standard, authenticated, human user who has lesser privileges than an admin user.
func (u *UserService) CreateUser(username string) (*model.User, error) {
	token, err := internal.GenerateAccessToken()
	if err != nil {
		return nil, err
	}
	user := model.User{
		Username:    username,
		Role:        types.UserRoleUser,
		AccessToken: token,
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
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

// DeleteUser removes a user with the specified username from the database.
// If a user's role is admin, the deletion will be rejected.
func (u *UserService) DeleteUser(username string) error {
	var user model.User
	err := u.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user with username %s not found", username)
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	if user.Role == types.UserRoleAdmin {
		return fmt.Errorf("cannot delete an admin user")
	}

	err = u.db.Unscoped().Where("username = ?", username).Delete(&model.User{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
