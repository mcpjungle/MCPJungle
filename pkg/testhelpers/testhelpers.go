// Package testhelpers provides common testing utilities and assertion functions
// for the MCPJungle project.
package testhelpers

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// CreateTestDB creates a test database using SQLite in-memory database
func CreateTestDB() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
}

// AssertError asserts that an error occurred
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// AssertNoError asserts that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// AssertNotNil asserts that an object is not nil
func AssertNotNil(t *testing.T, obj any) {
	t.Helper()
	if obj == nil {
		t.Error("Expected not nil, got nil")
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Error(message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Error(message)
	}
}

// AssertStringContains asserts that a string contains a substring
func AssertStringContains(t *testing.T, str, substr string) {
	t.Helper()
	if !Contains(str, substr) {
		t.Errorf("Expected string '%s' to contain '%s'", str, substr)
	}
}

// AssertStringNotContains asserts that a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if Contains(str, substr) {
		t.Errorf("Expected string '%s' to not contain '%s'", str, substr)
	}
}

// AssertSliceLength asserts that a slice has the expected length
func AssertSliceLength(t *testing.T, slice any, expectedLength int) {
	t.Helper()
	switch v := slice.(type) {
	case []any:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	case []string:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	case []int:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	default:
		t.Error("Unsupported slice type for length assertion")
	}
}

// AssertMapContainsKey asserts that a map contains a specific key
func AssertMapContainsKey(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, exists := m[key]; !exists {
		t.Errorf("Expected map to contain key '%s'", key)
	}
}

// AssertMapNotContainsKey asserts that a map does not contain a specific key
func AssertMapNotContainsKey(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, exists := m[key]; exists {
		t.Errorf("Expected map to not contain key '%s'", key)
	}
}

// AssertPanic asserts that a function panics
func AssertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected function to panic")
		}
	}()
	fn()
}

// AssertNoPanic asserts that a function does not panic
func AssertNoPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected no panic, but got: %v", r)
		}
	}()
	fn()
}

// CreateTestTable creates a test table for table-driven tests
func CreateTestTable[T any](tests []T) []T {
	return tests
}

// RunTableTests runs table-driven tests
func RunTableTests[T any](t *testing.T, tests []T, testFunc func(t *testing.T, test T)) {
	for i, tt := range tests {
		t.Run(testName(i, tt), func(t *testing.T) {
			testFunc(t, tt)
		})
	}
}

// Helper function to generate test names
func testName(index int, test any) string {
	if named, ok := test.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fmt.Sprintf("test_%d", index)
}

// Contains checks if a string contains a substring
// This is a public function that can be used by other packages
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// containsSubstring is a helper function for Contains
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FormatError formats error messages consistently
func FormatError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// FormatSliceError formats slice error messages
func FormatSliceError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// FormatMapError formats map error messages
func FormatMapError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// TestDBSetup represents a test database setup with common models
type TestDBSetup struct {
	DB *gorm.DB
}

// SetupTestDB creates a test database with all common models migrated
func SetupTestDB(t *testing.T) *TestDBSetup {
	t.Helper()

	db, err := CreateTestDB()
	AssertNoError(t, err)

	// Migrate all common models
	err = db.AutoMigrate(
		&model.User{},
		&model.McpClient{},
		&model.McpServer{},
		&model.Tool{},
		&model.ServerConfig{},
		&model.ToolGroup{},
	)
	AssertNoError(t, err)

	return &TestDBSetup{DB: db}
}

// SetupUserTest creates a test database with user-related models and a basic test user
func SetupUserTest(t *testing.T) (*TestDBSetup, *model.User) {
	t.Helper()

	setup := SetupTestDB(t)

	// Create a basic test user
	testUser := &model.User{
		Username:    "testuser",
		Role:        types.UserRoleUser,
		AccessToken: "test-access-token-123",
	}

	err := setup.DB.Create(testUser).Error
	AssertNoError(t, err)

	return setup, testUser
}

// SetupAdminTest creates a test database with user-related models and a basic admin user
func SetupAdminTest(t *testing.T) (*TestDBSetup, *model.User) {
	t.Helper()

	setup := SetupTestDB(t)

	// Create a basic test admin user
	testAdmin := &model.User{
		Username:    "testadmin",
		Role:        types.UserRoleAdmin,
		AccessToken: "test-admin-token-456",
	}

	err := setup.DB.Create(testAdmin).Error
	AssertNoError(t, err)

	return setup, testAdmin
}

// SetupMCPTest creates a test database with MCP-related models
func SetupMCPTest(t *testing.T) *TestDBSetup {
	t.Helper()

	setup := SetupTestDB(t)

	// Additional MCP-specific setup can be added here
	// For example, creating test MCP servers, tools, etc.

	return setup
}

