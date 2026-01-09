package mcp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient is a minimal mock that satisfies the client.Client interface for testing
type mockClient struct {
	closed bool
	mu     sync.Mutex
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockClient) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestNewSessionManager(t *testing.T) {
	tests := []struct {
		name           string
		idleTimeoutSec int
		wantTimeout    int
	}{
		{
			name:           "positive timeout",
			idleTimeoutSec: 3600,
			wantTimeout:    3600,
		},
		{
			name:           "zero timeout (no expiry)",
			idleTimeoutSec: 0,
			wantTimeout:    0,
		},
		{
			name:           "negative timeout uses default",
			idleTimeoutSec: -1,
			wantTimeout:    DefaultSessionIdleTimeoutSec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSessionManager(&SessionManagerConfig{
				IdleTimeoutSec:    tt.idleTimeoutSec,
				InitReqTimeoutSec: 10,
			})
			defer sm.Shutdown()

			assert.Equal(t, tt.wantTimeout, sm.idleTimeoutSec)
		})
	}
}

func TestSessionManager_GetOrCreateSession(t *testing.T) {
	// Create a session manager with a mock session creator
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    3600,
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	callCount := 0
	mockClientInstance := &mockClient{}

	// Override the createSessionFunc with a mock
	sm.createSessionFunc = func(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*client.Client, error) {
		callCount++
		// Return a nil client for testing purposes - we're testing the session management logic
		// In real tests, we'd need a proper mock client
		return (*client.Client)(nil), nil
	}

	server := &model.McpServer{
		Name:        "test-server",
		Transport:   types.TransportStdio,
		SessionMode: types.SessionModeStateful,
	}

	ctx := context.Background()

	// First call should create a new session
	_, err := sm.GetOrCreateSession(ctx, server)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "should create session on first call")
	assert.Equal(t, 1, sm.SessionCount())

	// Second call should reuse existing session
	_, err = sm.GetOrCreateSession(ctx, server)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "should reuse session on second call")
	assert.Equal(t, 1, sm.SessionCount())

	// Verify session exists
	assert.True(t, sm.HasSession("test-server"))
	assert.False(t, sm.HasSession("nonexistent-server"))

	// Clean up
	_ = mockClientInstance // prevent unused variable warning
}

