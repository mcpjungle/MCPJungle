package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) registerServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		force, err := parseForceQueryParam(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var input types.RegisterServerInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		transport, err := types.ValidateTransport(input.Transport)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sessionMode, err := types.ValidateSessionMode(input.SessionMode)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var server *model.McpServer

		switch transport {
		case types.TransportStreamableHTTP:
			server, err = model.NewStreamableHTTPServer(
				input.Name,
				input.Description,
				input.URL,
				input.BearerToken,
				input.Headers,
				sessionMode,
			)
			if err != nil {
				c.JSON(
					http.StatusBadRequest,
					gin.H{"error": fmt.Sprintf("Error creating streamable http server: %v", err)},
				)
				return
			}
		case types.TransportStdio:
			server, err = model.NewStdioServer(
				input.Name,
				input.Description,
				input.Command,
				input.Args,
				input.Env,
				sessionMode,
			)
			if err != nil {
				c.JSON(
					http.StatusBadRequest,
					gin.H{"error": fmt.Sprintf("Error creating stdio server: %v", err)},
				)
				return
			}
		default:
			// transport is SSE
			server, err = model.NewSSEServer(
				input.Name,
				input.Description,
				input.URL,
				input.BearerToken,
				sessionMode,
			)
			if err != nil {
				c.JSON(
					http.StatusBadRequest,
					gin.H{"error": fmt.Sprintf("Error creating SSE server: %v", err)},
				)
				return
			}
		}

		if force {
			// If "force" option is set, we check if a server with the same name already exists. If it does, we deregister it before registering the new one.
			if _, err := s.mcpService.GetMcpServer(input.Name); err == nil {
				log.Printf("[INFO] force=true: deregistering existing MCP server %s before re-registration", input.Name)
				if err := s.mcpService.DeregisterMcpServer(input.Name); err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{"error": fmt.Sprintf("Error deregistering existing server with name %s: %v", input.Name, err)},
					)
					return
				}
			} else if !errors.Is(err, apierrors.ErrNotFound) {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": fmt.Sprintf("Error checking for existing server with name %s: %v", input.Name, err)},
				)
				return
			}
		}

		if err := s.mcpService.RegisterMcpServer(c, server); err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusCreated, server)
	}
}

func parseForceQueryParam(c *gin.Context) (bool, error) {
	if c.Query("force") == "" {
		return false, nil
	}

	force, err := strconv.ParseBool(c.Query("force"))
	if err != nil {
		return false, fmt.Errorf("invalid force query parameter: %w", err)
	}

	return force, nil
}

func (s *Server) deregisterServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")

		if err := s.mcpService.DeregisterMcpServer(name); err != nil {
			handleServiceError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) listServersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		records, err := s.mcpService.ListMcpServers()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		servers := make([]*types.McpServer, len(records))

		for i, record := range records {
			servers[i] = &types.McpServer{
				Name:        record.Name,
				Transport:   string(record.Transport),
				Description: record.Description,
				SessionMode: string(record.SessionMode),
			}

			switch record.Transport {
			case types.TransportStreamableHTTP:
				conf, err := record.GetStreamableHTTPConfig()
				if err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{
							"error": fmt.Sprintf("Error getting streamable HTTP config for server %s: %v", record.Name, err),
						},
					)
					return
				}
				servers[i].URL = conf.URL
			case types.TransportStdio:
				conf, err := record.GetStdioConfig()
				if err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{
							"error": fmt.Sprintf("Error getting stdio config for server %s: %v", record.Name, err),
						},
					)
					return
				}
				servers[i].Command = conf.Command
				servers[i].Args = conf.Args
				servers[i].Env = conf.Env
			default:
				// transport is SSE
				conf, err := record.GetSSEConfig()
				if err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{
							"error": fmt.Sprintf("Error getting SSE config for server %s: %v", record.Name, err),
						},
					)
					return
				}
				servers[i].URL = conf.URL
			}
		}

		c.JSON(http.StatusOK, servers)
	}
}

func (s *Server) enableServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")

		tools, prompts, err := s.mcpService.EnableMcpServer(name)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		result := types.EnableDisableServerResult{
			Name:            name,
			ToolsAffected:   tools,
			PromptsAffected: prompts,
		}
		c.JSON(http.StatusOK, result)
	}
}

func (s *Server) disableServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")

		tools, prompts, err := s.mcpService.DisableMcpServer(name)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		result := types.EnableDisableServerResult{
			Name:            name,
			ToolsAffected:   tools,
			PromptsAffected: prompts,
		}
		c.JSON(http.StatusOK, result)
	}
}

// getServerConfigsHandler returns the configurations of all registered MCP servers.
// This is different from listServersHandler because it returns the complete configuration of each server
// used to register them, including potentially sensitive information.
// The configs can be used to register the servers again elsewhere.
func (s *Server) getServerConfigsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		records, err := s.mcpService.ListMcpServers()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		servers := make([]*types.RegisterServerInput, len(records))

		for i, record := range records {
			servers[i] = &types.RegisterServerInput{
				Name:        record.Name,
				Transport:   string(record.Transport),
				Description: record.Description,
				SessionMode: string(record.SessionMode),
			}

			switch record.Transport {
			case types.TransportStreamableHTTP:
				conf, err := record.GetStreamableHTTPConfig()
				if err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{
							"error": fmt.Sprintf("Error getting streamable HTTP config for server %s: %v", record.Name, err),
						},
					)
					return
				}
				servers[i].URL = conf.URL
				servers[i].BearerToken = conf.BearerToken
				servers[i].Headers = conf.Headers
			case types.TransportStdio:
				conf, err := record.GetStdioConfig()
				if err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{
							"error": fmt.Sprintf("Error getting stdio config for server %s: %v", record.Name, err),
						},
					)
					return
				}
				servers[i].Command = conf.Command
				servers[i].Args = conf.Args
				servers[i].Env = conf.Env
			default:
				// transport is SSE
				conf, err := record.GetSSEConfig()
				if err != nil {
					c.JSON(
						http.StatusInternalServerError,
						gin.H{
							"error": fmt.Sprintf("Error getting SSE config for server %s: %v", record.Name, err),
						},
					)
					return
				}
				servers[i].URL = conf.URL
				servers[i].BearerToken = conf.BearerToken
			}
		}

		c.JSON(http.StatusOK, servers)
	}
}
