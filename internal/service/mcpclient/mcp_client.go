// Package mcpclient provides MCP client service functionality for the MCPJungle application.
package mcpclient

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/gorm"
)

// McpClientService provides methods to manage MCP clients in the database.
type McpClientService struct {
	db *gorm.DB
}

type UpdateClientInput struct {
	Name              string
	Description       string
	AllowList         []string
	AccessToken       string
	RotateAccessToken bool
}

func NewMCPClientService(db *gorm.DB) *McpClientService {
	return &McpClientService{db: db}
}

// ListClients retrieves all MCP clients known to mcpjungle from the database
func (m *McpClientService) ListClients() ([]*model.McpClient, error) {
	var clients []*model.McpClient
	if err := m.db.Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

// CreateClient creates a new MCP client in the database.
// It also generates a new access token for the client.
func (m *McpClientService) CreateClient(client model.McpClient) (*model.McpClient, error) {
	if client.AccessToken != "" {
		// user has supplied a custom access token, validate it
		if err := internal.ValidateAccessToken(client.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		// todo: add audit log entry for custom token usage
	} else {
		// no access token is provided by user, generate a new one
		token, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate access token: %w", err)
		}
		client.AccessToken = token
	}

	// Initialize AllowList with empty array if not provided
	if client.AllowList == nil {
		client.AllowList = []byte("[]")
	}

	if err := m.db.Create(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

// GetClientByToken retrieves an MCP client by its access token from the database.
// It returns an error if no such client is found.
func (m *McpClientService) GetClientByToken(token string) (*model.McpClient, error) {
	var client model.McpClient
	if err := m.db.Where("access_token = ?", token).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &client, nil
}

// DeleteClient removes an MCP client from the database and immediately revokes its access.
// It is an idempotent operation. Deleting a client that does not exist will not return an error.
func (m *McpClientService) DeleteClient(name string) error {
	result := m.db.Unscoped().Where("name = ?", name).Delete(&model.McpClient{})
	return result.Error
}

// UpdateClient updates an existing MCP client's information in the database.
func (m *McpClientService) UpdateClient(input UpdateClientInput) (*model.McpClient, error) {
	var client model.McpClient
	if err := m.db.Where("name = ?", input.Name).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}

	if input.AccessToken != "" && input.RotateAccessToken {
		return nil, fmt.Errorf("access_token and rotate_access_token cannot both be set: %w", apierrors.ErrInvalidInput)
	}

	client.Description = input.Description

	allowListJSON, err := json.Marshal(input.AllowList)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allow list: %w", err)
	}
	client.AllowList = allowListJSON

	switch {
	case input.AccessToken != "":
		if err := internal.ValidateAccessToken(input.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		client.AccessToken = input.AccessToken
	case input.RotateAccessToken:
		token, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate access token: %w", err)
		}
		client.AccessToken = token
	}

	if err := m.db.Save(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}
