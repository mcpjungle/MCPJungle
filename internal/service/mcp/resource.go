package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

const resourceURIPrefix = "mcpj://res/"

func buildResourceURI(serverName string, originalURI string) string {
	return resourceURIPrefix + serverName + "/" + base64.RawStdEncoding.EncodeToString([]byte(originalURI))
}

func parseResourceURI(resourceURI string) (string, string, error) {
	if len(resourceURI) <= len(resourceURIPrefix) || resourceURI[:len(resourceURIPrefix)] != resourceURIPrefix {
		return "", "", fmt.Errorf("resource URI %s is not a valid MCPJungle resource URI", resourceURI)
	}

	rest := resourceURI[len(resourceURIPrefix):]
	separatorIndex := -1
	for i := range rest {
		if rest[i] == '/' {
			separatorIndex = i
			break
		}
	}
	if separatorIndex <= 0 || separatorIndex == len(rest)-1 {
		return "", "", fmt.Errorf("resource URI %s is not a valid MCPJungle resource URI", resourceURI)
	}

	serverName := rest[:separatorIndex]
	encodedOriginalURI := rest[separatorIndex+1:]
	if err := validateServerName(serverName); err != nil {
		return "", "", err
	}

	decodedOriginalURI, err := base64.RawStdEncoding.DecodeString(encodedOriginalURI)
	if err != nil {
		return "", "", fmt.Errorf("resource URI %s contains an invalid upstream URI encoding: %w", resourceURI, err)
	}

	return serverName, string(decodedOriginalURI), nil
}

func rewriteResourceContentsURI(contents []mcp.ResourceContents, resourceURI string) []mcp.ResourceContents {
	rewritten := make([]mcp.ResourceContents, 0, len(contents))
	for _, content := range contents {
		if textContent, ok := mcp.AsTextResourceContents(content); ok {
			c := *textContent
			c.URI = resourceURI
			rewritten = append(rewritten, c)
			continue
		}
		if blobContent, ok := mcp.AsBlobResourceContents(content); ok {
			c := *blobContent
			c.URI = resourceURI
			rewritten = append(rewritten, c)
			continue
		}
		rewritten = append(rewritten, content)
	}
	return rewritten
}

// ListResources returns all resources registered in the registry.
// It sets each resource's name to its canonical display form by prepending its server name.
func (m *MCPService) ListResources() ([]model.Resource, error) {
	var resources []model.Resource
	if err := m.db.Preload("Server").Find(&resources).Error; err != nil {
		return nil, err
	}

	for i := range resources {
		resources[i].Name = mergeServerResourceNames(resources[i].Server.Name, resources[i].Name)
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

// GetResource fetches resource metadata by URI, optionally scoped to a specific server.
func (m *MCPService) GetResource(uri string, serverName string) (*model.Resource, error) {
	if uri == "" {
		return nil, fmt.Errorf("resource URI must not be empty")
	}

	resourceServerName, _, err := parseResourceURI(uri)
	if err != nil {
		return nil, err
	}
	if serverName != "" && serverName != resourceServerName {
		return nil, fmt.Errorf("resource URI %s does not belong to server %s", uri, serverName)
	}

	var resource model.Resource
	if err := m.db.Preload("Server").Where("uri = ?", uri).First(&resource).Error; err != nil {
		return nil, fmt.Errorf("failed to get resource %s from DB: %w", uri, err)
	}
	if resource.ID == 0 {
		return nil, fmt.Errorf("resource %s not found", uri)
	}
	resource.Name = mergeServerResourceNames(resource.Server.Name, resource.Name)
	return &resource, nil
}

type resourceMatch struct {
	resource model.Resource
	server   *model.McpServer
}

func (m *MCPService) resolveResourceMatch(ctx context.Context, uri string, serverName string) (*resourceMatch, error) {
	resource, err := m.GetResource(uri, serverName)
	if err != nil {
		return nil, err
	}

	match := resourceMatch{
		resource: *resource,
		server:   &resource.Server,
	}

	if modeValue := ctx.Value("mode"); modeValue != nil {
		if serverMode, ok := modeValue.(model.ServerMode); ok && model.IsEnterpriseMode(serverMode) {
			c := ctx.Value("client").(*model.McpClient)
			if !c.CheckHasServerAccess(match.server.Name) {
				return nil, fmt.Errorf("client %s is not authorized to access resource %s", c.Name, uri)
			}
		}
	}

	return &match, nil
}

// ReadResource reads live resource content by URI, optionally scoped to a specific server.
func (m *MCPService) ReadResource(ctx context.Context, uri string, serverName string) (*types.ResourceReadResult, error) {
	match, err := m.resolveResourceMatch(ctx, uri, serverName)
	if err != nil {
		return nil, err
	}

	session, err := m.getSession(ctx, match.server)
	if err != nil {
		return nil, err
	}
	defer session.closeIfApplicable()

	req := mcp.ReadResourceRequest{}
	req.Params.URI = match.resource.OriginalURI
	res, err := session.client.ReadResource(ctx, req)
	if err != nil {
		session.invalidateOnError(err)
		return nil, err
	}

	contents := make([]map[string]any, 0, len(res.Contents))
	for _, item := range res.Contents {
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize resource content: %w", err)
		}
		var content map[string]any
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, fmt.Errorf("failed to deserialize resource content: %w", err)
		}
		content["uri"] = match.resource.URI
		contents = append(contents, content)
	}

	return &types.ResourceReadResult{Contents: contents}, nil
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
	if err := m.db.Preload("Server").Where("uri = ?", entity).Find(&resources).Error; err != nil {
		return nil, fmt.Errorf("failed to get resources for URI %s: %w", entity, err)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("failed to get resource %s: not found", entity)
	}

	resource := resources[0]
	if resource.Enabled == enabled {
		return []string{resource.URI}, nil
	}
	resource.Enabled = enabled
	if err := m.db.Save(&resource).Error; err != nil {
		return nil, fmt.Errorf("failed to set resource %s enabled=%t: %w", resource.URI, enabled, err)
	}

	if enabled {
		mcpResource, err := convertResourceModelToMcpObject(&resource)
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource model to MCP object for resource %s: %w", resource.URI, err)
		}
		mcpResource.Name = mergeServerResourceNames(resource.Server.Name, mcpResource.Name)

		if resource.Server.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
		} else {
			m.mcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
		}
	} else {
		if resource.Server.Transport == types.TransportSSE {
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
			URI:         buildResourceURI(s.Name, resource.URI),
			OriginalURI: resource.URI,
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

		resource.URI = r.URI
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
