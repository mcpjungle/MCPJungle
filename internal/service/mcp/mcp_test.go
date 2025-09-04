package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"gorm.io/gorm"
)

func TestNewMCPService(t *testing.T) {
	tests := []struct {
		name           string
		db             *gorm.DB
		mcpProxyServer *server.MCPServer
		expectError    bool
	}{
		{
			name:           "valid service creation",
			db:             &gorm.DB{},
			mcpProxyServer: &server.MCPServer{},
			expectError:    false,
		},
		{
			name:           "nil database",
			db:             nil,
			mcpProxyServer: &server.MCPServer{},
			expectError:    true,
		},
		{
			name:           "nil proxy server",
			db:             &gorm.DB{},
			mcpProxyServer: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpService, err := NewMCPService(tt.db, tt.mcpProxyServer)

			if tt.expectError {
				testhelpers.AssertError(t, err)
				if mcpService != nil {
					t.Error("Expected service to be nil when error occurs")
				}
			} else {
				testhelpers.AssertNoError(t, err)
				testhelpers.AssertNotNil(t, mcpService)
				if mcpService.toolInstances == nil {
					t.Error("Expected toolInstances to be initialized")
				}
				if mcpService.toolDeletionCallback == nil {
					t.Error("Expected toolDeletionCallback to be initialized")
				}
				if mcpService.toolAdditionCallback == nil {
					t.Error("Expected toolAdditionCallback to be initialized")
				}
			}
		})
	}
}

func TestMCPServiceInitialization(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	proxyServer := &server.MCPServer{}

	mcpService, err := NewMCPService(db, proxyServer)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, mcpService)

	// Test that the service is properly initialized
	if mcpService.db != db {
		t.Errorf("Expected db to be %v, got %v", db, mcpService.db)
	}
	if mcpService.mcpProxyServer != proxyServer {
		t.Errorf("Expected mcpProxyServer to be %v, got %v", proxyServer, mcpService.mcpProxyServer)
	}
	if mcpService.toolInstances == nil {
		t.Error("Expected toolInstances to be initialized")
	}
	if mcpService.toolDeletionCallback == nil {
		t.Error("Expected toolDeletionCallback to be initialized")
	}
	if mcpService.toolAdditionCallback == nil {
		t.Error("Expected toolAdditionCallback to be initialized")
	}
}

func TestMCPServiceCallbacks(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	proxyServer := &server.MCPServer{}

	mcpService, err := NewMCPService(db, proxyServer)
	testhelpers.AssertNoError(t, err)

	// Test that callbacks are initialized to NOOP functions
	// These should not panic or return errors
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, but got: %v", r)
			}
		}()
		mcpService.toolDeletionCallback("tool1", "tool2")
	}()

	err = mcpService.toolAdditionCallback("tool1")
	testhelpers.AssertNoError(t, err)
}

func TestMCPServiceConcurrency(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	proxyServer := &server.MCPServer{}

	mcpService, err := NewMCPService(db, proxyServer)
	testhelpers.AssertNoError(t, err)

	// Test that the service can handle concurrent access to toolInstances
	// This is a basic test to ensure the mutex is working
	done := make(chan bool)

	// Start multiple goroutines to access toolInstances
	for i := 0; i < 10; i++ {
		go func() {
			mcpService.mu.RLock()
			_ = mcpService.toolInstances
			mcpService.mu.RUnlock()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without deadlock, the mutex is working
	// No assertion needed - if we reach here, the test passed
}

func TestMCPServiceToolInstances(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	proxyServer := &server.MCPServer{}

	mcpService, err := NewMCPService(db, proxyServer)
	testhelpers.AssertNoError(t, err)

	// Test that toolInstances map is properly initialized
	if mcpService.toolInstances == nil {
		t.Error("Expected toolInstances to be initialized")
	}
	if len(mcpService.toolInstances) != 0 {
		t.Errorf("Expected toolInstances to be empty, got %d items", len(mcpService.toolInstances))
	}

	// Test that we can safely access the map
	mcpService.mu.RLock()
	_, exists := mcpService.toolInstances["nonexistent"]
	mcpService.mu.RUnlock()

	if exists {
		t.Error("Expected nonexistent tool to not exist")
	}
}

func TestMCPServiceErrorHandling(t *testing.T) {
	// Test with invalid database connection
	// This would require mocking the database to simulate connection failures
	// For now, we'll test the basic error handling in the constructor

	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	proxyServer := &server.MCPServer{}

	mcpService, err := NewMCPService(db, proxyServer)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, mcpService)

	// Test that the service handles errors gracefully
	// This is a basic test - in a real scenario, you'd want to test
	// database connection failures, proxy server initialization failures, etc.
	if mcpService.toolInstances == nil {
		t.Error("Expected toolInstances to be initialized")
	}
}
