// Package configsync provides configuration-as-code synchronization for MCPJungle.
package configsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

const (
	defaultConfigDirName = ".mcpjungle"

	subdirMcpServers = "mcp_servers"
	subdirMcpClients = "mcp_clients"
	subdirToolGroups = "tool_groups"
	subdirUsers      = "users"
)

const (
	entityTypeMcpServer = "mcp_server"
	entityTypeMcpClient = "mcp_client"
	entityTypeToolGroup = "tool_group"
	entityTypeUser      = "user"
)

// Manager synchronizes filesystem configuration with the database and in-memory state.
type Manager struct {
	db *gorm.DB

	mcpService       *mcp.MCPService
	mcpClientService *mcpclient.McpClientService
	toolGroupService *toolgroup.ToolGroupService
	userService      *user.UserService

	configDir string

	watcher *fsnotify.Watcher

	mu sync.Mutex
}

// DefaultConfigDir returns the default config directory path.
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}
	return filepath.Join(home, defaultConfigDirName), nil
}

// NewManager creates a new config sync manager.
func NewManager(
	db *gorm.DB,
	mcpService *mcp.MCPService,
	mcpClientService *mcpclient.McpClientService,
	toolGroupService *toolgroup.ToolGroupService,
	userService *user.UserService,
	configDir string,
) *Manager {
	return &Manager{
		db:               db,
		mcpService:       mcpService,
		mcpClientService: mcpClientService,
		toolGroupService: toolGroupService,
		userService:      userService,
		configDir:        configDir,
	}
}

// Start loads configuration files and starts watching for changes until the context is canceled.
func (m *Manager) Start(ctx context.Context) error {
	if err := m.ensureConfigDirs(); err != nil {
		return err
	}

	if err := m.syncAll(ctx); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create config watcher: %w", err)
	}
	m.watcher = watcher

	if err := m.addWatchDirs(); err != nil {
		_ = watcher.Close()
		return err
	}

	go m.watchLoop(ctx)
	return nil
}

// Close stops the filesystem watcher.
func (m *Manager) Close() {
	if m.watcher != nil {
		_ = m.watcher.Close()
	}
}

func (m *Manager) ensureConfigDirs() error {
	paths := []string{
		m.configDir,
		filepath.Join(m.configDir, subdirMcpServers),
		filepath.Join(m.configDir, subdirMcpClients),
		filepath.Join(m.configDir, subdirToolGroups),
		filepath.Join(m.configDir, subdirUsers),
	}

	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create config directory %s: %w", path, err)
		}
	}
	return nil
}

func (m *Manager) addWatchDirs() error {
	dirs := []string{
		m.configDir,
		filepath.Join(m.configDir, subdirMcpServers),
		filepath.Join(m.configDir, subdirMcpClients),
		filepath.Join(m.configDir, subdirToolGroups),
		filepath.Join(m.configDir, subdirUsers),
	}

	for _, dir := range dirs {
		if err := m.watcher.Add(dir); err != nil {
			return fmt.Errorf("failed to watch config directory %s: %w", dir, err)
		}
	}
	return nil
}

func (m *Manager) watchLoop(ctx context.Context) {
	const debounceWindow = 250 * time.Millisecond

	var debounceTimer *time.Timer
	resetDebounce := func() {
		if debounceTimer == nil {
			debounceTimer = time.NewTimer(debounceWindow)
			return
		}
		if !debounceTimer.Stop() {
			select {
			case <-debounceTimer.C:
			default:
			}
		}
		debounceTimer.Reset(debounceWindow)
	}

	triggerSync := func() {
		if err := m.syncAll(ctx); err != nil {
			log.Printf("[configsync] sync failed: %v", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			m.Close()
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					if err := m.watcher.Add(event.Name); err != nil {
						log.Printf("[configsync] failed to watch new directory %s: %v", event.Name, err)
					}
				}
			}

			resetDebounce()
		case <-func() <-chan time.Time {
			if debounceTimer == nil {
				return nil
			}
			return debounceTimer.C
		}():
			triggerSync()
			debounceTimer = nil
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[configsync] watcher error: %v", err)
		}
	}
}

