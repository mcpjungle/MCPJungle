package configsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/migrations"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "configsync-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := migrations.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	return db
}

func TestNew_DisabledSyncSkipsDependencyValidation(t *testing.T) {
	s, err := New(Options{Enabled: false}, nil, Services{})
	if err != nil {
		t.Fatalf("expected disabled config sync constructor to succeed, got: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNew_EnabledRequiresCoreDependencies(t *testing.T) {
	db := newTestDB(t)

	tests := []struct {
		name     string
		db       *gorm.DB
		services Services
	}{
		{
			name:     "missing db",
			db:       nil,
			services: Services{MCPService: &mcp.MCPService{}, ToolGroupService: &toolgroup.ToolGroupService{}},
		},
		{
			name:     "missing mcp service",
			db:       db,
			services: Services{ToolGroupService: &toolgroup.ToolGroupService{}},
		},
		{
			name:     "missing toolgroup service",
			db:       db,
			services: Services{MCPService: &mcp.MCPService{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Options{
				Enabled: true,
				Dir:     t.TempDir(),
			}, tt.db, tt.services)
			if err == nil {
				t.Fatal("expected constructor error")
			}
			if !strings.Contains(err.Error(), "requires DB, MCP service and ToolGroup service") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNew_EnabledRequiresDir(t *testing.T) {
	db := newTestDB(t)
	_, err := New(Options{
		Enabled: true,
		Dir:     "",
	}, db, Services{
		MCPService:       &mcp.MCPService{},
		ToolGroupService: &toolgroup.ToolGroupService{},
	})
	if err == nil {
		t.Fatal("expected constructor error")
	}
	if !strings.Contains(err.Error(), "requires a directory to watch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_EnterpriseSyncRequiresUserAndMcpClientServices(t *testing.T) {
	db := newTestDB(t)
	_, err := New(Options{
		Enabled:                    true,
		Dir:                        t.TempDir(),
		EnableEnterpriseEntitySync: true,
	}, db, Services{
		MCPService:       &mcp.MCPService{},
		ToolGroupService: &toolgroup.ToolGroupService{},
	})
	if err == nil {
		t.Fatal("expected constructor error")
	}
	if !strings.Contains(err.Error(), "requires User service and MCP Client service") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_EnterpriseSyncWithAllDependenciesSucceeds(t *testing.T) {
	db := newTestDB(t)
	svc, err := New(Options{
		Enabled:                    true,
		Dir:                        t.TempDir(),
		EnableEnterpriseEntitySync: true,
	}, db, Services{
		MCPService:       &mcp.MCPService{},
		ToolGroupService: &toolgroup.ToolGroupService{},
		UserService:      user.NewUserService(db),
		MCPClientService: mcpclient.NewMCPClientService(db),
	})
	if err != nil {
		t.Fatalf("expected constructor success, got: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestReconcileUsers_AdminUserIsRejected(t *testing.T) {
	db := newTestDB(t)
	userService := user.NewUserService(db)
	tmp := t.TempDir()
	s := &Service{db: db, services: Services{UserService: userService}, opts: Options{Dir: tmp, EnableEnterpriseEntitySync: true}}

	if err := s.ensureSubDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	if err := db.Create(&model.User{Username: "admin", Role: types.UserRoleAdmin, AccessToken: "mcpjungle_test_admin"}).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	cfg := `{"name":"admin","access_token":"mcpjungle_test_replacement"}`
	if err := os.WriteFile(filepath.Join(tmp, "users", "admin.json"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := s.reconcileUsers()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot manage admin user") {
		t.Fatalf("expected admin restriction error, got: %v", err)
	}

	var tracked []model.ManagedConfigFile
	if err := db.Where("entity_type = ?", model.EntityTypeUser).Find(&tracked).Error; err != nil {
		t.Fatalf("query tracked: %v", err)
	}
	if len(tracked) != 0 {
		t.Fatalf("expected no tracking rows for admin user, got %d", len(tracked))
	}
}

func TestReconcileUsers_AdoptsExistingManualUser(t *testing.T) {
	db := newTestDB(t)
	userService := user.NewUserService(db)
	tmp := t.TempDir()
	s := &Service{db: db, services: Services{UserService: userService}, opts: Options{Dir: tmp, EnableEnterpriseEntitySync: true}}

	if err := s.ensureSubDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	if err := db.Create(&model.User{Username: "alice", Role: types.UserRoleUser, AccessToken: "mcpjungle_test_oldtoken"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	cfg := `{"name":"alice","access_token":"mcpjungle_test_newtoken"}`
	if err := os.WriteFile(filepath.Join(tmp, "users", "alice.json"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := s.reconcileUsers(); err != nil {
		t.Fatalf("reconcile users: %v", err)
	}

	var updated model.User
	if err := db.Where("username = ?", "alice").First(&updated).Error; err != nil {
		t.Fatalf("fetch user: %v", err)
	}
	if updated.AccessToken != "mcpjungle_test_newtoken" {
		t.Fatalf("expected token update, got %s", updated.AccessToken)
	}

	var tracked model.ManagedConfigFile
	if err := db.Where("entity_type = ? AND entity_name = ?", model.EntityTypeUser, "alice").First(&tracked).Error; err != nil {
		t.Fatalf("fetch tracking row: %v", err)
	}
	if tracked.FilePath == "" || tracked.FileHash == "" {
		t.Fatalf("expected tracking metadata to be set")
	}
}

func TestReconcileUsers_SkipsInDevelopmentMode(t *testing.T) {
	db := newTestDB(t)
	userService := user.NewUserService(db)
	tmp := t.TempDir()
	s := &Service{db: db, services: Services{UserService: userService}, opts: Options{Dir: tmp, EnableEnterpriseEntitySync: false}}

	if err := s.ensureSubDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}

	if err := db.Create(&model.User{Username: "alice", Role: types.UserRoleUser, AccessToken: "mcpjungle_test_oldtoken"}).Error; err != nil {
		t.Fatalf("seed alice: %v", err)
	}
	if err := db.Create(&model.User{Username: "bob", Role: types.UserRoleUser, AccessToken: "mcpjungle_test_bob"}).Error; err != nil {
		t.Fatalf("seed bob: %v", err)
	}
	if err := db.Create(&model.ManagedConfigFile{
		EntityType: model.EntityTypeUser,
		EntityName: "bob",
		FilePath:   filepath.Join(tmp, "users", "bob.json"),
		FileHash:   "oldhash",
	}).Error; err != nil {
		t.Fatalf("seed managed row: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmp, "users", "alice.json"), []byte(`{"name":"alice","access_token":"mcpjungle_test_newtoken"}`), 0o644); err != nil {
		t.Fatalf("write alice file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "users", "carol.json"), []byte(`{"name":"carol","access_token":"mcpjungle_test_carol"}`), 0o644); err != nil {
		t.Fatalf("write carol file: %v", err)
	}

	if err := s.reconcileUsers(); err != nil {
		t.Fatalf("reconcile users in dev mode: %v", err)
	}

	var alice model.User
	if err := db.Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("fetch alice: %v", err)
	}
	if alice.AccessToken != "mcpjungle_test_oldtoken" {
		t.Fatalf("expected alice to remain unchanged in dev mode, got %s", alice.AccessToken)
	}

	var carol model.User
	if err := db.Where("username = ?", "carol").First(&carol).Error; err == nil {
		t.Fatal("expected carol to not be created in dev mode")
	}

	var bob model.User
	if err := db.Where("username = ?", "bob").First(&bob).Error; err != nil {
		t.Fatalf("expected bob to not be deleted in dev mode: %v", err)
	}

	var tracked model.ManagedConfigFile
	if err := db.Where("entity_type = ? AND entity_name = ?", model.EntityTypeUser, "bob").First(&tracked).Error; err != nil {
		t.Fatalf("expected managed tracking row for bob to remain: %v", err)
	}
}

func TestReconcileMcpClients_SkipsInDevelopmentMode(t *testing.T) {
	db := newTestDB(t)
	clientService := mcpclient.NewMCPClientService(db)
	tmp := t.TempDir()
	s := &Service{db: db, services: Services{MCPClientService: clientService}, opts: Options{Dir: tmp, EnableEnterpriseEntitySync: false}}

	if err := s.ensureSubDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}

	if _, err := clientService.CreateClient(model.McpClient{
		Name:        "alpha",
		AccessToken: "mcpjungle_test_alpha_old",
	}); err != nil {
		t.Fatalf("seed alpha client: %v", err)
	}
	if _, err := clientService.CreateClient(model.McpClient{
		Name:        "beta",
		AccessToken: "mcpjungle_test_beta",
	}); err != nil {
		t.Fatalf("seed beta client: %v", err)
	}
	if err := db.Create(&model.ManagedConfigFile{
		EntityType: model.EntityTypeMcpClient,
		EntityName: "beta",
		FilePath:   filepath.Join(tmp, types.ConfigSyncMcpClientsDirName, "beta.json"),
		FileHash:   "oldhash",
	}).Error; err != nil {
		t.Fatalf("seed managed client row: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmp, types.ConfigSyncMcpClientsDirName, "alpha.json"), []byte(`{"name":"alpha","access_token":"mcpjungle_test_alpha_new","allowed_servers":[]}`), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, types.ConfigSyncMcpClientsDirName, "gamma.json"), []byte(`{"name":"gamma","access_token":"mcpjungle_test_gamma","allowed_servers":[]}`), 0o644); err != nil {
		t.Fatalf("write gamma file: %v", err)
	}

	if err := s.reconcileMcpClients(); err != nil {
		t.Fatalf("reconcile mcp clients in dev mode: %v", err)
	}

	alpha, err := clientService.GetClient("alpha")
	if err != nil {
		t.Fatalf("fetch alpha: %v", err)
	}
	if alpha.AccessToken != "mcpjungle_test_alpha_old" {
		t.Fatalf("expected alpha to remain unchanged in dev mode, got %s", alpha.AccessToken)
	}

	if _, err := clientService.GetClient("gamma"); err == nil {
		t.Fatal("expected gamma to not be created in dev mode")
	}
	if _, err := clientService.GetClient("beta"); err != nil {
		t.Fatalf("expected beta to not be deleted in dev mode: %v", err)
	}
}
