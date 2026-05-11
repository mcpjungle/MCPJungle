package api

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

const dashboardOAuthResultRetention = 10 * time.Minute

func (s *Server) dashboardOAuthCallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")
		oauthError := c.Query("error")
		errorDescription := c.Query("error_description")

		if oauthError != "" {
			s.tryMarkDashboardOAuthFailed(c, state, oauthError, errorDescription)
			renderDashboardOAuthHTML(c, http.StatusBadRequest, "Authorization failed", safeOAuthErrorMessage(oauthError, errorDescription))
			return
		}
		if code == "" || state == "" {
			renderDashboardOAuthHTML(c, http.StatusBadRequest, "Authorization failed", "Missing required OAuth callback parameters.")
			return
		}

		session, err := s.mcpService.GetPendingUpstreamOAuthSessionByState(c, state)
		if err != nil {
			renderDashboardOAuthHTML(c, dashboardOAuthErrorStatus(err), "Authorization failed", safeOAuthCallbackError(err))
			return
		}

		_, err = s.mcpService.CompleteUpstreamOAuthSession(c, session.SessionID, code, state)
		if err != nil {
			s.storeDashboardOAuthResult(session.SessionID, dashboardOAuthSessionResult{
				Status:     dashboardOAuthStatusForError(err),
				Error:      safeOAuthCallbackError(err),
				ServerName: session.ServerName,
				ExpiresAt:  session.ExpiresAt,
				UpdatedAt:  time.Now(),
			})
			renderDashboardOAuthHTML(c, dashboardOAuthErrorStatus(err), "Authorization failed", safeOAuthCallbackError(err))
			return
		}

		s.storeDashboardOAuthResult(session.SessionID, dashboardOAuthSessionResult{
			Status:     "completed",
			ServerName: session.ServerName,
			ExpiresAt:  session.ExpiresAt,
			UpdatedAt:  time.Now(),
		})
		renderDashboardOAuthHTML(c, http.StatusOK, "Authorization successful", "Authorization successful. You can close this tab and return to MCPJungle.")
	}
}

func (s *Server) dashboardOAuthSessionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
			return
		}

		if result, ok := s.getDashboardOAuthResult(sessionID); ok {
			c.JSON(http.StatusOK, dashboardOAuthSessionResponse{
				SessionID:  sessionID,
				Status:     result.Status,
				ServerName: result.ServerName,
				ExpiresAt:  &result.ExpiresAt,
				Error:      result.Error,
			})
			return
		}

		session, err := s.mcpService.GetPendingUpstreamOAuthSession(c, sessionID)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		if time.Now().After(session.ExpiresAt) {
			_ = s.mcpService.DeletePendingUpstreamOAuthSession(c, session.SessionID)
			s.storeDashboardOAuthResult(session.SessionID, dashboardOAuthSessionResult{
				Status:     "expired",
				Error:      "OAuth authorization expired. Start registration again.",
				ServerName: session.ServerName,
				ExpiresAt:  session.ExpiresAt,
				UpdatedAt:  time.Now(),
			})
			c.JSON(http.StatusOK, dashboardOAuthSessionResponse{
				SessionID:  session.SessionID,
				Status:     "expired",
				ServerName: session.ServerName,
				ExpiresAt:  &session.ExpiresAt,
				Error:      "OAuth authorization expired. Start registration again.",
			})
			return
		}

		c.JSON(http.StatusOK, dashboardOAuthSessionResponse{
			SessionID:  session.SessionID,
			Status:     "pending",
			ServerName: session.ServerName,
			ExpiresAt:  &session.ExpiresAt,
		})
	}
}

func renderDashboardOAuthHTML(c *gin.Context, status int, title, message string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(status, fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, sans-serif; background: #f6f8fa; color: #1f2328; margin: 0; }
    main { max-width: 560px; margin: 64px auto; padding: 24px; background: #fff; border: 1px solid #d0d7de; border-radius: 16px; box-shadow: 0 12px 32px rgba(31,35,40,.08); }
    h1 { margin: 0 0 12px; font-size: 1.25rem; }
    p { margin: 0; line-height: 1.5; color: #57606a; }
  </style>
</head>
<body>
  <main>
    <h1>%s</h1>
    <p>%s</p>
  </main>
</body>
</html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message)))
}

func (s *Server) getDashboardOAuthResult(sessionID string) (dashboardOAuthSessionResult, bool) {
	s.dashboardOAuthMu.Lock()
	defer s.dashboardOAuthMu.Unlock()

	s.cleanupDashboardOAuthResultsLocked(time.Now())
	result, ok := s.dashboardOAuthResults[sessionID]
	return result, ok
}

func (s *Server) storeDashboardOAuthResult(sessionID string, result dashboardOAuthSessionResult) {
	s.dashboardOAuthMu.Lock()
	defer s.dashboardOAuthMu.Unlock()

	s.cleanupDashboardOAuthResultsLocked(time.Now())
	s.dashboardOAuthResults[sessionID] = result
}

func (s *Server) cleanupDashboardOAuthResultsLocked(now time.Time) {
	for sessionID, result := range s.dashboardOAuthResults {
		if !result.ExpiresAt.IsZero() && now.After(result.ExpiresAt.Add(dashboardOAuthResultRetention)) {
			delete(s.dashboardOAuthResults, sessionID)
			continue
		}
		if !result.UpdatedAt.IsZero() && now.After(result.UpdatedAt.Add(dashboardOAuthResultRetention)) {
			delete(s.dashboardOAuthResults, sessionID)
		}
	}
}

func (s *Server) tryMarkDashboardOAuthFailed(c *gin.Context, state, oauthError, errorDescription string) {
	if state == "" {
		return
	}
	session, err := s.mcpService.GetPendingUpstreamOAuthSessionByState(c, state)
	if err != nil {
		return
	}
	s.storeDashboardOAuthResult(session.SessionID, dashboardOAuthSessionResult{
		Status:     "failed",
		Error:      safeOAuthErrorMessage(oauthError, errorDescription),
		ServerName: session.ServerName,
		ExpiresAt:  session.ExpiresAt,
		UpdatedAt:  time.Now(),
	})
	_ = s.mcpService.DeletePendingUpstreamOAuthSession(c, session.SessionID)
}

func safeOAuthCallbackError(err error) string {
	switch {
	case errors.Is(err, apierrors.ErrNotFound):
		return "OAuth session was not found. Start registration again."
	case errors.Is(err, apierrors.ErrInvalidInput):
		if strings.Contains(err.Error(), "expired") {
			return "OAuth authorization expired. Start registration again."
		}
		return "OAuth authorization could not be completed. Start registration again."
	default:
		return "OAuth authorization could not be completed. Check the MCPJungle server logs for details."
	}
}

func safeOAuthErrorMessage(oauthError, errorDescription string) string {
	if errorDescription != "" {
		return fmt.Sprintf("OAuth authorization failed: %s.", errorDescription)
	}
	if oauthError != "" {
		return fmt.Sprintf("OAuth authorization failed: %s.", oauthError)
	}
	return "OAuth authorization failed."
}

func dashboardOAuthErrorStatus(err error) int {
	if errors.Is(err, apierrors.ErrNotFound) || errors.Is(err, apierrors.ErrInvalidInput) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func dashboardOAuthStatusForError(err error) string {
	if errors.Is(err, apierrors.ErrInvalidInput) && strings.Contains(err.Error(), "expired") {
		return "expired"
	}
	return "failed"
}
