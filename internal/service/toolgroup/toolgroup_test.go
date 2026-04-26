package toolgroup

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestValidGroupNameRegex(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"valid-group", true},
		{"valid_group", true},
		{"validGroup", true},
		{"group123", true},
		{"123group", true}, // starts with number (allowed by regex)
		{"-group", false},  // starts with hyphen
		{"_group", false},  // starts with underscore
		{"", false},        // empty
		{"group-name", true},
		{"group_name", true},
		{"group name", false}, // contains space
		{"group@name", false}, // contains special character
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := ValidGroupName.MatchString(tt.name)
			testhelpers.AssertEqual(t, tt.valid, isValid)
		})
	}
}

func TestValidGroupNameEdgeCases(t *testing.T) {
	// Test very long names
	longName := "a"
	for i := 0; i < 100; i++ {
		longName += "a"
	}

	isValid := ValidGroupName.MatchString(longName)
	testhelpers.AssertTrue(t, isValid, "Expected long name to be valid")

	// Test single character names
	singleCharNames := []string{"a", "A", "1", "0"}
	for _, name := range singleCharNames {
		isValid := ValidGroupName.MatchString(name)
		testhelpers.AssertTrue(t, isValid, "Expected single character name '"+name+"' to be valid")
	}

	// Test names with mixed characters
	mixedNames := []string{"a1-b_c", "A1-B_C", "test-123_group"}
	for _, name := range mixedNames {
		isValid := ValidGroupName.MatchString(name)
		testhelpers.AssertTrue(t, isValid, "Expected mixed name '"+name+"' to be valid")
	}
}

func TestValidGroupNameUnicode(t *testing.T) {
	// Test that the regex only allows ASCII characters
	unicodeNames := []string{"group-ñ", "group-é", "group-ü", "group-ß"}
	for _, name := range unicodeNames {
		isValid := ValidGroupName.MatchString(name)
		testhelpers.AssertFalse(t, isValid, "Expected unicode name '"+name+"' to be invalid")
	}
}

func TestValidGroupNameSpecialCharacters(t *testing.T) {
	// Test various special characters that should not be allowed
	specialChars := []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "+", "=", "[", "]", "{", "}", "|", "\\", ":", ";", "\"", "'", "<", ">", ",", ".", "?", "/"}

	for _, char := range specialChars {
		name := "group" + char
		isValid := ValidGroupName.MatchString(name)
		testhelpers.AssertFalse(t, isValid, "Expected name with special character '"+char+"' to be invalid")
	}
}

func TestValidGroupNameBoundaryConditions(t *testing.T) {
	// Test names that are exactly at the boundary of what's allowed
	boundaryNames := []string{
		"a",  // single lowercase letter
		"A",  // single uppercase letter
		"0",  // single digit
		"a0", // letter followed by digit
		"0a", // digit followed by letter (allowed by regex)
		"a-", // letter followed by hyphen (allowed by regex)
		"a_", // letter followed by underscore (allowed by regex)
	}

	expectedResults := []bool{true, true, true, true, true, true, true}

	for i, name := range boundaryNames {
		isValid := ValidGroupName.MatchString(name)
		expected := expectedResults[i]
		if isValid != expected {
			t.Errorf("Expected '%s' to be %v, got %v", name, expected, isValid)
		}
	}
}

func TestValidGroupNamePerformance(t *testing.T) {
	// Test that the regex performs reasonably well with long strings
	longName := "a"
	for i := 0; i < 1000; i++ {
		longName += "a"
	}

	// This should complete quickly
	isValid := ValidGroupName.MatchString(longName)
	if !isValid {
		t.Errorf("Expected very long name to be valid")
	}
}

func TestValidGroupNameConsistency(t *testing.T) {
	// Test that the same input always produces the same result
	testName := "test-group-name"

	// Run the test multiple times to ensure consistency
	for i := 0; i < 100; i++ {
		result1 := ValidGroupName.MatchString(testName)
		result2 := ValidGroupName.MatchString(testName)

		if result1 != result2 {
			t.Errorf("Regex results inconsistent for '%s': got %v and %v", testName, result1, result2)
		}
	}
}

func setupInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := testhelpers.CreateTestDB()
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.McpServer{}, &model.Tool{}, &model.ToolGroup{}, &model.Prompt{}, &model.Resource{}); err != nil {
		t.Fatalf("failed to migrate test models: %v", err)
	}
	return db
}

func newTestMCPService(t *testing.T, db *gorm.DB) *mcp.MCPService {
	t.Helper()

	proxyServer := server.NewMCPServer(
		"test proxy",
		"0.0.1",
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)
	sseProxyServer := server.NewMCPServer(
		"test sse proxy",
		"0.0.1",
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)

	svc, err := mcp.NewMCPService(&mcp.ServiceConfig{
		DB:                      db,
		McpProxyServer:          proxyServer,
		SseMcpProxyServer:       sseProxyServer,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 10,
	})
	if err != nil {
		t.Fatalf("failed to create MCP service: %v", err)
	}

	return svc
}

func TestResolveEffectiveTools_GroupNotFound(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolGroupService{
		db:         db,
		mcpService: &mcp.MCPService{}, // zero value is fine for this test
	}

	_, err := s.ResolveEffectiveTools("nonexistent-group")
	if !errors.Is(err, ErrToolGroupNotFound) {
		t.Fatalf("expected ErrToolGroupNotFound, got: %v", err)
	}
}