func (m *Manager) syncAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingFiles := map[string]struct{}{}

	if err := m.syncMcpServers(ctx, existingFiles); err != nil {
		return err
	}
	if err := m.syncMcpClients(existingFiles); err != nil {
		return err
	}
	if err := m.syncToolGroups(existingFiles); err != nil {
		return err
	}
	if err := m.syncUsers(existingFiles); err != nil {
		return err
	}

	return m.pruneRemovedFiles(existingFiles)
}

func (m *Manager) syncMcpServers(ctx context.Context, existingFiles map[string]struct{}) error {
	dir := filepath.Join(m.configDir, subdirMcpServers)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read MCP server configs: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Clean(filepath.Join(dir, entry.Name()))
		existingFiles[path] = struct{}{}

		var input types.RegisterServerInput
		if err := readJSONFile(path, &input); err != nil {
			log.Printf("[configsync] failed to read MCP server config %s: %v", path, err)
			continue
		}
		if input.Name == "" {
			log.Printf("[configsync] MCP server config %s missing name", path)
			continue
		}

		if err := m.applyMcpServer(ctx, &input); err != nil {
			log.Printf("[configsync] failed to apply MCP server %s: %v", input.Name, err)
			continue
		}

		if err := m.upsertConfigFileRecord(path, entityTypeMcpServer, input.Name); err != nil {
			log.Printf("[configsync] failed to track MCP server config %s: %v", path, err)
		}
	}

	return nil
}

func (m *Manager) syncMcpClients(existingFiles map[string]struct{}) error {
	dir := filepath.Join(m.configDir, subdirMcpClients)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read MCP client configs: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Clean(filepath.Join(dir, entry.Name()))
		existingFiles[path] = struct{}{}

		var input types.McpClientConfig
		if err := readJSONFile(path, &input); err != nil {
			log.Printf("[configsync] failed to read MCP client config %s: %v", path, err)
			continue
		}
		if input.Name == "" {
			log.Printf("[configsync] MCP client config %s missing name", path)
			continue
		}

		accessToken, err := resolveAccessTokenFromConfig(input.AccessToken, input.AccessTokenRef)
		if err != nil {
			log.Printf("[configsync] failed to resolve MCP client access token for %s: %v", input.Name, err)
			continue
		}
		if accessToken == "" {
			log.Printf("[configsync] MCP client %s config must supply an access token", input.Name)
			continue
		}

		allowListJSON, err := json.Marshal(input.AllowMcpServers)
		if err != nil {
			log.Printf("[configsync] failed to encode allow list for MCP client %s: %v", input.Name, err)
			continue
		}

		client := model.McpClient{
			Name:        input.Name,
			Description: input.Description,
			AccessToken: accessToken,
			AllowList:   allowListJSON,
		}

		if _, err := m.mcpClientService.UpsertClientFromConfig(client); err != nil {
			log.Printf("[configsync] failed to apply MCP client %s: %v", input.Name, err)
			continue
		}

		if err := m.upsertConfigFileRecord(path, entityTypeMcpClient, input.Name); err != nil {
			log.Printf("[configsync] failed to track MCP client config %s: %v", path, err)
		}
	}

	return nil
}

