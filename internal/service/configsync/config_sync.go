package configsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/accesstoken"
	"github.com/mcpjungle/mcpjungle/pkg/configfiles"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Options configures config directory synchronization.
type Options struct {
	// Enabled determines whether config syncing is active. If false, the service will be a no-op.
	Enabled bool

	// Dir is the directory to watch for config files.
	Dir string

	// EnableEnterpriseEntitySync should be set to true to allow syncing of enterprise entities like mcp clients and users.
	// If false, those entities will be ignored by the sync process.
	EnableEnterpriseEntitySync bool
}

// Services groups the external services that the config sync service depends on to do its job.
type Services struct {
	MCPService       *mcp.MCPService
	ToolGroupService *toolgroup.ToolGroupService
	UserService      *user.UserService
	MCPClientService *mcpclient.McpClientService
}

type Service struct {
	opts     Options
	db       *gorm.DB
	services Services

	watcher *fsnotify.Watcher
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func New(opts Options, db *gorm.DB, services Services) (*Service, error) {
	cs := &Service{opts: opts, db: db, services: services}

	if !opts.Enabled {
		// if syncing is not enabled, we can skip all the validation of dependencies and just return the service
		// as-is, since it will be a no-op
		return cs, nil
	}

	if db == nil || services.MCPService == nil || services.ToolGroupService == nil {
		return nil, fmt.Errorf("config sync requires DB, MCP service and ToolGroup service")
	}
	if opts.EnableEnterpriseEntitySync && (services.UserService == nil || services.MCPClientService == nil) {
		return nil, fmt.Errorf("config sync with enterprise entity syncing enabled requires User service and MCP Client service")
	}
	if opts.Dir == "" {
		return nil, fmt.Errorf("config sync requires a directory to watch")
	}
	return cs, nil
}

// Start begins watching the config directory for changes in the background and reconciling them with mcpjungle.
// It also performs an initial reconciliation on startup.
// If config sync is not enabled, this method is a no-op.
func (s *Service) Start(ctx context.Context) error {
	if !s.opts.Enabled {
		return nil
	}
	log.Printf("[config-sync] starting configuration syncing")
	if err := s.ensureSubDirs(); err != nil {
		return err
	}
	if err := s.Reconcile(ctx); err != nil {
		return fmt.Errorf("initial config reconciliation failed: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	s.watcher = watcher
	for _, d := range []string{
		s.opts.Dir,
		filepath.Join(s.opts.Dir, types.ConfigSyncMcpServersDirName),
		filepath.Join(s.opts.Dir, types.ConfigSyncMcpClientsDirName),
		filepath.Join(s.opts.Dir, types.ConfigSyncGroupsDirName),
		filepath.Join(s.opts.Dir, types.ConfigSyncGroupsDirName),
	} {
		if err := watcher.Add(d); err != nil {
			_ = watcher.Close()
			return fmt.Errorf("failed to watch %s: %w", d, err)
		}
	}

	watchCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.watchLoop(watchCtx)
	return nil
}

func (s *Service) Stop() {
	// if config wasn't enabled to begin with, there's nothing to stop
	if !s.opts.Enabled {
		return
	}
	log.Printf("[config-sync] stopping configuration syncing")
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	if s.watcher != nil {
		_ = s.watcher.Close()
	}
}

// watchLoop listens for file system events from the watcher.
// It is intended to be run in the background.
func (s *Service) watchLoop(ctx context.Context) {
	defer s.wg.Done()
	debounce := time.NewTimer(time.Hour)
	if !debounce.Stop() {
		<-debounce.C
	}
	pending := false
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[config-sync] watcher error: %v", err)
		case ev, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if !isRelevantFsEvent(ev) {
				continue
			}
			pending = true
			debounce.Reset(300 * time.Millisecond)
		case <-debounce.C:
			if pending {
				if err := s.Reconcile(context.Background()); err != nil {
					log.Printf("[config-sync] reconcile failed after file changes: %v", err)
				}
				pending = false
			}
		}
	}
}

// isRelevantFsEvent filters fsnotify events to only those that are relevant for triggering a reconciliation.
// We only care when a file is created, modified, removed, or renamed.
func isRelevantFsEvent(ev fsnotify.Event) bool {
	return ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0
}

// ensureSubDirs creates the expected subdirectories if they don't already exist.
// This ensures that the directories exist before syncing process begins.
func (s *Service) ensureSubDirs() error {
	subDirs := []string{
		types.ConfigSyncMcpServersDirName,
		types.ConfigSyncMcpClientsDirName,
		types.ConfigSyncGroupsDirName,
		types.ConfigSyncUsersDirName,
	}
	for _, sub := range subDirs {
		if err := os.MkdirAll(filepath.Join(s.opts.Dir, sub), 0o755); err != nil {
			return fmt.Errorf("failed to create config subdirectory %s: %w", sub, err)
		}
	}
	return nil
}

func (s *Service) Reconcile(ctx context.Context) error {
	var errs []error
	if err := s.reconcileMcpServers(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.reconcileMcpClients(); err != nil {
		errs = append(errs, err)
	}
	if err := s.reconcileGroups(); err != nil {
		errs = append(errs, err)
	}
	if err := s.reconcileUsers(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		for _, err := range errs {
			log.Printf("[config-sync] %v", err)
		}
		return errors.Join(errs...)
	}
	return nil
}

func (s *Service) loadManaged(entityType model.EntityType) (map[string]model.ManagedConfigFile, error) {
	var rows []model.ManagedConfigFile
	if err := s.db.Where("entity_type = ?", entityType).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]model.ManagedConfigFile, len(rows))
	for _, r := range rows {
		out[r.EntityName] = r
	}
	return out, nil
}

