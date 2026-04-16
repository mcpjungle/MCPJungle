package mcp

import (
	"context"
	"errors"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBWithResources(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.McpServer{}, &model.Tool{}, &model.Prompt{}, &model.Resource{})
	require.NoError(t, err)

	return db
}

func createTestResource(t *testing.T, db *gorm.DB, server *model.McpServer, originalURI, name string) *model.Resource {
	resource := &model.Resource{
		URI:         buildResourceURI(server.Name, originalURI),
		OriginalURI: originalURI,
		Name:        name,
		Description: "Test resource",
		MIMEType:    "text/plain",
		Enabled:     true,
		ServerID:    server.ID,
	}
	err := db.Create(resource).Error
	require.NoError(t, err)
	return resource
}

func TestListResources(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{db: db}

	srv := createTestServer(t, db)
	createTestResource(t, db, srv, "resource://test/code-review", "code-review")
	createTestResource(t, db, srv, "resource://test/security-audit", "security-audit")

	resources, err := service.ListResources()
	require.NoError(t, err)
	assert.Len(t, resources, 2)

	expectedNames := []string{
		"test-server__code-review",
		"test-server__security-audit",
	}
	actualNames := []string{resources[0].Name, resources[1].Name}
	assert.ElementsMatch(t, expectedNames, actualNames)
	assert.Equal(t, buildResourceURI("test-server", "resource://test/code-review"), resources[0].URI)
}

func TestListResourcesByServer(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{db: db}

	srv := createTestServer(t, db)
	createTestResource(t, db, srv, "resource://test/code-review", "code-review")

	resources, err := service.ListResourcesByServer("test-server")
	require.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "test-server__code-review", resources[0].Name)
	assert.Equal(t, buildResourceURI("test-server", "resource://test/code-review"), resources[0].URI)
}

func TestEnableDisableResources(t *testing.T) {
	db := setupTestDBWithResources(t)

	mcpProxyServer := server.NewMCPServer("Test Proxy", "0.1.0")
	service := &MCPService{
		db:             db,
		mcpProxyServer: mcpProxyServer,
	}

	srv := createTestServer(t, db)
	resource := createTestResource(t, db, srv, "resource://test/code-review", "code-review")

	disabledResources, err := service.DisableResources(resource.URI)
	require.NoError(t, err)
	assert.Len(t, disabledResources, 1)
	assert.Equal(t, resource.URI, disabledResources[0])

	var updatedResource model.Resource
	err = db.First(&updatedResource, resource.ID).Error
	require.NoError(t, err)
	assert.False(t, updatedResource.Enabled)

	enabledResources, err := service.EnableResources(resource.URI)
	require.NoError(t, err)
	assert.Len(t, enabledResources, 1)
	assert.Equal(t, resource.URI, enabledResources[0])

	err = db.First(&updatedResource, resource.ID).Error
	require.NoError(t, err)
	assert.True(t, updatedResource.Enabled)
}

func TestEnableDisableServerResources(t *testing.T) {
	db := setupTestDBWithResources(t)

	mcpProxyServer := server.NewMCPServer("Test Proxy", "0.1.0")
	service := &MCPService{
		db:             db,
		mcpProxyServer: mcpProxyServer,
	}

	srv := createTestServer(t, db)
	createTestResource(t, db, srv, "resource://test/code-review", "code-review")
	createTestResource(t, db, srv, "resource://test/security-audit", "security-audit")

	disabledResources, err := service.DisableResources("test-server")
	require.NoError(t, err)
	assert.Len(t, disabledResources, 2)

	var resources []model.Resource
	err = db.Where("server_id = ?", srv.ID).Find(&resources).Error
	require.NoError(t, err)
	for _, resource := range resources {
		assert.False(t, resource.Enabled)
	}

	enabledResources, err := service.EnableResources("test-server")
	require.NoError(t, err)
	assert.Len(t, enabledResources, 2)

	err = db.Where("server_id = ?", srv.ID).Find(&resources).Error
	require.NoError(t, err)
	for _, resource := range resources {
		assert.True(t, resource.Enabled)
	}
}