func TestResolveEffectiveTools_ReturnsSorted(t *testing.T) {
	db := setupInMemoryDB(t)

	// Create a ToolGroup that contains an unsorted list of tools.
	// The model.ToolGroup implementation is expected to return those tools
	// (or otherwise resolve them); this test asserts that ResolveEffectiveTools
	// sorts the result before returning.
	group := model.ToolGroup{
		Name:          "my-group",
		IncludedTools: datatypes.JSON([]byte(`["tool-b","tool-a","tool-c"]`)),
	}

	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create tool group: %v", err)
	}

	s := &ToolGroupService{
		db:         db,
		mcpService: &mcp.MCPService{},
	}

	tools, err := s.ResolveEffectiveTools("my-group")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect sorted order
	expected := []string{"tool-a", "tool-b", "tool-c"}
	// If the underlying ResolveEffectiveTools implementation returns a different
	// set because it resolves servers/excludes, adapt expectations accordingly.
	// For the common case where explicit tools were stored, assert sorting.
	if !reflect.DeepEqual(tools, expected) {
		t.Fatalf("expected sorted tools %v, got %v", expected, tools)
	}
}

func TestCreateToolGroup_InvalidNameReturnsInvalidInput(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolGroupService{
		db:         db,
		mcpService: &mcp.MCPService{},
	}

	err := s.CreateToolGroup(&model.ToolGroup{Name: "-bad-group"})
	if !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestCreateToolGroup_EmptyResolvedToolsReturnsInvalidInput(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolGroupService{
		db:         db,
		mcpService: &mcp.MCPService{},
	}

	err := s.CreateToolGroup(&model.ToolGroup{Name: "empty-group"})
	if !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestNewToolGroupService_DegradedPersistedGroupDoesNotFailStartup(t *testing.T) {
	db := setupInMemoryDB(t)

	validServer, err := model.NewStdioServer("valid-server", "Valid server", "echo", nil, nil, "")
	if err != nil {
		t.Fatalf("failed to create valid server model: %v", err)
	}
	if err := db.Create(validServer).Error; err != nil {
		t.Fatalf("failed to persist valid server: %v", err)
	}

	validTool := model.Tool{
		ServerID:    validServer.ID,
		Name:        "sum",
		Description: "adds numbers",
		InputSchema: []byte(`{"type":"object"}`),
		Enabled:     true,
	}
	if err := db.Create(&validTool).Error; err != nil {
		t.Fatalf("failed to persist valid tool: %v", err)
	}

	validGroup := model.ToolGroup{
		Name:            "valid-group",
		IncludedServers: datatypes.JSON([]byte(`["valid-server"]`)),
	}
	degradedGroup := model.ToolGroup{
		Name:            "degraded-group",
		IncludedServers: datatypes.JSON([]byte(`["missing-server"]`)),
	}
	if err := db.Create(&validGroup).Error; err != nil {
		t.Fatalf("failed to persist valid group: %v", err)
	}
	if err := db.Create(&degradedGroup).Error; err != nil {
		t.Fatalf("failed to persist degraded group: %v", err)
	}

	mcpService := newTestMCPService(t, db)

	svc, err := NewToolGroupService(db, mcpService)
	if err != nil {
		t.Fatalf("expected degraded persisted group not to fail startup, got: %v", err)
	}

	validProxy, ok := svc.GetToolGroupMCPServer("valid-group")
	if !ok {
		t.Fatal("expected valid group MCP proxy to be initialized")
	}
	validTools := validProxy.ListTools()
	if len(validTools) != 1 {
		t.Fatalf("expected valid group proxy to expose 1 tool, got %d", len(validTools))
	}
	if _, ok := validTools["valid-server__sum"]; !ok {
		t.Fatalf("expected valid group proxy to expose valid-server__sum, got keys %v", reflect.ValueOf(validTools).MapKeys())
	}

	degradedProxy, ok := svc.GetToolGroupMCPServer("degraded-group")
	if !ok {
		t.Fatal("expected degraded group MCP proxy to be initialized")
	}
	if len(degradedProxy.ListTools()) != 0 {
		t.Fatalf("expected degraded group proxy to expose 0 tools, got %d", len(degradedProxy.ListTools()))
	}

	degradedSSEProxy, ok := svc.GetToolGroupSseMCPServer("degraded-group")
	if !ok {
		t.Fatal("expected degraded group SSE MCP proxy to be initialized")
	}
	if len(degradedSSEProxy.ListTools()) != 0 {
		t.Fatalf("expected degraded group SSE proxy to expose 0 tools, got %d", len(degradedSSEProxy.ListTools()))
	}
}

func TestCreateToolGroup_InvalidIncludedServerStillFailsFast(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolGroupService{
		db:         db,
		mcpService: newTestMCPService(t, db),
	}

	err := s.CreateToolGroup(&model.ToolGroup{
		Name:            "invalid-server-group",
		IncludedServers: datatypes.JSON([]byte(`["missing-server"]`)),
	})
	if err == nil {
		t.Fatal("expected create tool group to fail for missing included server")
	}
	if !testhelpers.Contains(err.Error(), "failed to resolve effective tools") {
		t.Fatalf("expected create error to mention failed resolution, got: %v", err)
	}
}
