package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) createUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newUser, err := s.userService.CreateUser(&input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := &types.CreateOrUpdateUserResponse{
			Username:    newUser.Username,
			Role:        string(newUser.Role),
			AccessToken: newUser.AccessToken,
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func (s *Server) listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := s.userService.ListUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := make([]*types.User, len(users))
		for i, u := range users {
			resp[i] = &types.User{
				Username: u.Username,
				Role:     string(u.Role),
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) updateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
			return
		}

		var input model.User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedUser, err := s.userService.UpdateUser(&input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := &types.CreateOrUpdateUserResponse{
			Username:    updatedUser.Username,
			Role:        string(updatedUser.Role),
			AccessToken: updatedUser.AccessToken,
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) deleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
			return
		}

		err := s.userService.DeleteUser(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) whoAmIHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		u, ok := currentUser.(*model.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user from context"})
			return
		}

		resp := types.User{
			Username: u.Username,
			Role:     string(u.Role),
		}
		c.JSON(http.StatusOK, resp)
	}
}
