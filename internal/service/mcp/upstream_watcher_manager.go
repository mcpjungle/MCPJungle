package mcp

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// upstreamWatcherSyncFunc refreshes MCPJungle's cached view of a server's tools
// using an already-initialized watcher client connection.
type upstreamWatcherSyncFunc func(context.Context, string, *client.Client) error

// upstreamWatcherClientFactory creates the long-lived client used purely for
// background upstream notifications.
type upstreamWatcherClientFactory func(context.Context, *gorm.DB, *model.McpServer, int) (*client.Client, error)

// UpstreamWatcherManagerConfig provides the dependencies required to create
// background watcher connections for supported upstream servers.
type UpstreamWatcherManagerConfig struct {
	DB *gorm.DB

	InitReqTimeoutSec int
}

// upstreamToolWatcher stores the lifecycle state for one server-specific
// background listener.
//
// syncCh is intentionally buffered to 1 so a burst of identical upstream
// notifications collapses into a single pending resync.
type upstreamToolWatcher struct {
	serverName string
	client     *client.Client
	syncCh     chan struct{}
	done       chan struct{}
}

// UpstreamWatcherManager owns long-lived watcher connections that listen for
// upstream notifications such as notifications/tools/list_changed.
//
// These watcher connections are separate from normal request execution and from
// stateful tool-call sessions. Their only purpose is to keep MCPJungle's cached
// tool list fresh in the background.
type UpstreamWatcherManager struct {
	db                *gorm.DB
	initReqTimeoutSec int

	mu       sync.Mutex
	watchers map[string]*upstreamToolWatcher

	syncFunc    upstreamWatcherSyncFunc
	createWatch upstreamWatcherClientFactory
}

// NewUpstreamWatcherManager constructs the watcher manager with the default
// streamable-http watcher client factory.
func NewUpstreamWatcherManager(cfg *UpstreamWatcherManagerConfig) *UpstreamWatcherManager {
	return &UpstreamWatcherManager{
		db:                cfg.DB,
		initReqTimeoutSec: cfg.InitReqTimeoutSec,
		watchers:          make(map[string]*upstreamToolWatcher),
		createWatch: func(ctx context.Context, db *gorm.DB, s *model.McpServer, initReqTimeoutSec int) (*client.Client, error) {
			return createHTTPMcpServerWatcherConn(ctx, db, s, initReqTimeoutSec, true)
		},
	}
}

// startRegisteredUpstreamWatchers restores watcher connections for all servers
// already present in the DB when MCPService starts up.
func (m *MCPService) startRegisteredUpstreamWatchers() {
	servers, err := m.ListMcpServers()
	if err != nil {
		log.Printf("[MCPService] failed to list registered servers while starting upstream watchers: %v", err)
		return
	}

	for i := range servers {
		if err := m.upstreamWatcherManager.StartWatching(&servers[i]); err != nil {
			log.Printf("[MCPService] failed to start upstream watcher for server '%s': %v", servers[i].Name, err)
		}
	}
}

// StartWatching creates a watcher connection for a supported upstream server if
// one is not already running.
//
// At the moment, push-based tool sync is supported only for
// streamable_http upstream servers that advertise tools.listChanged.
func (m *UpstreamWatcherManager) StartWatching(server *model.McpServer) error {
	if server == nil || server.Transport != types.TransportStreamableHTTP {
		return nil
	}

	m.mu.Lock()
	if _, exists := m.watchers[server.Name]; exists {
		// A watcher already exists for this server, so no new connection is needed.
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// a watcher doesn't already exist for this server, create a new one
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.initReqTimeoutSec)*time.Second)
	defer cancel()

	c, err := m.createWatch(ctx, m.db, server, m.initReqTimeoutSec)
	if err != nil {
		return err
	}

	caps := c.GetServerCapabilities()
	if caps.Tools == nil || !caps.Tools.ListChanged {
		if err := c.Close(); err != nil {
			log.Printf("[UpstreamWatcherManager] failed to close unused watcher connection for server '%s': %v", server.Name, err)
		}
		return nil
	}

	watcher := &upstreamToolWatcher{
		serverName: server.Name,
		client:     c,
		syncCh:     make(chan struct{}, 1),
		done:       make(chan struct{}),
	}

	c.OnNotification(func(notification mcp.JSONRPCNotification) {
		if notification.Method != mcp.MethodNotificationToolsListChanged {
			return
		}
		watcher.enqueueSync()
	})

	m.mu.Lock()
	if _, exists := m.watchers[server.Name]; exists {
		// A watcher was created for this server while we were setting up the new one, so discard the new connection.
		m.mu.Unlock()
		_ = c.Close()
		return nil
	}
	m.watchers[server.Name] = watcher
	m.mu.Unlock()

	go m.runWatcher(watcher)
	log.Printf("[UpstreamWatcherManager] listening for tool list changes from upstream server '%s'", server.Name)

	return nil
}

// runWatcher serializes tool-sync work for a single upstream watcher. Multiple
// upstream notifications may arrive quickly, but only one sync runs at a time
// for a given server.
func (m *UpstreamWatcherManager) runWatcher(watcher *upstreamToolWatcher) {
	for {
		select {
		case <-watcher.done:
			return
		case <-watcher.syncCh:
			if m.syncFunc == nil {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.initReqTimeoutSec)*time.Second)
			err := m.syncFunc(ctx, watcher.serverName, watcher.client)
			cancel()
			if err != nil {
				log.Printf("[UpstreamWatcherManager] failed to sync tools for server '%s' after upstream notification: %v", watcher.serverName, err)
			}
		}
	}
}

// StopWatching terminates the watcher connection for a server and removes it
// from the manager's active set.
func (m *UpstreamWatcherManager) StopWatching(serverName string) {
	m.mu.Lock()
	watcher, exists := m.watchers[serverName]
	if exists {
		delete(m.watchers, serverName)
	}
	m.mu.Unlock()

	if !exists {
		return
	}

	close(watcher.done)
	if err := watcher.client.Close(); err != nil {
		log.Printf("[UpstreamWatcherManager] failed to close watcher for server '%s': %v", serverName, err)
	}
}

// Shutdown stops every active watcher. This is called during MCPService
// shutdown so background listener connections do not outlive the process.
func (m *UpstreamWatcherManager) Shutdown() {
	m.mu.Lock()
	names := make([]string, 0, len(m.watchers))
	for name := range m.watchers {
		names = append(names, name)
	}
	m.mu.Unlock()

	for _, name := range names {
		m.StopWatching(name)
	}
}

// enqueueSync requests a background resync without blocking the notification
// handler. If a sync is already queued, the new notification is coalesced.
func (w *upstreamToolWatcher) enqueueSync() {
	select {
	case <-w.done:
		return
	default:
	}

	select {
	case w.syncCh <- struct{}{}:
	default:
	}
}