func (m *Manager) syncToolGroups(existingFiles map[string]struct{}) error {
	dir := filepath.Join(m.configDir, subdirToolGroups)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read tool group configs: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Clean(filepath.Join(dir, entry.Name()))
		existingFiles[path] = struct{}{}

		var input types.ToolGroup
		if err := readJSONFile(path, &input); err != nil {
			log.Printf("[configsync] failed to read tool group config %s: %v", path, err)
			continue
		}
		if input.Name == "" {
			log.Printf("[configsync] tool group config %s missing name", path)
			continue
		}

		group, err := toolGroupFromConfig(&input)
		if err != nil {
			log.Printf("[configsync] failed to build tool group %s: %v", input.Name, err)
			continue
		}

		if err := m.applyToolGroup(group); err != nil {
			log.Printf("[configsync] failed to apply tool group %s: %v", input.Name, err)
			continue
		}

		if err := m.upsertConfigFileRecord(path, entityTypeToolGroup, input.Name); err != nil {
			log.Printf("[configsync] failed to track tool group config %s: %v", path, err)
		}
	}

	return nil
}

func (m *Manager) syncUsers(existingFiles map[string]struct{}) error {
	dir := filepath.Join(m.configDir, subdirUsers)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read user configs: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Clean(filepath.Join(dir, entry.Name()))
		existingFiles[path] = struct{}{}

		var input types.UserConfig
		if err := readJSONFile(path, &input); err != nil {
			log.Printf("[configsync] failed to read user config %s: %v", path, err)
			continue
		}
		if input.Username == "" {
			log.Printf("[configsync] user config %s missing name", path)
			continue
		}

		accessToken, err := resolveAccessTokenFromConfig(input.AccessToken, input.AccessTokenRef)
		if err != nil {
			log.Printf("[configsync] failed to resolve access token for user %s: %v", input.Username, err)
			continue
		}
		if accessToken == "" {
			log.Printf("[configsync] user %s config must supply an access token", input.Username)
			continue
		}

		userModel := model.User{
			Username:    input.Username,
			AccessToken: accessToken,
		}

		if _, err := m.userService.UpsertUserFromConfig(&userModel); err != nil {
			log.Printf("[configsync] failed to apply user %s: %v", input.Username, err)
			continue
		}

		if err := m.upsertConfigFileRecord(path, entityTypeUser, input.Username); err != nil {
			log.Printf("[configsync] failed to track user config %s: %v", path, err)
		}
	}

	return nil
}