func toJSON(v any) (datatypes.JSON, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func logDesiredConfigChange(entityType, action, name, path string) {
	if strings.TrimSpace(path) == "" {
		log.Printf("[config-sync] desired config change: %s %s %q", action, entityType, name)
		return
	}
	log.Printf("[config-sync] desired config change: %s %s %q (source: %s)", action, entityType, name, path)
}

func (s *Service) createOrUpdateManagedRow(entityType model.EntityType, name, path, hash string) error {
	var row model.ManagedConfigFile
	err := s.db.Where("entity_type = ? AND entity_name = ?", entityType, name).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.Create(&model.ManagedConfigFile{
			EntityType: entityType,
			EntityName: name,
			FilePath:   path,
			FileHash:   hash,
		}).Error
	}
	row.FilePath = path
	row.FileHash = hash
	return s.db.Save(&row).Error
}

func (s *Service) deleteManagedRow(entityType model.EntityType, name string) error {
	return s.db.Unscoped().Where("entity_type = ? AND entity_name = ?", entityType, name).Delete(&model.ManagedConfigFile{}).Error
}

func (s *Service) reconcileMcpServers(ctx context.Context) error {
	mcpServersDir := filepath.Join(s.opts.Dir, types.ConfigSyncMcpServersDirName)
	desired, blocked, parseErrs := configfiles.LoadDesired[types.RegisterServerInput](
		mcpServersDir,
		func(i types.RegisterServerInput) string { return i.Name },
	)

	managed, err := s.loadManaged(model.EntityTypeMcpServer)
	if err != nil {
		return fmt.Errorf("failed to load tracked mcp server configs: %w", err)
	}

	var errs []error
	errs = append(errs, parseErrs...)

	// for each desired mcp server config, ensure the corresponding server exists with the correct attributes
	// also track the desired state config if not already tracked
	for name, d := range desired {
		transport, err := types.ValidateTransport(d.Entity.Transport)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid transport in %s: %w", d.Path, err))
			continue
		}
		sessionMode, err := types.ValidateSessionMode(d.Entity.SessionMode)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid session mode in %s: %w", d.Path, err))
			continue
		}
		server, err := newServerFromInput(d.Entity, transport, sessionMode)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid mcp server config in %s: %w", d.Path, err))
			continue
		}

		existing, getErr := s.services.MCPService.GetMcpServer(name)
		if getErr != nil && !errors.Is(getErr, mcp.ErrMCPServerNotFound) {
			errs = append(errs, fmt.Errorf("failed to read mcp server %s: %w", name, getErr))
			continue
		}

		track, tracked := managed[name]
		if errors.Is(getErr, mcp.ErrMCPServerNotFound) {
			// server is desired but doesn't yet exist, create it and start tracking it
			logDesiredConfigChange("mcp server", "create", name, d.Path)
			if err := s.services.MCPService.RegisterMcpServer(ctx, server); err != nil {
				errs = append(errs, fmt.Errorf("failed to create mcp server %s from %s: %w", name, d.Path, err))
				continue
			}
			if err := s.createOrUpdateManagedRow(model.EntityTypeMcpServer, name, d.Path, d.Hash); err != nil {
				errs = append(errs, fmt.Errorf("failed to track mcp server %s: %w", name, err))
			}
			continue
		}

		if tracked && track.FileHash == d.Hash {
			// server is already tracked and the config hasn't changed since the last time we applied it,
			// so we can skip any updates for this server
			continue
		}

		if !serverEqual(existing, server) {
			// server exists but is different from the desired state, update it.
			logDesiredConfigChange("mcp server", "update", name, d.Path)
			if err := s.services.MCPService.DeregisterMcpServer(name); err != nil {
				errs = append(errs, fmt.Errorf("failed to deregister mcp server %s for update: %w", name, err))
				continue
			}
			if err := s.services.MCPService.RegisterMcpServer(ctx, server); err != nil {
				errs = append(errs, fmt.Errorf("failed to register updated mcp server %s from %s: %w", name, d.Path, err))
				continue
			}
		}
		if err := s.createOrUpdateManagedRow(model.EntityTypeMcpServer, name, d.Path, d.Hash); err != nil {
			errs = append(errs, fmt.Errorf("failed to track mcp server %s: %w", name, err))
		}
	}

	// for each tracked mcp server config, if the corresponding config file is now missing and there are no errors
	// blocking deletion, delete the server and stop tracking its config
	for name, trackedRow := range managed {
		if _, ok := desired[name]; ok {
			continue
		}
		if blocked[trackedRow.FilePath] || fileExists(trackedRow.FilePath) {
			continue
		}
		logDesiredConfigChange("mcp server", "delete", name, trackedRow.FilePath)
		if err := s.services.MCPService.DeregisterMcpServer(name); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete managed mcp server %s after file removal: %w", name, err))
			continue
		}
		if err := s.deleteManagedRow(model.EntityTypeMcpServer, name); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove tracking for mcp server %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// newServerFromInput creates a model.McpServer from the given RegisterServerInput, transport, and session mode.