func TestDisableResourcesByPublicURI(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{
		db:             db,
		mcpProxyServer: server.NewMCPServer("Test Proxy", "0.1.0"),
	}

	srv1 := createTestServer(t, db)
	srv2, err := model.NewStdioServer(
		"test-server-2",
		"Test MCP server 2",
		"echo",
		[]string{"hello"},
		nil,
		types.SessionModeStateful,
	)
	require.NoError(t, err)
	err = db.Create(srv2).Error
	require.NoError(t, err)

	createTestResource(t, db, srv1, "resource://shared/status", "status")
	createTestResource(t, db, srv2, "resource://shared/status", "status")

	resourceURI := buildResourceURI("test-server", "resource://shared/status")
	disabledResources, err := service.DisableResources(resourceURI)
	require.NoError(t, err)
	assert.Equal(t, []string{resourceURI}, disabledResources)
}

func TestGetResourceInvalidURI(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{db: db}

	_, err := service.GetResource("not-a-mcpj-uri")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrInvalidInput))
}

func TestGetResourceNotFound(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{db: db}

	srv := createTestServer(t, db)
	missingURI := buildResourceURI(srv.Name, "resource://missing")

	_, err := service.GetResource(missingURI)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrNotFound))
}

func TestDisableResourcesInvalidURI(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{
		db:             db,
		mcpProxyServer: server.NewMCPServer("Test Proxy", "0.1.0"),
	}

	_, err := service.DisableResources("not-a-mcpj-uri")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrInvalidInput))
}

func TestDisableResourcesNotFound(t *testing.T) {
	db := setupTestDBWithResources(t)
	service := &MCPService{
		db:             db,
		mcpProxyServer: server.NewMCPServer("Test Proxy", "0.1.0"),
	}

	srv := createTestServer(t, db)
	missingURI := buildResourceURI(srv.Name, "resource://missing")

	_, err := service.DisableResources(missingURI)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrNotFound))
}

func TestMCPProxyResourceHandlerRoutesReadByURI(t *testing.T) {
	db := setupTestDBWithResources(t)
	srv := createTestServer(t, db)
	srv.SessionMode = "stateful"
	err := db.Save(srv).Error
	require.NoError(t, err)
	createTestResource(t, db, srv, "resource://test/status", "status")

	upstreamServer := server.NewMCPServer("Upstream", "0.1.0")
	upstreamServer.AddResource(
		mcp.Resource{
			URI:         "resource://test/status",
			Name:        "status",
			Description: "Current status",
			MIMEType:    "text/plain",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     "ok",
				},
			}, nil
		},
	)

	sessionManager := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    0,
		InitReqTimeoutSec: 10,
	})
	sessionManager.createSessionFunc = func(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcpclient.Client, error) {
		client, err := mcpclient.NewInProcessClient(upstreamServer)
		if err != nil {
			return nil, err
		}
		if err := client.Start(ctx); err != nil {
			return nil, err
		}
		initReq := mcp.InitializeRequest{}
		initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initReq.Params.ClientInfo = mcp.Implementation{Name: "test-client", Version: "1.0.0"}
		if _, err := client.Initialize(ctx, initReq); err != nil {
			return nil, err
		}
		return client, nil
	}

	service := &MCPService{
		db:                         db,
		mcpProxyServer:             server.NewMCPServer("Test Proxy", "0.1.0"),
		sseMcpProxyServer:          server.NewMCPServer("Test Proxy SSE", "0.1.0"),
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 10,
		sessionManager:             sessionManager,
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = buildResourceURI("test-server", "resource://test/status")
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)

	contents, err := service.mcpProxyResourceHandler(ctx, req)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, "ok", textContent.Text)
	assert.Equal(t, buildResourceURI("test-server", "resource://test/status"), textContent.URI)
}

func TestMCPProxyResourceHandlerEnterpriseRejectsUnauthorizedClient(t *testing.T) {
	db := setupTestDBWithResources(t)
	srv := createTestServer(t, db)
	createTestResource(t, db, srv, "resource://test/status", "status")

	sessionCreated := false
	sessionManager := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    0,
		InitReqTimeoutSec: 10,
	})
	sessionManager.createSessionFunc = func(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcpclient.Client, error) {
		sessionCreated = true
		return nil, errors.New("session should not be created for unauthorized client")
	}

	service := &MCPService{
		db:                         db,
		mcpProxyServer:             server.NewMCPServer("Test Proxy", "0.1.0"),
		sseMcpProxyServer:          server.NewMCPServer("Test Proxy SSE", "0.1.0"),
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 10,
		sessionManager:             sessionManager,
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = buildResourceURI("test-server", "resource://test/status")
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "client", &model.McpClient{
		Name:      "scoped-client",
		AllowList: datatypes.JSON(`["other-server"]`),
	})

	_, err := service.mcpProxyResourceHandler(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authorized to access MCP server test-server")
	assert.False(t, sessionCreated)
}