func (m *Manager) applyMcpServer(ctx context.Context, input *types.RegisterServerInput) error {
	transport, err := types.ValidateTransport(input.Transport)
	if err != nil {
		return err
	}
	sessionMode, err := types.ValidateSessionMode(input.SessionMode)
	if err != nil {
		return err
	}

	var server *model.McpServer
	switch transport {
	case types.TransportStreamableHTTP:
		server, err = model.NewStreamableHTTPServer(
			input.Name,
			input.Description,
			input.URL,
			input.BearerToken,
			sessionMode,
		)
	case types.TransportStdio:
		server, err = model.NewStdioServer(
			input.Name,
			input.Description,
			input.Command,
			input.Args,
			input.Env,
			sessionMode,
		)
	default:
		server, err = model.NewSSEServer(
			input.Name,
			input.Description,
			input.URL,
			input.BearerToken,
			sessionMode,
		)
	}
	if err != nil {
		return err
	}

	existing, err := m.mcpService.GetMcpServer(input.Name)
	if err == nil {
		if mcpServersEqual(existing, server) {
			return nil
		}
		if err := m.mcpService.DeregisterMcpServer(input.Name); err != nil {
			return err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return m.mcpService.RegisterMcpServer(ctx, server)
}

func (m *Manager) applyToolGroup(group *model.ToolGroup) error {
	_, err := m.toolGroupService.GetToolGroup(group.Name)
	if err != nil {
		if errors.Is(err, toolgroup.ErrToolGroupNotFound) {
			return m.toolGroupService.CreateToolGroup(group)
		}
		return err
	}

	_, err = m.toolGroupService.UpdateToolGroup(group.Name, group)
	return err
}

func (m *Manager) pruneRemovedFiles(existingFiles map[string]struct{}) error {
	var records []model.ConfigFile
	if err := m.db.Find(&records).Error; err != nil {
		return fmt.Errorf("failed to list tracked config files: %w", err)
	}

	for _, record := range records {
		if _, ok := existingFiles[record.Path]; ok {
			continue
		}
		if err := m.deleteManagedEntity(&record); err != nil {
			log.Printf("[configsync] failed to delete managed entity %s (%s): %v", record.EntityName, record.EntityType, err)
			continue
		}
		if err := m.db.Unscoped().Delete(&model.ConfigFile{}, record.ID).Error; err != nil {
			log.Printf("[configsync] failed to remove config file record %s: %v", record.Path, err)
		}
	}

	return nil
}

func (m *Manager) deleteManagedEntity(record *model.ConfigFile) error {
	switch record.EntityType {
	case entityTypeMcpServer:
		return m.mcpService.DeregisterMcpServer(record.EntityName)
	case entityTypeMcpClient:
		return m.mcpClientService.DeleteClient(record.EntityName)
	case entityTypeToolGroup:
		return m.toolGroupService.DeleteToolGroup(record.EntityName)
	case entityTypeUser:
		return m.userService.DeleteUser(record.EntityName)
	default:
		return fmt.Errorf("unknown entity type %s", record.EntityType)
	}
}

func (m *Manager) upsertConfigFileRecord(path, entityType, entityName string) error {
	path = filepath.Clean(path)

	var record model.ConfigFile
	if err := m.db.Where("path = ?", path).First(&record).Error; err == nil {
		record.EntityType = entityType
		record.EntityName = entityName
		return m.db.Save(&record).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if err := m.db.Where("entity_type = ? AND entity_name = ?", entityType, entityName).First(&record).Error; err == nil {
		record.Path = path
		return m.db.Save(&record).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	record = model.ConfigFile{
		Path:       path,
		EntityType: entityType,
		EntityName: entityName,
	}
	return m.db.Create(&record).Error
}

func readJSONFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return fs.ErrInvalid
	}
	return json.Unmarshal(data, out)
}

func resolveAccessTokenFromConfig(accessToken string, accessTokenRef types.AccessTokenRef) (string, error) {
	if accessToken != "" {
		return accessToken, nil
	}

	if accessTokenRef.Env != "" {
		value, ok := os.LookupEnv(accessTokenRef.Env)
		if ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed, nil
			}
		}
		if accessTokenRef.File == "" {
			return "", fmt.Errorf("environment variable %s is not set or empty", accessTokenRef.Env)
		}
	}

	if accessTokenRef.File != "" {
		data, err := os.ReadFile(accessTokenRef.File)
		if err != nil {
			return "", fmt.Errorf("failed to read access token file %s: %w", accessTokenRef.File, err)
		}
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			return "", fmt.Errorf("access token file %s is empty", accessTokenRef.File)
		}
		return trimmed, nil
	}

	return "", nil
}

func toolGroupFromConfig(input *types.ToolGroup) (*model.ToolGroup, error) {
	includedTools, err := json.Marshal(input.IncludedTools)
	if err != nil {
		return nil, err
	}
	includedServers, err := json.Marshal(input.IncludedServers)
	if err != nil {
		return nil, err
	}
	excludedTools, err := json.Marshal(input.ExcludedTools)
	if err != nil {
		return nil, err
	}

	return &model.ToolGroup{
		Name:            input.Name,
		Description:     input.Description,
		IncludedTools:   includedTools,
		IncludedServers: includedServers,
		ExcludedTools:   excludedTools,
	}, nil
}

func mcpServersEqual(existing, desired *model.McpServer) bool {
	if existing.Name != desired.Name {
		return false
	}
	if existing.Transport != desired.Transport {
		return false
	}
	if existing.Description != desired.Description {
		return false
	}
	if existing.SessionMode != desired.SessionMode {
		return false
	}
	return string(existing.Config) == string(desired.Config)
}
