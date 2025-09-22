package mcpclient

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestNewMCPClientService(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)
	testhelpers.AssertNotNil(t, svc)
	if svc.db != db {
		t.Errorf("Expected db to be %v, got %v", db, svc.db)
	}
}

func TestListClientsEmpty(t *testing.T) {
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	svc := NewMCPClientService(setup.DB)

	clients, err := svc.ListClients()
	testhelpers.AssertNoError(t, err)
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients initially, got %d", len(clients))
	}
}

func TestCreateClient(t *testing.T) {
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	svc := NewMCPClientService(setup.DB)

	clientInput := model.McpClient{
		Name:        "test-client",
		Description: "Test MCP client",
	}

	client, err := svc.CreateClient(clientInput)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, client)

	// Verify client properties
	testhelpers.AssertEqual(t, "test-client", client.Name)
	testhelpers.AssertEqual(t, "Test MCP client", client.Description)
	if client.AccessToken == "" {
		t.Error("Expected access token to be generated")
	}

	// Verify client was saved to database
	var savedClient model.McpClient
	err = setup.DB.Where("name = ?", "test-client").First(&savedClient).Error
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, "test-client", savedClient.Name)
	testhelpers.AssertEqual(t, "Test MCP client", savedClient.Description)
	if savedClient.AccessToken == "" {
		t.Error("Expected saved client to have access token")
	}
}

func TestCreateClientWithExistingName(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	clientInput := model.McpClient{
		Name:        "test-client",
		Description: "Test MCP client",
	}

	// Create first client
	client1, err := svc.CreateClient(clientInput)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, client1)

	// Try to create another client with same name
	client2, err := svc.CreateClient(clientInput)
	testhelpers.AssertError(t, err)
	if client2 != nil {
		t.Error("Expected second client creation to fail")
	}
}

func TestGetClientByToken(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	// Create a test client
	clientInput := model.McpClient{
		Name:        "test-client",
		Description: "Test MCP client",
	}

	client, err := svc.CreateClient(clientInput)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, client)

	// Get client by token
	retrievedClient, err := svc.GetClientByToken(client.AccessToken)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, client.ID, retrievedClient.ID)
	testhelpers.AssertEqual(t, client.Name, retrievedClient.Name)
	testhelpers.AssertEqual(t, client.Description, retrievedClient.Description)
	testhelpers.AssertEqual(t, client.AccessToken, retrievedClient.AccessToken)
}

func TestGetClientByTokenNotFound(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	// Try to get client with non-existent token
	client, err := svc.GetClientByToken("non-existent-token")
	testhelpers.AssertError(t, err)
	if client != nil {
		t.Error("Expected client to be nil when token not found")
	}
}

func TestDeleteClient(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	// Create a test client
	clientInput := model.McpClient{
		Name:        "test-client",
		Description: "Test MCP client",
	}

	client, err := svc.CreateClient(clientInput)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, client)

	// Delete client
	err = svc.DeleteClient(client.Name)
	testhelpers.AssertNoError(t, err)

	// Verify client was deleted
	_, err = svc.GetClientByToken(client.AccessToken)
	testhelpers.AssertError(t, err)
}

func TestDeleteClientNotFound(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	// Try to delete non-existent client
	err = svc.DeleteClient("non-existent-client")
	testhelpers.AssertNoError(t, err) // DeleteClient is idempotent and doesn't error on non-existent clients
}

func TestListClientsWithData(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	// Create multiple test clients
	clientInputs := []model.McpClient{
		{Name: "client-1", Description: "First test client"},
		{Name: "client-2", Description: "Second test client"},
		{Name: "client-3", Description: "Third test client"},
	}

	for _, input := range clientInputs {
		_, err := svc.CreateClient(input)
		testhelpers.AssertNoError(t, err)
	}

	// List all clients
	clients, err := svc.ListClients()
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, 3, len(clients))

	// Verify all clients are present
	names := make(map[string]bool)
	for _, client := range clients {
		names[client.Name] = true
	}

	testhelpers.AssertTrue(t, names["client-1"], "Expected client-1 to be present")
	testhelpers.AssertTrue(t, names["client-2"], "Expected client-2 to be present")
	testhelpers.AssertTrue(t, names["client-3"], "Expected client-3 to be present")
}

func TestClientTokenUniqueness(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the McpClient model
	err = db.AutoMigrate(&model.McpClient{})
	testhelpers.AssertNoError(t, err)

	svc := NewMCPClientService(db)

	// Create multiple clients
	clientInputs := []model.McpClient{
		{Name: "client-1", Description: "First test client"},
		{Name: "client-2", Description: "Second test client"},
		{Name: "client-3", Description: "Third test client"},
	}

	tokens := make(map[string]bool)
	for _, input := range clientInputs {
		client, err := svc.CreateClient(input)
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertNotNil(t, client)

		// Verify token is unique
		if tokens[client.AccessToken] {
			t.Errorf("Duplicate token generated: %s", client.AccessToken)
		}
		tokens[client.AccessToken] = true
	}
}
