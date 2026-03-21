package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListResources returns all resources registered in the registry.
// It sets each resource's name to its canonical display form by prepending its server name.
func (m *MCPService) ListResources() ([]model.Resource, error) {
	var resources []model.Resource
	if err := m.db.Find(&resources).Error; err != nil {
		return nil, err
	}

	serverCache := make(map[uint]string)
	for i := range resources {
		serverName, ok := serverCache[resources[i].ServerID]
		if !ok {
			var s model.McpServer
			if err := m.db.First(&s, "id = ?", resources[i].ServerID).Error; err != nil {
				return nil, fmt.Errorf("failed to get server for resource %s: %w", resources[i].URI, err)
			}
			serverName = s.Name
			serverCache[resources[i].ServerID] = serverName
		}

		resources[i].Name = mergeServerResourceNames(serverName, resources[i].Name)
	}

	return resources, nil
}

// ListResourcesByServer fetches resources provided by an MCP server from the registry.
func (m *MCPService) ListResourcesByServer(name string) ([]model.Resource, error) {
	if err := validateServerName(name); err != nil {
		return nil, err
	}

	s, err := m.GetMcpServer(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", name, err)
	}

	var resources []model.Resource
	if err := m.db.Where("server_id = ?", s.ID).Find(&resources).Error; err != nil {
		return nil, fmt.Errorf("failed to get resources for server %s from DB: %w", name, err)
	}

	for i := range resources {
		resources[i].Name = mergeServerResourceNames(s.Name, resources[i].Name)
	}

	return resources, nil
}

// GetResourcesByURI fetches enabled resources matching a URI.
func (m *MCPService) GetResourcesByURI(uri string) ([]model.Resource, error) {
	var resources []model.Resource
	if err := m.db.Where("uri = ? AND enabled = ?", uri, true).Find(&resources).Error; err != nil {
		return nil, fmt.Errorf("failed to get resources for URI %s from DB: %w", uri, err)
	}
	return resources, nil
}

// EnableResources enables one or more resources.
// If the entity is a server name, all resources of that server are enabled.
// Otherwise, the entity is treated as a resource URI and the matching resource is enabled
// only when the URI uniquely identifies a single registered resource.
func (m *MCPService) EnableResources(entity string) ([]string, error) {
	return m.setResourcesEnabled(entity, true)
}

// DisableResources disables one or more resources.
// If the entity is a server name, all resources of that server are disabled.
// Otherwise, the entity is treated as a resource URI and the matching resource is disabled
// only when the URI uniquely identifies a single registered resource.
func (m *MCPService) DisableResources(entity string) ([]string, error) {
	return m.setResourcesEnabled(entity, false)
}

func (m *MCPService) setResourcesEnabled(entity string, enabled bool) ([]string, error) {
	if validateServerName(entity) == nil {
		if s, err := m.GetMcpServer(entity); err == nil {
			return m.setServerResourcesEnabled(s, enabled)
		}
	}

	var resources []model.Resource
	if err := m.db.Where("uri = ?", entity).Find(&resources).Error; err != nil {
		return nil, fmt.Errorf("failed to get resources for URI %s: %w", entity, err)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("failed to get resource %s: not found", entity)
	}
	if len(resources) > 1 {
		return nil, fmt.Errorf("resource URI %s is ambiguous across multiple servers", entity)
	}

	resource := resources[0]
	if resource.Enabled == enabled {
		return []string{resource.URI}, nil
	}
	resource.Enabled = enabled
	if err := m.db.Save(&resource).Error; err != nil {
		return nil, fmt.Errorf("failed to set resource %s enabled=%t: %w", resource.URI, enabled, err)
	}

	s, err := m.GetServerByID(resource.ServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server for resource %s: %w", resource.URI, err)
	}

	if enabled {
		mcpResource, err := convertResourceModelToMcpObject(&resource)
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource model to MCP object for resource %s: %w", resource.URI, err)
		}
		mcpResource.Name = mergeServerResourceNames(s.Name, mcpResource.Name)

		if s.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
		} else {
			m.mcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
		}
	} else {
		if s.Transport == types.TransportSSE {
			m.sseMcpProxyServer.DeleteResources(resource.URI)
		} else {
			m.mcpProxyServer.DeleteResources(resource.URI)
		}
	}

	return []string{resource.URI}, nil
}