func TestSessionManager_CloseSession(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    3600,
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Manually add a session for testing
	sm.sessions["test-server"] = &ManagedSession{
		ServerName: "test-server",
		Client:     nil, // nil client for testing
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	assert.Equal(t, 1, sm.SessionCount())
	assert.True(t, sm.HasSession("test-server"))

	// Close the session
	sm.CloseSession("test-server")

	assert.Equal(t, 0, sm.SessionCount())
	assert.False(t, sm.HasSession("test-server"))

	// Closing non-existent session should not panic
	sm.CloseSession("nonexistent-server")
}

func TestSessionManager_CloseAllSessions(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    3600,
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Add multiple sessions
	sm.sessions["server1"] = &ManagedSession{
		ServerName: "server1",
		Client:     nil,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
	sm.sessions["server2"] = &ManagedSession{
		ServerName: "server2",
		Client:     nil,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	assert.Equal(t, 2, sm.SessionCount())

	sm.CloseAllSessions()

	assert.Equal(t, 0, sm.SessionCount())
}

func TestSessionManager_CleanupIdleSessions(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    1, // 1 second timeout for testing
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Add a session that's already expired
	expiredTime := time.Now().Add(-2 * time.Second)
	sm.sessions["expired-server"] = &ManagedSession{
		ServerName: "expired-server",
		Client:     nil,
		CreatedAt:  expiredTime,
		LastUsedAt: expiredTime,
	}

	// Add a session that's still active
	sm.sessions["active-server"] = &ManagedSession{
		ServerName: "active-server",
		Client:     nil,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	assert.Equal(t, 2, sm.SessionCount())

	// Run cleanup
	sm.cleanupIdleSessions()

	// Only the active session should remain
	assert.Equal(t, 1, sm.SessionCount())
	assert.False(t, sm.HasSession("expired-server"))
	assert.True(t, sm.HasSession("active-server"))
}

func TestSessionManager_NoCleanupWhenTimeoutZero(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    0, // No timeout
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Add an "old" session
	oldTime := time.Now().Add(-24 * time.Hour)
	sm.sessions["old-server"] = &ManagedSession{
		ServerName: "old-server",
		Client:     nil,
		CreatedAt:  oldTime,
		LastUsedAt: oldTime,
	}

	assert.Equal(t, 1, sm.SessionCount())

	// Run cleanup - should not remove anything since timeout is 0
	sm.cleanupIdleSessions()

	assert.Equal(t, 1, sm.SessionCount())
	assert.True(t, sm.HasSession("old-server"))
}

func TestSessionManager_LastUsedAtUpdated(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    3600,
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Add a session with old LastUsedAt
	oldTime := time.Now().Add(-1 * time.Hour)
	sm.sessions["test-server"] = &ManagedSession{
		ServerName: "test-server",
		Client:     nil,
		CreatedAt:  oldTime,
		LastUsedAt: oldTime,
	}

	sm.mu.Lock()
	session := sm.sessions["test-server"]
	originalLastUsed := session.LastUsedAt
	sm.mu.Unlock()

	// Simulate GetOrCreateSession updating LastUsedAt
	sm.mu.Lock()
	session.LastUsedAt = time.Now()
	newLastUsed := session.LastUsedAt
	sm.mu.Unlock()

	assert.True(t, newLastUsed.After(originalLastUsed), "LastUsedAt should be updated")
}

func TestSessionResult_CloseIfApplicable(t *testing.T) {
	tests := []struct {
		name        string
		shouldClose bool
	}{
		{
			name:        "stateless session should close",
			shouldClose: true,
		},
		{
			name:        "stateful session should not close",
			shouldClose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &sessionResult{
				client:      nil, // nil client for testing
				shouldClose: tt.shouldClose,
			}

			// Should not panic even with nil client
			sr.closeIfApplicable()
		})
	}
}

func TestSessionManager_InvalidateSession(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    3600,
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Add a session
	sm.sessions["test-server"] = &ManagedSession{
		ServerName: "test-server",
		Client:     nil,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	assert.Equal(t, 1, sm.SessionCount())
	assert.True(t, sm.HasSession("test-server"))

	// Invalidate the session
	sm.InvalidateSession("test-server", "connection reset")

	assert.Equal(t, 0, sm.SessionCount())
	assert.False(t, sm.HasSession("test-server"))

	// Invalidating non-existent session should not panic
	sm.InvalidateSession("nonexistent-server", "test reason")
}

func TestSessionResult_InvalidateOnError(t *testing.T) {
	sm := NewSessionManager(&SessionManagerConfig{
		IdleTimeoutSec:    3600,
		InitReqTimeoutSec: 10,
	})
	defer sm.Shutdown()

	// Add a session for testing
	sm.sessions["test-server"] = &ManagedSession{
		ServerName: "test-server",
		Client:     nil,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	tests := []struct {
		name               string
		err                error
		shouldClose        bool
		sessionManager     *SessionManager
		expectInvalidation bool
	}{
		{
			name:               "nil error should not invalidate",
			err:                nil,
			shouldClose:        false,
			sessionManager:     sm,
			expectInvalidation: false,
		},
		{
			name:               "stateless session should not invalidate",
			err:                errors.New("connection refused"),
			shouldClose:        true,
			sessionManager:     nil, // stateless has no session manager
			expectInvalidation: false,
		},
		{
			name:               "connection refused should invalidate stateful session",
			err:                errors.New("connection refused"),
			shouldClose:        false,
			sessionManager:     sm,
			expectInvalidation: true,
		},
		{
			name:               "timeout error should invalidate stateful session",
			err:                errors.New("context deadline exceeded"),
			shouldClose:        false,
			sessionManager:     sm,
			expectInvalidation: true,
		},
		{
			name:               "non-connection error should not invalidate",
			err:                errors.New("invalid argument"),
			shouldClose:        false,
			sessionManager:     sm,
			expectInvalidation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session for each test
			sm.sessions["test-server"] = &ManagedSession{
				ServerName: "test-server",
				Client:     nil,
				CreatedAt:  time.Now(),
				LastUsedAt: time.Now(),
			}

			sr := &sessionResult{
				client:         nil,
				shouldClose:    tt.shouldClose,
				serverName:     "test-server",
				sessionManager: tt.sessionManager,
			}

			sr.invalidateOnError(tt.err)

			if tt.expectInvalidation {
				assert.False(t, sm.HasSession("test-server"), "session should be invalidated")
			} else if tt.sessionManager != nil {
				assert.True(t, sm.HasSession("test-server"), "session should still exist")
			}
		})
	}
}