func TestMCPProxyResourceHandlerEnterpriseAllowsAuthorizedClient(t *testing.T) {
	db := setupTestDBWithResources(t)
	srv := createTestServer(t, db)
	srv.SessionMode = types.SessionModeStateful
	err := db.Save(srv).Error
	require.NoError(t, err)
	createTestResource(t, db, srv, "resource://test/status", "status")

	upstreamServer := server.NewMCPServer("Upstream", "0.1.0")
	upstreamServer.AddResource(
		mcp.Resource{
			URI:         "resource://test/status",
			Name:        "status",
			Description: "Current status",
			MIMEType:    "text/plain",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     "ok",
				},
			}, nil
		},
	)

	sessionManager := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    0,
		InitReqTimeoutSec: 10,
	})
	sessionManager.createSessionFunc = func(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcpclient.Client, error) {
		client, err := mcpclient.NewInProcessClient(upstreamServer)
		if err != nil {
			return nil, err
		}
		if err := client.Start(ctx); err != nil {
			return nil, err
		}
		initReq := mcp.InitializeRequest{}
		initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initReq.Params.ClientInfo = mcp.Implementation{Name: "test-client", Version: "1.0.0"}
		if _, err := client.Initialize(ctx, initReq); err != nil {
			return nil, err
		}
		return client, nil
	}

	service := &MCPService{
		db:                         db,
		mcpProxyServer:             server.NewMCPServer("Test Proxy", "0.1.0"),
		sseMcpProxyServer:          server.NewMCPServer("Test Proxy SSE", "0.1.0"),
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 10,
		sessionManager:             sessionManager,
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = buildResourceURI("test-server", "resource://test/status")
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "client", &model.McpClient{
		Name:      "scoped-client",
		AllowList: datatypes.JSON(`["test-server"]`),
	})

	contents, err := service.mcpProxyResourceHandler(ctx, req)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, "ok", textContent.Text)
	assert.Equal(t, buildResourceURI("test-server", "resource://test/status"), textContent.URI)
}

func TestMCPProxyResourceHandlerRoutesDuplicateUpstreamURIs(t *testing.T) {
	db := setupTestDBWithResources(t)
	srv1 := createTestServer(t, db)
	srv2, err := model.NewStdioServer(
		"test-server-2",
		"Test MCP server 2",
		"echo",
		[]string{"hello"},
		nil,
		types.SessionModeStateful,
	)
	require.NoError(t, err)
	err = db.Create(srv2).Error
	require.NoError(t, err)

	createTestResource(t, db, srv1, "resource://shared/status", "status")
	createTestResource(t, db, srv2, "resource://shared/status", "status")

	proxyServer := server.NewMCPServer("Test Proxy", "0.1.0")
	upstreamServer := server.NewMCPServer("Upstream", "0.1.0")
	upstreamServer.AddResource(
		mcp.Resource{
			URI:         "resource://shared/status",
			Name:        "status",
			Description: "Current status",
			MIMEType:    "text/plain",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     "from second server",
				},
			}, nil
		},
	)

	sessionManager := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    0,
		InitReqTimeoutSec: 10,
	})
	sessionManager.createSessionFunc = func(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcpclient.Client, error) {
		client, err := mcpclient.NewInProcessClient(upstreamServer)
		if err != nil {
			return nil, err
		}
		if err := client.Start(ctx); err != nil {
			return nil, err
		}
		initReq := mcp.InitializeRequest{}
		initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initReq.Params.ClientInfo = mcp.Implementation{Name: "test-client", Version: "1.0.0"}
		if _, err := client.Initialize(ctx, initReq); err != nil {
			return nil, err
		}
		return client, nil
	}

	service := &MCPService{
		db:                         db,
		mcpProxyServer:             proxyServer,
		sseMcpProxyServer:          proxyServer,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 10,
		sessionManager:             sessionManager,
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = buildResourceURI("test-server-2", "resource://shared/status")
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)

	resource, err := service.GetResource(req.Params.URI)
	require.NoError(t, err)
	require.Equal(t, types.SessionModeStateful, resource.Server.SessionMode)

	contents, err := service.mcpProxyResourceHandler(ctx, req)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, "from second server", textContent.Text)
	assert.Equal(t, buildResourceURI("test-server-2", "resource://shared/status"), textContent.URI)
}