func newServerFromInput(input types.RegisterServerInput, transport types.McpServerTransport, sessionMode types.SessionMode) (*model.McpServer, error) {
	switch transport {
	case types.TransportStreamableHTTP:
		return model.NewStreamableHTTPServer(input.Name, input.Description, input.URL, input.BearerToken, input.Headers, sessionMode)
	case types.TransportStdio:
		return model.NewStdioServer(input.Name, input.Description, input.Command, input.Args, input.Env, sessionMode)
	default:
		return model.NewSSEServer(input.Name, input.Description, input.URL, input.BearerToken, sessionMode)
	}
}

func serverEqual(a, b *model.McpServer) bool {
	return a.Name == b.Name && a.Description == b.Description && a.Transport == b.Transport && a.SessionMode == b.SessionMode && slices.Equal(a.Config, b.Config)
}

func (s *Service) reconcileMcpClients() error {
	if !s.opts.EnableEnterpriseEntitySync {
		return nil
	}

	desired, blocked, parseErrs := configfiles.LoadDesired[types.McpClientConfig](filepath.Join(s.opts.Dir, types.ConfigSyncMcpClientsDirName), func(i types.McpClientConfig) string { return i.Name })

	managed, err := s.loadManaged(model.EntityTypeMcpClient)
	if err != nil {
		return err
	}

	var errs []error
	errs = append(errs, parseErrs...)

	// for each desired mcp client config, ensure the corresponding client exists with the correct attributes
	// also track the desired state config if not already tracked
	for name, d := range desired {
		allowJSON, err := toJSON(d.Entity.AllowMcpServers)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid allow list in %s: %w", d.Path, err))
			continue
		}

		accessToken, err := accesstoken.Resolve(accesstoken.Input{
			Inline: d.Entity.AccessToken,
			Ref:    d.Entity.AccessTokenRef,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to resolve access token in %s: %w", d.Path, err))
			continue
		}
		if accessToken == "" {
			errs = append(errs, fmt.Errorf("mcp client config %s must provide access token or access_token_ref", d.Path))
			continue
		}

		existing, err := s.services.MCPClientService.GetClient(name)
		if err != nil && !errors.Is(err, mcpclient.ErrMCPClientNotFound) {
			errs = append(errs, fmt.Errorf("failed to fetch mcp client %s: %w", name, err))
			continue
		}
		if errors.Is(err, mcpclient.ErrMCPClientNotFound) {
			// client is desired but doesn't yet exist, create it
			logDesiredConfigChange("mcp client", "create", name, d.Path)
			newClient := model.McpClient{
				Name:        name,
				Description: d.Entity.Description,
				AccessToken: accessToken,
				AllowList:   allowJSON,
			}
			_, createClientErr := s.services.MCPClientService.CreateClient(newClient)
			if createClientErr != nil {
				errs = append(errs, fmt.Errorf("failed to create mcp client %s: %w", name, err))
				continue
			}
		} else {
			// by this point, the client definitely exists. check for updates to any attributes.
			if existing.Description != d.Entity.Description || existing.AccessToken != accessToken || !slices.Equal(existing.AllowList, allowJSON) {
				logDesiredConfigChange("mcp client", "update", name, d.Path)
				existing.Description = d.Entity.Description
				existing.AccessToken = accessToken
				existing.AllowList = allowJSON

				_, err := s.services.MCPClientService.UpdateClient(*existing)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to update mcp client %s: %w", name, err))
					continue
				}
			}
		}

		if err := s.createOrUpdateManagedRow(model.EntityTypeMcpClient, name, d.Path, d.Hash); err != nil {
			errs = append(errs, fmt.Errorf("failed to track mcp client %s: %w", name, err))
		}
	}

	// for each tracked mcp client config, if the corresponding config file is now missing and there are no errors
	// blocking deletion, delete the client and stop tracking its config
	for name, trackedRow := range managed {
		if _, ok := desired[name]; ok {
			continue
		}
		if blocked[trackedRow.FilePath] || fileExists(trackedRow.FilePath) {
			continue
		}

		// client is no longer desired, delete it and stop tracking
		logDesiredConfigChange("mcp client", "delete", name, trackedRow.FilePath)
		if err := s.services.MCPClientService.DeleteClient(name); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete managed mcp client %s after file removal: %w", name, err))
			continue
		}
		if err := s.deleteManagedRow(model.EntityTypeMcpClient, name); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove tracking for mcp client %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *Service) reconcileGroups() error {
	desired, blocked, parseErrs := configfiles.LoadDesired[types.ToolGroup](filepath.Join(s.opts.Dir, types.ConfigSyncGroupsDirName), func(i types.ToolGroup) string { return i.Name })

	managed, err := s.loadManaged(model.EntityTypeGroup)
	if err != nil {
		return err
	}

	var errs []error
	errs = append(errs, parseErrs...)

	// for each desired group config, ensure the corresponding group exists with the correct attributes
	// also track the desired state config if not already tracked
	for name, d := range desired {
		incTools, err := toJSON(d.Entity.IncludedTools)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid included_tools in %s: %w", d.Path, err))
			continue
		}

		incServers, err := toJSON(d.Entity.IncludedServers)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid included_servers in %s: %w", d.Path, err))
			continue
		}

		exclTools, err := toJSON(d.Entity.ExcludedTools)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid excluded_tools in %s: %w", d.Path, err))
			continue
		}

		groupModel := &model.ToolGroup{
			Name:            name,
			Description:     d.Entity.Description,
			IncludedTools:   incTools,
			IncludedServers: incServers,
			ExcludedTools:   exclTools,
		}

		_, tracked := managed[name]

		old, err := s.services.ToolGroupService.GetToolGroup(name)
		if err != nil {
			if errors.Is(err, toolgroup.ErrToolGroupNotFound) {
				// group is desired but doesn't yet exist, create it
				logDesiredConfigChange("group", "create", name, d.Path)
				if err := s.services.ToolGroupService.CreateToolGroup(groupModel); err != nil {
					errs = append(errs, fmt.Errorf("failed to create group %s from %s: %w", name, d.Path, err))
					continue
				}
			} else {
				errs = append(errs, fmt.Errorf("failed to fetch group %s: %w", name, err))
				continue
			}
		} else {
			if !tracked || !groupEqual(old, groupModel) {
				// group exists but is either not tracked (manually created) or has drifted from the desired state,
				// update it
				logDesiredConfigChange("group", "update", name, d.Path)
				if _, err := s.services.ToolGroupService.UpdateToolGroup(name, groupModel); err != nil {
					errs = append(errs, fmt.Errorf("failed to update group %s from %s: %w", name, d.Path, err))
					continue
				}
			}
		}

		// by this point, the group definitely exists and is up to date. if it's not tracked already, track it now.
		if err := s.createOrUpdateManagedRow(model.EntityTypeGroup, name, d.Path, d.Hash); err != nil {
			errs = append(errs, fmt.Errorf("failed to track group %s: %w", name, err))
		}
	}

	// for each tracked group config, if the corresponding config file is now missing and there are no errors
	// blocking deletion, delete the group and stop tracking its config
	for name, trackedRow := range managed {
		if _, ok := desired[name]; ok {
			continue
		}
		if blocked[trackedRow.FilePath] || fileExists(trackedRow.FilePath) {
			continue
		}

		logDesiredConfigChange("group", "delete", name, trackedRow.FilePath)
		if err := s.services.ToolGroupService.DeleteToolGroup(name); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete managed group %s after file removal: %w", name, err))
			continue
		}
		if err := s.deleteManagedRow(model.EntityTypeGroup, name); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove tracking for group %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// groupEqual compares two ToolGroup models for equality based on their relevant fields.
