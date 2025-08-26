package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mcpjungle/mcpjungle/internal/metrics"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

const serverToolNameSep = "::"

// splitServerToolName splits the unique tool name into server name and tool name.
func splitServerToolName(name string) (string, string, bool) {
	return strings.Cut(name, serverToolNameSep)
}

// listToolsHandler handles tool listing requests
func (s *Server) listToolsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		server := c.Query("server")
		var (
			tools []model.Tool
			err   error
		)
		if server == "" {
			// no server specified, list all tools
			tools, err = s.mcpService.ListTools()
		} else {
			// server specified, list tools for that server
			tools, err = s.mcpService.ListToolsByServer(server)
		}
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "list_tools", metrics.StatusError, started, err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Context(), "list_tools", metrics.StatusSuccess, started, nil)
		}
		c.JSON(http.StatusOK, tools)
	}
}

// invokeToolHandler handles tool invocation requests
func (s *Server) invokeToolHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		var args map[string]any
		if err := json.NewDecoder(c.Request.Body).Decode(&args); err != nil {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "call_tool", metrics.StatusError, started, err)
			}
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "failed to decode request body: " + err.Error()},
			)
			return
		}

		rawName, ok := args["name"]
		if !ok {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "call_tool", metrics.StatusError, started, nil)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'name' field in request body"})
			return
		}
		name, ok := rawName.(string)
		if !ok {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "call_tool", metrics.StatusError, started, nil)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "'name' field must be a string"})
			return
		}

		// remove name from args since it was an input for the api, not for the tool
		delete(args, "name")

		// Extract server and tool names for metrics
		serverName, toolName, _ := splitServerToolName(name)

		resp, err := s.mcpService.InvokeTool(c, name, args)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "call_tool", metrics.StatusError, started, err)
				s.metrics.RecordTool(c.Request.Context(), serverName, toolName, metrics.StatusError, started, err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to invoke tool: " + err.Error()})
			return
		}

		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Context(), "call_tool", metrics.StatusSuccess, started, nil)
			s.metrics.RecordTool(c.Request.Context(), serverName, toolName, metrics.StatusSuccess, started, nil)
		}

		c.JSON(http.StatusOK, resp)
	}
}

// enableToolsHandler handles tool enabling requests
func (s *Server) enableToolsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		entity := c.Query("entity")
		if entity == "" {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "enable_tools", metrics.StatusError, started, nil)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'entity' query parameter"})
			return
		}
		enabledTools, err := s.mcpService.EnableTools(entity)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "enable_tools", metrics.StatusError, started, err)
				s.metrics.RecordEnhancedError(c.Request.Context(), metrics.ErrorTypeValidation)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable tool(s): " + err.Error()})
			return
		}

		// Record tool availability changes
		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Context(), "enable_tools", metrics.StatusSuccess, started, nil)
			for _, toolName := range enabledTools {
				s.metrics.RecordToolAvailability(c.Request.Context(), toolName, true)
			}
		}
		c.JSON(http.StatusOK, gin.H{"enabled": enabledTools})
	}
}

// disableToolsHandler handles tool disabling requests
func (s *Server) disableToolsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		entity := c.Query("entity")
		if entity == "" {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "disable_tools", metrics.StatusError, started, nil)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'entity' query parameter"})
			return
		}
		disabledTools, err := s.mcpService.DisableTools(entity)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "disable_tools", metrics.StatusError, started, err)
				s.metrics.RecordEnhancedError(c.Request.Context(), metrics.ErrorTypeValidation)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable tool(s): " + err.Error()})
			return
		}

		// Record tool availability changes
		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Context(), "disable_tools", metrics.StatusSuccess, started, nil)
			for _, toolName := range disabledTools {
				s.metrics.RecordToolAvailability(c.Request.Context(), toolName, false)
			}
		}
		c.JSON(http.StatusOK, gin.H{"disabled": disabledTools})
	}
}

// getToolHandler handles individual tool retrieval requests
func (s *Server) getToolHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()

		toolName := c.Query("name")
		if toolName == "" {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "get_tool", metrics.StatusError, started, nil)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'name' query parameter"})
			return
		}

		tool, err := s.mcpService.GetTool(toolName)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordRequest(c.Request.Context(), "get_tool", metrics.StatusError, started, err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if s.metrics != nil {
			s.metrics.RecordRequest(c.Request.Context(), "get_tool", metrics.StatusSuccess, started, nil)
		}
		c.JSON(http.StatusOK, tool)
	}
}
