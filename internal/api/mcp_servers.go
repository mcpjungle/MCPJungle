package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func registerServerHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		var server *model.McpServer
		if transport == types.TransportStreamableHTTP {
			server, err = model.NewStreamableHTTPServer(
				input.Name,
				input.Description,
				input.URL,
				input.BearerToken,
			)
			if err != nil {
				c.JSON(
					http.StatusBadRequest,
					gin.H{"error": fmt.Sprintf("Error creating streamable http server: %v", err)},
				)
				return
			}
		} else {
			server, err = model.NewStdioServer(
				input.Name,
				input.Description,
				input.Command,
				input.Args,
				input.Env,
			)
			if err != nil {
				c.JSON(
					http.StatusBadRequest,
					gin.H{"error": fmt.Sprintf("Error creating stdio server: %v", err)},
				)
				return
			}
		}

		if err := mcpService.RegisterMcpServer(c, server); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, server)
	}
}

func deregisterServerHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := mcpService.DeregisterMcpServer(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func listServersHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		records, err := mcpService.ListMcpServers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		servers := make([]*types.McpServer, len(records))
		for i, record := range records {
			servers[i] = &types.McpServer{
				Name:        record.Name,
				Transport:   string(record.Transport),
				Description: record.Description,
			}
			if record.Transport == types.TransportStreamableHTTP {
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
			} else {
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
			}
		}
		c.JSON(http.StatusOK, servers)
	}
}
