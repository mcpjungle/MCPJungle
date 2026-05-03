package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
)

func (s *Server) listMcpClientsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clients, err := s.mcpClientService.ListClients()
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, clients)
	}
}

func (s *Server) createMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.McpClient
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		// TODO: if allow list in the request is null, convert it to an empty JSON array
		client, err := s.mcpClientService.CreateClient(req)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, client)
	}
}

// createSelfClientHandler lets any authenticated user create a personal MCP client token
// without needing admin privileges. The client name is scoped to the user's username.
func (s *Server) createSelfClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		// name is optional — default to "<username>-mcp"
		_ = c.ShouldBindJSON(&req)

		u, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}
		userModel := u.(*model.User)
		username := userModel.Username
		name := req.Name
		if name == "" {
			name = fmt.Sprintf("%s-mcp", username)
		}

		// Inherit the allow_list configured by admin for this user.
		// If the user has no explicit allow_list, fall back to wildcard.
		allowList := userModel.AllowList
		if len(allowList) == 0 {
			allowList, _ = json.Marshal([]string{"*"})
		}
		client, err := s.mcpClientService.CreateClient(model.McpClient{
			Name:      name,
			AllowList: allowList,
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, client)
	}
}

// applySelfClientConfigHandler runs scripts/setup-mcp-clients.sh for the requesting user's
// chosen IDE targets, injecting the provided MCP client token automatically.
func (s *Server) applySelfClientConfigHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if runtime.GOOS == "windows" {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "apply-config is not supported on Windows"})
			return
		}

		var req struct {
			Token   string   `json:"token"`
			Targets []string `json:"targets"` // e.g. ["claude","cursor","codex","copilot","opencode","zed"]
			Host    string   `json:"host"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}
		if len(req.Targets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one target is required"})
			return
		}

		// Locate the setup script relative to the binary's working directory or
		// fall back to the directory of the running executable.
		scriptCandidates := []string{
			"scripts/setup-mcp-clients.sh",
			filepath.Join(filepath.Dir(os.Args[0]), "scripts/setup-mcp-clients.sh"),
		}
		script := ""
		for _, candidate := range scriptCandidates {
			if _, err := os.Stat(candidate); err == nil {
				script = candidate
				break
			}
		}
		if script == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "setup-mcp-clients.sh not found"})
			return
		}

		host := req.Host
		if host == "" {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			host = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
		}

		args := []string{script, "--token", req.Token, "--host", host}
		valid := map[string]bool{
			"claude": true, "cursor": true, "codex": true,
			"copilot": true, "opencode": true, "zed": true,
		}
		for _, t := range req.Targets {
			if valid[t] {
				args = append(args, "--"+t)
			}
		}

		cmd := exec.Command("bash", args...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		output := strings.TrimSpace(string(out))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "script failed", "output": output})
			return
		}
		c.JSON(http.StatusOK, gin.H{"output": output})
	}
}

func (s *Server) deleteMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := s.mcpClientService.DeleteClient(name); err != nil {
			handleServiceError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) updateMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		var req struct {
			Description       string   `json:"description"`
			AllowList         []string `json:"allow_list"`
			AccessToken       string   `json:"access_token"`
			RotateAccessToken bool     `json:"rotate_access_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		resp, err := s.mcpClientService.UpdateClient(mcpclient.UpdateClientInput{
			Name:              name,
			Description:       req.Description,
			AllowList:         req.AllowList,
			AccessToken:       req.AccessToken,
			RotateAccessToken: req.RotateAccessToken,
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
