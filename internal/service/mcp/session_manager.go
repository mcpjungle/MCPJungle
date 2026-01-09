package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

const (
	// DefaultSessionIdleTimeoutSec is the default idle timeout for stateful sessions in seconds.
	// Sessions that are idle for longer than this duration will be closed.
	DefaultSessionIdleTimeoutSec = 3600 // 1 hour

	// sessionCleanupIntervalSec is the interval at which the session manager checks for idle sessions.
	sessionCleanupIntervalSec = 60 // 1 minute
)

// ManagedSession represents a persistent connection to an MCP server.
type ManagedSession struct {
	ServerName string
	Client     *client.Client
	CreatedAt  time.Time
	LastUsedAt time.Time
}

// SessionManager manages persistent connections to MCP servers configured in stateful mode.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*ManagedSession // key: server name

	idleTimeoutSec    int
	initReqTimeoutSec int
	cleanupTicker     *time.Ticker
	cleanupStopChan   chan struct{}
	createSessionFunc func(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*client.Client, error)
}

// SessionManagerConfig holds configuration for the SessionManager.
type SessionManagerConfig struct {
	// IdleTimeoutSec is the idle timeout in seconds for stateful sessions.
	// Sessions idle for longer than this will be closed.
	// If set to 0, sessions will never be closed due to inactivity.
	IdleTimeoutSec int

	// InitReqTimeoutSec is the timeout for MCP server initialization requests.
	InitReqTimeoutSec int
}

// NewSessionManager creates a new SessionManager instance.
func NewSessionManager(cfg *SessionManagerConfig) *SessionManager {
	idleTimeout := cfg.IdleTimeoutSec
	if idleTimeout < 0 {
		idleTimeout = DefaultSessionIdleTimeoutSec
	}

	sm := &SessionManager{
		sessions:          make(map[string]*ManagedSession),
		idleTimeoutSec:    idleTimeout,
		initReqTimeoutSec: cfg.InitReqTimeoutSec,
		cleanupStopChan:   make(chan struct{}),
		// Use the actual session creation function by default
		createSessionFunc: createMcpServerConnection,
	}

	// Start cleanup goroutine if idle timeout is enabled
	if idleTimeout > 0 {
		sm.startCleanupRoutine()
	}

	return sm
}

// GetOrCreateSession returns an existing session for the server or creates a new one.
// This method should only be called for servers with SessionMode set to stateful.
func (sm *SessionManager) GetOrCreateSession(ctx context.Context, server *model.McpServer) (*client.Client, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if we have an existing session
	if session, exists := sm.sessions[server.Name]; exists {
		session.LastUsedAt = time.Now()
		return session.Client, nil
	}

	// Create a new session
	mcpClient, err := sm.createSessionFunc(ctx, server, sm.initReqTimeoutSec)
	if err != nil {
		return nil, fmt.Errorf("failed to create session for server '%s': %w", server.Name, err)
	}

	sm.sessions[server.Name] = &ManagedSession{
		ServerName: server.Name,
		Client:     mcpClient,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	log.Printf("[SessionManager] Created new stateful session for server '%s'", server.Name)

	return mcpClient, nil
}

// CloseSession closes and removes the session for the given server.
func (sm *SessionManager) CloseSession(serverName string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[serverName]; exists {
		if session.Client != nil {
			if err := session.Client.Close(); err != nil {
				log.Printf("[SessionManager] Error closing session for server '%s': %v", serverName, err)
			}
		}
		delete(sm.sessions, serverName)
		log.Printf("[SessionManager] Closed session for server '%s'", serverName)
	}
}

// InvalidateSession closes and removes a session due to a detected error.
// This is called reactively when a connection error is detected during a tool call.
// The next call to GetOrCreateSession will create a fresh session.
func (sm *SessionManager) InvalidateSession(serverName string, reason string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[serverName]; exists {
		if session.Client != nil {
			if err := session.Client.Close(); err != nil {
				log.Printf("[SessionManager] Error closing unhealthy session for server '%s': %v", serverName, err)
			}
		}
		delete(sm.sessions, serverName)
		log.Printf("[SessionManager] Invalidated unhealthy session for server '%s': %s", serverName, reason)
	}
}

// CloseAllSessions closes all managed sessions.
// This should be called when the MCPJungle server is shutting down.
func (sm *SessionManager) CloseAllSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for name, session := range sm.sessions {
		if session.Client != nil {
			if err := session.Client.Close(); err != nil {
				log.Printf("[SessionManager] Error closing session for server '%s': %v", name, err)
			}
		}
		delete(sm.sessions, name)
	}

	log.Printf("[SessionManager] Closed all sessions")
}

// Shutdown stops the cleanup routine and closes all sessions.
func (sm *SessionManager) Shutdown() {
	// Stop the cleanup routine
	if sm.cleanupTicker != nil {
		close(sm.cleanupStopChan)
	}

	// Close all sessions
	sm.CloseAllSessions()
}

// HasSession returns true if a session exists for the given server.
func (sm *SessionManager) HasSession(serverName string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.sessions[serverName]
	return exists
}

// SessionCount returns the number of active sessions.
func (sm *SessionManager) SessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// startCleanupRoutine starts a background goroutine that periodically cleans up idle sessions.
func (sm *SessionManager) startCleanupRoutine() {
	sm.cleanupTicker = time.NewTicker(time.Duration(sessionCleanupIntervalSec) * time.Second)

	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.cleanupIdleSessions()
			case <-sm.cleanupStopChan:
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupIdleSessions closes sessions that have been idle for longer than the idle timeout.
func (sm *SessionManager) cleanupIdleSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.idleTimeoutSec == 0 {
		return // No timeout configured
	}

	now := time.Now()
	idleThreshold := time.Duration(sm.idleTimeoutSec) * time.Second

	for name, session := range sm.sessions {
		if now.Sub(session.LastUsedAt) > idleThreshold {
			log.Printf("[SessionManager] Closing idle session for server '%s' (idle for %v)", name, now.Sub(session.LastUsedAt))
			if session.Client != nil {
				if err := session.Client.Close(); err != nil {
					log.Printf("[SessionManager] Error closing session for server '%s': %v", name, err)
				}
			}
			delete(sm.sessions, name)
		}
	}
}

// createMcpServerConnection creates a new MCP client connection based on the server's transport type.
// This is a wrapper around the transport-specific connection functions.
func createMcpServerConnection(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*client.Client, error) {
	switch s.Transport {
	case types.TransportStreamableHTTP:
		return createHTTPMcpServerConn(ctx, s, initReqTimeoutSec)
	case types.TransportSSE:
		return createSSEMcpServerConn(ctx, s)
	case types.TransportStdio:
		return runStdioServer(ctx, s, initReqTimeoutSec)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", s.Transport)
	}
}