func groupEqual(a, b *model.ToolGroup) bool {
	return a.Name == b.Name && a.Description == b.Description && slices.Equal(a.IncludedTools, b.IncludedTools) && slices.Equal(a.IncludedServers, b.IncludedServers) && slices.Equal(a.ExcludedTools, b.ExcludedTools)
}

// reconcileUsers syncs user accounts based on config files in the users directory.
func (s *Service) reconcileUsers() error {
	if !s.opts.EnableEnterpriseEntitySync {
		return nil
	}

	// load the desired state of all users
	userDir := filepath.Join(s.opts.Dir, types.ConfigSyncUsersDirName)
	desired, blocked, parseErrs := configfiles.LoadDesired[types.UserConfig](userDir, func(i types.UserConfig) string { return i.Username })

	// load the current state of all users from db
	managed, err := s.loadManaged(model.EntityTypeUser)
	if err != nil {
		return err
	}

	var errs []error
	errs = append(errs, parseErrs...)

	// for each desired user config, ensure the corresponding user exists with the correct attributes
	// also track the desired state config if not already tracked
	for name, d := range desired {
		accessToken, err := accesstoken.Resolve(accesstoken.Input{
			Inline: d.Entity.AccessToken,
			Ref:    d.Entity.AccessTokenRef,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to resolve access token in %s: %w", d.Path, err))
			continue
		}
		if accessToken == "" {
			errs = append(errs, fmt.Errorf("user config %s must provide access token or access_token_ref", d.Path))
			continue
		}

		existing, err := s.services.UserService.GetUser(name)
		if err != nil {
			if !errors.Is(err, user.ErrUserNotFound) {
				// unexpected error
				errs = append(errs, fmt.Errorf("failed to fetch user %s: %w", name, err))
				continue
			}

			// user doesn't exist in db, create one
			logDesiredConfigChange("user", "create", name, d.Path)
			newUser := &model.User{Username: name, AccessToken: accessToken}
			createdUser, err := s.services.UserService.CreateUser(newUser)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to create user %s: %w", name, err))
				continue
			}
			existing = createdUser
		}

		// by this point, the user definitely exists. check for updates to any attributes.
		if existing.Role == types.UserRoleAdmin {
			errs = append(errs, fmt.Errorf("config sync cannot manage admin user %s (file: %s)", name, d.Path))
			continue
		}
		if existing.AccessToken != accessToken {
			logDesiredConfigChange("user", "update", name, d.Path)
			existing.AccessToken = accessToken
			_, err := s.services.UserService.UpdateUser(existing)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to update user %s: %w", name, err))
				continue
			}
		}

		if err := s.createOrUpdateManagedRow(model.EntityTypeUser, name, d.Path, d.Hash); err != nil {
			errs = append(errs, fmt.Errorf("failed to track user %s: %w", name, err))
		}
	}

	// for each tracked user config, if the corresponding config file is now missing and there are no errors
	// blocking deletion, delete the user and stop tracking it
	for name, trackedRow := range managed {
		if _, ok := desired[name]; ok {
			continue
		}
		if blocked[trackedRow.FilePath] || fileExists(trackedRow.FilePath) {
			continue
		}

		// the user is no longer desired at this point
		managedUser, err := s.services.UserService.GetUser(name)
		if err != nil {
			if errors.Is(err, user.ErrUserNotFound) {
				// if the user is already gone, stop tracking its config in the system as well
				_ = s.deleteManagedRow(model.EntityTypeUser, name)
				continue
			}
			errs = append(errs, fmt.Errorf("failed to fetch managed user %s during deletion: %w", name, err))
			continue
		}

		if managedUser.Role == types.UserRoleAdmin {
			errs = append(errs, fmt.Errorf("config sync cannot delete admin user %s", name))
			continue
		}

		// delete the user and stop tracking its config
		logDesiredConfigChange("user", "delete", name, trackedRow.FilePath)
		if err := s.services.UserService.DeleteUser(name); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete managed user %s after file removal: %w", name, err))
			continue
		}
		if err := s.deleteManagedRow(model.EntityTypeUser, name); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove tracking for user %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// fileExists checks if a file exists at the given path. It returns false if the path is empty or only whitespace.
func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
