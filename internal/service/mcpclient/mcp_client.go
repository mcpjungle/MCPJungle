// Package mcpclient provides MCP client service functionality for the MCPJungle application.
package mcpclient

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/gorm"
)

// ToolGroupLookup is the interface McpClientService needs to validate bound_tool_group values.
// It is satisfied by *toolgroup.ToolGroupService.
type ToolGroupLookup interface {
	GetToolGroup(name string) (*model.ToolGroup, error)
}

// McpClientService provides methods to manage MCP clients in the database.
type McpClientService struct {
	db             *gorm.DB
	toolGroupSvc   ToolGroupLookup // may be nil when tool-group features are not used
}

func NewMCPClientService(db *gorm.DB) *McpClientService {
	return &McpClientService{db: db}
}

// SetToolGroupService injects the tool-group lookup dependency after construction.
// This breaks the import cycle between mcpclient and toolgroup packages.
func (m *McpClientService) SetToolGroupService(tgs ToolGroupLookup) {
	m.toolGroupSvc = tgs
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

	// If a bound_tool_group is specified, verify that the group exists.
	if client.BoundToolGroup != nil {
		if m.toolGroupSvc == nil {
			return nil, fmt.Errorf("tool group service is not available: %w", apierrors.ErrInvalidInput)
		}
		if _, err := m.toolGroupSvc.GetToolGroup(*client.BoundToolGroup); err != nil {
			return nil, fmt.Errorf("bound_tool_group %q does not exist: %w", *client.BoundToolGroup, apierrors.ErrInvalidInput)
		}
	}

	if err := m.db.Create(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

// CountBoundClients returns the number of MCP clients bound to the specified tool group name.
func (m *McpClientService) CountBoundClients(toolGroupName string) (int64, error) {
	var count int64
	err := m.db.Model(&model.McpClient{}).Where("bound_tool_group = ?", toolGroupName).Count(&count).Error
	return count, err
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
// Currently, it supports updating the access token and bound_tool_group of the client.
func (m *McpClientService) UpdateClient(updatedClient model.McpClient) (*model.McpClient, error) {
	var client model.McpClient
	if err := m.db.Where("name = ?", updatedClient.Name).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}

	if err := internal.ValidateAccessToken(updatedClient.AccessToken); err != nil {
		return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
	}

	// If a new bound_tool_group is specified, verify that the group exists.
	if updatedClient.BoundToolGroup != nil {
		if m.toolGroupSvc == nil {
			return nil, fmt.Errorf("tool group service is not available: %w", apierrors.ErrInvalidInput)
		}
		if _, err := m.toolGroupSvc.GetToolGroup(*updatedClient.BoundToolGroup); err != nil {
			return nil, fmt.Errorf("bound_tool_group %q does not exist: %w", *updatedClient.BoundToolGroup, apierrors.ErrInvalidInput)
		}
	}

	// Update access token and bound_tool_group
	client.AccessToken = updatedClient.AccessToken
	client.BoundToolGroup = updatedClient.BoundToolGroup

	if err := m.db.Save(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}
