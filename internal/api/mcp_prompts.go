package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) listPromptsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		server := c.Query("server")
		var (
			prompts []model.Prompt
			err     error
		)
		if server == "" {
			// no server specified, list all prompts
			prompts, err = s.mcpService.ListPrompts()
		} else {
			// server specified, list prompts for that server
			prompts, err = s.mcpService.ListPromptsByServer(server)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, prompts)
	}
}

// getPromptHandler returns the prompt metadata with the given name.
func (s *Server) getPromptHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// prompt name has to be supplied as a query param because it contains double underscores.
		// cannot be supplied as a path param.
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'name' query parameter"})
			return
		}
		prompt, err := s.mcpService.GetPrompt(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get prompt: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, prompt)
	}
}

// getPromptWithArgsHandler retrieves a prompt with arguments and returns the rendered template.
func (s *Server) getPromptWithArgsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request types.PromptGetRequest
		if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "failed to decode request body: " + err.Error()},
			)
			return
		}

		if request.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'name' field in request body"})
			return
		}

		// Convert map[string]string to map[string]any for the service layer
		args := make(map[string]any)
		for k, v := range request.Arguments {
			args[k] = v
		}

		resp, err := s.mcpService.GetPromptWithArgs(c, request.Name, args)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get prompt: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// enablePromptsHandler enables the given prompt or all prompts of the given mcp server
func (s *Server) enablePromptsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := c.Query("entity")
		if entity == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'entity' query parameter"})
			return
		}
		enabledPrompts, err := s.mcpService.EnablePrompts(entity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable prompt(s): " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, enabledPrompts)
	}
}

// disablePromptsHandler disables the given prompt or all prompts of the given mcp server
func (s *Server) disablePromptsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := c.Query("entity")
		if entity == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'entity' query parameter"})
			return
		}
		disabledPrompts, err := s.mcpService.DisablePrompts(entity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable prompt(s): " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, disabledPrompts)
	}
}
