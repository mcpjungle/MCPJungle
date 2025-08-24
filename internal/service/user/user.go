package user

import (
	"errors"
	"fmt"
	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
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
		Role:        model.UserRoleAdmin,
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
	if user.Role != model.UserRoleAdmin {
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
		Role:        model.UserRoleUser,
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
func (u *UserService) DeleteUser(username string) error {
	if err := u.db.Where("username = ?", username).Delete(&model.User{}).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