// SetupClientTest creates a test database with MCP client models and a basic test client
func SetupClientTest(t *testing.T) (*TestDBSetup, *model.McpClient) {
	t.Helper()

	setup := SetupTestDB(t)

	// Create a basic test MCP client
	testClient := &model.McpClient{
		Name:        "test-client",
		Description: "Test MCP client for unit tests",
		AccessToken: "test-client-token-789",
		AllowList:   []byte("[]"), // Empty allow list
	}

	err := setup.DB.Create(testClient).Error
	AssertNoError(t, err)

	return setup, testClient
}

// SetupServerConfigTest creates a test database with server config models
func SetupServerConfigTest(t *testing.T) *TestDBSetup {
	t.Helper()

	setup := SetupTestDB(t)

	// Additional server config setup can be added here

	return setup
}

// CreateTestUser creates a test user with the given parameters
func (s *TestDBSetup) CreateTestUser(username string, role types.UserRole, accessToken string) *model.User {
	user := &model.User{
		Username:    username,
		Role:        role,
		AccessToken: accessToken,
	}

	err := s.DB.Create(user).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test user: %v", err))
	}

	return user
}

// CreateTestMcpClient creates a test MCP client with the given parameters
func (s *TestDBSetup) CreateTestMcpClient(name, description, accessToken string, allowList []string) *model.McpClient {
	allowListJSON := []byte("[]")
	if len(allowList) > 0 {
		// Create a proper JSON array
		jsonStr := "["
		for i, item := range allowList {
			if i > 0 {
				jsonStr += ","
			}
			jsonStr += fmt.Sprintf(`"%s"`, item)
		}
		jsonStr += "]"
		allowListJSON = []byte(jsonStr)
	}

	client := &model.McpClient{
		Name:        name,
		Description: description,
		AccessToken: accessToken,
		AllowList:   allowListJSON,
	}

	err := s.DB.Create(client).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test MCP client: %v", err))
	}

	return client
}

// CreateTestMcpServer creates a test MCP server with the given parameters
func (s *TestDBSetup) CreateTestMcpServer(name, description string, transport types.McpServerTransport, config []byte) *model.McpServer {
	server := &model.McpServer{
		Name:        name,
		Description: description,
		Transport:   transport,
		Config:      config,
	}

	err := s.DB.Create(server).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test MCP server: %v", err))
	}

	return server
}

// CreateTestTool creates a test tool with the given parameters
func (s *TestDBSetup) CreateTestTool(name, description string, serverID uint, enabled bool, inputSchema []byte) *model.Tool {
	tool := &model.Tool{
		Name:        name,
		Description: description,
		ServerID:    serverID,
		Enabled:     enabled,
		InputSchema: inputSchema,
	}

	err := s.DB.Create(tool).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test tool: %v", err))
	}

	return tool
}

// CreateTestServerConfig creates a test server config with the given parameters
func (s *TestDBSetup) CreateTestServerConfig(mode model.ServerMode, initialized bool) *model.ServerConfig {
	config := &model.ServerConfig{
		Mode:        mode,
		Initialized: initialized,
	}

	err := s.DB.Create(config).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test server config: %v", err))
	}

	return config
}

// Cleanup closes the database connection
func (s *TestDBSetup) Cleanup() {
	if s.DB != nil {
		if sqlDB, err := s.DB.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

// CommandAnnotationTest represents test data for command annotation testing
type CommandAnnotationTest struct {
	Key      string
	Expected string
}

// TestCommandAnnotations tests command annotations using table-driven approach
func TestCommandAnnotations(t *testing.T, annotations map[string]string, tests []CommandAnnotationTest) {
	t.Helper()

	AssertNotNil(t, annotations)

	for _, tt := range tests {
		t.Run(tt.Key, func(t *testing.T) {
			value, exists := annotations[tt.Key]
			AssertTrue(t, exists, "Missing '"+tt.Key+"' annotation")
			AssertEqual(t, tt.Expected, value)
		})
	}
}

// TestCommandProperties tests basic command properties
func TestCommandProperties(t *testing.T, actualUse, expectedUse, actualShort, expectedShort string) {
	t.Helper()

	AssertEqual(t, expectedUse, actualUse)
	AssertEqual(t, expectedShort, actualShort)
}

// SubcommandTestData represents test data for subcommand testing
type SubcommandTestData struct {
	Use   string
	Short string
	Long  string
}

// TestSubcommandStructure tests a subcommand's basic structure
func TestSubcommandStructure(t *testing.T, actualUse, expectedUse, actualShort, expectedShort, actualLong string) {
	t.Helper()

	AssertEqual(t, expectedUse, actualUse)
	AssertEqual(t, expectedShort, actualShort)
	if actualLong != "" {
		AssertTrue(t, len(actualLong) > 0, "Long description should not be empty")
	}
}

// FlagTestData represents test data for flag testing
type FlagTestData struct {
	Name        string
	Description string
}