func (m *MCPService) setServerResourcesEnabled(s *model.McpServer, enabled bool) ([]string, error) {
	var resources []model.Resource
	if err := m.db.Where("server_id = ?", s.ID).Find(&resources).Error; err != nil {
		return nil, fmt.Errorf("failed to get resources for server %s: %w", s.Name, err)
	}

	var changedURIs []string
	for i := range resources {
		if resources[i].Enabled == enabled {
			continue
		}
		resources[i].Enabled = enabled
		if err := m.db.Save(&resources[i]).Error; err != nil {
			return nil, fmt.Errorf("failed to set resource %s enabled=%t: %w", resources[i].URI, enabled, err)
		}

		if enabled {
			mcpResource, err := convertResourceModelToMcpObject(&resources[i])
			if err != nil {
				return nil, fmt.Errorf("failed to convert resource model to MCP object for resource %s: %w", resources[i].URI, err)
			}
			mcpResource.Name = mergeServerResourceNames(s.Name, mcpResource.Name)

			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
			} else {
				m.mcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
			}
		} else {
			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.DeleteResources(resources[i].URI)
			} else {
				m.mcpProxyServer.DeleteResources(resources[i].URI)
			}
		}

		changedURIs = append(changedURIs, resources[i].URI)
	}

	return changedURIs, nil
}

// registerServerResources fetches all resources from an MCP server and registers them in the DB.
func (m *MCPService) registerServerResources(ctx context.Context, s *model.McpServer, c *client.Client) error {
	resp, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch resources from MCP server %s: %w", s.Name, err)
	}

	for _, resource := range resp.Resources {
		canonicalResourceName := mergeServerResourceNames(s.Name, resource.GetName())

		annotationsJSON, _ := json.Marshal(resource.Annotations)
		metaJSON, _ := json.Marshal(resource.Meta)

		r := &model.Resource{
			ServerID:    s.ID,
			URI:         resource.URI,
			Name:        resource.GetName(),
			Description: resource.Description,
			MIMEType:    resource.MIMEType,
			Annotations: annotationsJSON,
			Meta:        metaJSON,
		}
		if err := m.db.Create(r).Error; err != nil {
			log.Printf("[ERROR] failed to register resource %s (%s) in DB: %v", canonicalResourceName, resource.URI, err)
			continue
		}

		resource.Name = canonicalResourceName
		if s.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddResource(resource, m.mcpProxyResourceHandler)
		} else {
			m.mcpProxyServer.AddResource(resource, m.mcpProxyResourceHandler)
		}
	}

	return nil
}

// deregisterServerResources deletes all resources that belong to an MCP server from the DB.
// It also removes the resources from the MCP proxy server.
func (m *MCPService) deregisterServerResources(s *model.McpServer) error {
	var resources []model.Resource
	if err := m.db.Where("server_id = ?", s.ID).Find(&resources).Error; err != nil {
		return fmt.Errorf("failed to list resources for server %s: %w", s.Name, err)
	}

	result := m.db.Unscoped().Where("server_id = ?", s.ID).Delete(&model.Resource{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete resources for server %s: %w", s.Name, result.Error)
	}

	resourceURIs := make([]string, len(resources))
	for i, resource := range resources {
		resourceURIs[i] = resource.URI
	}

	if s.Transport == types.TransportSSE {
		m.sseMcpProxyServer.DeleteResources(resourceURIs...)
	} else {
		m.mcpProxyServer.DeleteResources(resourceURIs...)
	}

	return nil
}
